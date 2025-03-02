//go:generate go run settings_gen.go
//go:generate go run keys_gen.go

package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/rueidis"

	"github.com/bytedance/sonic"
	"github.com/robalyx/rotector/internal/common/storage/database"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"go.uber.org/zap"
)

// Session maintains user state through a Redis-backed key-value store where values are
// serialized as JSON strings. The session automatically expires after a configured timeout.
type Session struct {
	valueProcessor     *ValueProcessor
	numericProcessor   *NumericProcessor
	userSettings       *types.UserSetting
	botSettings        *types.BotSetting
	userSettingsUpdate bool
	botSettingsUpdate  bool
	db                 database.Client
	redis              rueidis.Client
	key                string
	data               map[string]any
	dataModified       map[string]bool
	mu                 sync.RWMutex
	logger             *zap.Logger
}

// NewSession creates a new session for the given user.
func NewSession(
	userSettings *types.UserSetting,
	botSettings *types.BotSetting,
	db database.Client,
	redis rueidis.Client,
	key string,
	data map[string]any,
	logger *zap.Logger,
) *Session {
	return &Session{
		valueProcessor:     NewValueProcessor(),
		numericProcessor:   NewNumericProcessor(),
		userSettings:       userSettings,
		botSettings:        botSettings,
		userSettingsUpdate: false,
		botSettingsUpdate:  false,
		db:                 db,
		redis:              redis,
		key:                key,
		data:               data,
		dataModified:       make(map[string]bool),
		logger:             logger,
	}
}

// Touch serializes the session data to JSON and updates the TTL in Redis to prevent expiration.
// If serialization fails, the error is logged but the session continues.
func (s *Session) Touch(ctx context.Context) {
	// Create a map of only persistent data
	persistentData := make(map[string]any)
	s.mu.RLock()
	for key, value := range s.data {
		isPersistent, ok := s.dataModified[key]
		if !ok || (ok && isPersistent) {
			persistentData[key] = s.valueProcessor.ProcessValue(value)
		}
	}
	s.mu.RUnlock()

	// Serialize only persistent data to JSON
	data, err := sonic.MarshalString(persistentData)
	if err != nil {
		s.logger.Error("Failed to marshal session data", zap.Error(err))
		return
	}

	// Update Redis with new data and expiration
	err = s.redis.Do(ctx,
		s.redis.B().Set().Key(s.key).Value(data).Ex(SessionTimeout).Build(),
	).Error()
	if err != nil {
		s.logger.Error("Failed to update session in Redis", zap.Error(err))
	}

	// Only save user settings if they've been updated
	if s.userSettingsUpdate {
		if err := s.db.Models().Settings().SaveUserSettings(ctx, s.userSettings); err != nil {
			s.logger.Error("Failed to save user settings", zap.Error(err))
			return
		}
		s.userSettingsUpdate = false
	}

	// Only save bot settings if they've been updated
	if s.botSettings.IsAdmin(UserID.Get(s)) && s.botSettingsUpdate {
		if err := s.db.Models().Settings().SaveBotSettings(ctx, s.botSettings); err != nil {
			s.logger.Error("Failed to save bot settings", zap.Error(err))
			return
		}
		s.botSettingsUpdate = false
	}
}

// UserSettings returns the current user settings.
func (s *Session) UserSettings() *types.UserSetting {
	return s.userSettings
}

// BotSettings returns the current bot settings.
func (s *Session) BotSettings() *types.BotSetting {
	return s.botSettings
}

// get retrieves a raw string value from the in-memory session cache.
// Returns empty string if key doesn't exist.
func (s *Session) get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if value, ok := s.data[key]; ok {
		return value
	}
	s.logger.Debug("Session key not found", zap.String("key", key))
	return nil
}

// getInterface unmarshals the stored value into the provided interface.
func (s *Session) getInterface(key string, v any) {
	value := s.get(key)
	if value == nil {
		return
	}

	// Special handling for primitive types that need string conversion
	strValue, isString := value.(string)
	if isString {
		switch typedPtr := v.(type) {
		case *uint64:
			// Handle uint64 values stored as strings
			parsedValue, err := strconv.ParseUint(strValue, 10, 64)
			if err == nil {
				*typedPtr = parsedValue
				return
			}
			s.logger.Error("Failed to parse uint64 from string",
				zap.Error(err),
				zap.String("key", key),
				zap.String("value", strValue))

		case *time.Time:
			// Handle time.Time values stored as strings
			parsedTime, err := time.Parse(time.RFC3339Nano, strValue)
			if err == nil {
				*typedPtr = parsedTime
				return
			}
			s.logger.Error("Failed to parse time.Time from string",
				zap.Error(err),
				zap.String("key", key),
				zap.String("value", strValue))
		}
	}

	// DEVELOPER NOTE:
	// This double marshal/unmarshal process is necessary to handle nested uint64 values with full precision.
	//
	// THE PROBLEM:
	// Standard JSON unmarshaling in Go treats all numbers as float64 by default. This causes precision loss
	// for uint64 values that exceed float64's exact integer representation limit (~2^53). For example, the
	// uint64 value 18446744073709551615 would lose precision if converted to float64.
	//
	// OUR SOLUTION:
	// 1. When storing in Redis: We convert uint64 values to strings with type metadata
	// 2. When retrieving simple values: The switch statement above handles direct string-to-uint64 conversion
	// 3. For complex nested structures: We need the process below
	//
	// THE PROCESS:
	// 1. Marshal the value to JSON
	// 2. Unmarshal with decoder.UseNumber() to preserve numeric precision as json.Number
	//    (This avoids automatic float64 conversion that would lose precision)
	// 3. Recursively process the structure to convert json.Number and string representations to uint64
	//    based on type metadata
	// 4. Re-marshal the processed structure with proper types
	// 5. Unmarshal again into the target type with all precision preserved

	// First marshal to JSON
	jsonBytes, err := sonic.Marshal(value)
	if err != nil {
		s.logger.Error("Failed to marshal interface",
			zap.Error(err),
			zap.String("key", key),
			zap.Any("value", value))
		return
	}

	// First unmarshal with number precision preservation
	var rawData any
	decoder := sonic.ConfigStd.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber() // This ensures numbers are decoded as json.Number to preserve precision
	if err := decoder.Decode(&rawData); err != nil {
		s.logger.Error("Failed to unmarshal to raw data",
			zap.Error(err),
			zap.String("key", key))
		return
	}

	// Process the raw data to handle uint64 conversions
	processedData := s.numericProcessor.PreserveNumericPrecision(rawData)

	// Second marshal of the processed data
	processedBytes, err := sonic.Marshal(processedData)
	if err != nil {
		s.logger.Error("Failed to marshal processed data",
			zap.Error(err),
			zap.String("key", key))
		return
	}

	// Second unmarshal into the target interface
	if err := sonic.Unmarshal(processedBytes, v); err != nil {
		s.logger.Error("Failed to unmarshal interface",
			zap.Error(err),
			zap.String("key", key),
			zap.String("json", string(processedBytes)),
			zap.String("type", fmt.Sprintf("%T", v)))
		return
	}
}

// getBuffer retrieves and decodes a base64 encoded buffer from the session.
func (s *Session) getBuffer(key string) *bytes.Buffer {
	value := s.get(key)
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	// Try to decode base64
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		s.logger.Error("Failed to decode base64 buffer",
			zap.Error(err),
			zap.String("key", key))
		return nil
	}

	return bytes.NewBuffer(decoded)
}

// set sets the value for the given key.
func (s *Session) set(key string, value any, persist bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	s.dataModified[key] = persist

	s.logger.Debug("Session key set", zap.String("key", key))
}

// setBuffer stores binary data by base64 encoding it first.
// This allows binary data to be safely stored as strings in the session.
func (s *Session) setBuffer(key string, buf *bytes.Buffer, persist bool) {
	if buf == nil {
		s.logger.Warn("Attempted to set nil buffer", zap.String("key", key))
		return
	}

	// Encode buffer to base64
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = encoded
	s.dataModified[key] = persist

	s.logger.Debug("Session key set with base64 encoded buffer", zap.String("key", key))
}

// delete removes a key from the session data.
func (s *Session) delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	s.logger.Debug("Session key deleted", zap.String("key", key))
}
