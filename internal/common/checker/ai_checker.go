package checker

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/openai/openai-go"
	"github.com/pkoukk/tiktoken-go"
	"github.com/rotector/rotector/internal/common/database"
	"github.com/rotector/rotector/internal/common/fetcher"
	"github.com/rotector/rotector/internal/common/translator"
	"github.com/rotector/rotector/internal/common/utils"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/json"
	"go.uber.org/zap"
)

//nolint:lll
const (
	// ReviewSystemPrompt provides detailed instructions to the AI model for analyzing user content.
	ReviewSystemPrompt = `You are a Roblox moderator analyzing user data for inappropriate sexual or suggestive content. Flag violations by considering explicit content, suggestive language, and combinations of phrases, symbols, and emojis. Use exact strings for 'flaggedContent'.

Calculate confidence starting at 0: +0.6 for explicit violations, +0.4 for clear suggestive content, +0.2 for subtle hints, +0.1 for each additional same-type violation, and +0.2 for each different violation type. High confidence (0.8-1.0) indicates explicit content or multiple violations, medium (0.4-0.7) shows clear patterns or coded language, and low (0.0-0.3) suggests subtle or ambiguous content.

Flag content with: explicit sexual terms, innuendos, body part references, hookup solicitation, porn references, suggestive emojis, NSFW content, ERP terms, fetish mentions, and grooming language (age questions, photo requests, off-platform chat, personal info seeking, gift offers, secret keeping). Also flag coded sexual language (number substitutions, misspellings, hidden meanings), sexualized roleplay, non-consensual references, exploitation, harassment, predatory behavior (love bombing, isolation, manipulation), suspicious requests (camera/mic usage, private games, social media), and adult industry references (OnlyFans, modeling scams, compensation offers).

Exclude non-suggestive orientation/gender identity, general friendship, non-sexual profanity, legitimate trading, and social/cultural discussions.`

	// FriendSystemPrompt provides detailed instructions to the AI model for analyzing friend networks.
	FriendSystemPrompt = `You are a content moderation assistant analyzing user friend networks for inappropriate patterns. Examine common violation themes, content severity, and network concentration. Generate a clear, short, factual 1 sentence reason highlighting the most serious violations and patterns.

Calculate confidence starting at 0: +0.6 for multiple confirmed friends with serious violations, +0.4 for multiple flagged friends with clear patterns, +0.2 for mixed confirmed/flagged friends, +0.1 for each additional same-type violation, and +0.2 for each different violation type. High confidence (0.8-1.0) indicates strong confirmed networks, medium (0.4-0.7) shows clear patterns with mixed status, and low (0.0-0.3) suggests limited connections.

Note: Leave the flaggedContent field empty.`

	FriendUserPrompt = `User: %s
Friend data: %s`

	// MaxFriendDataTokens is the maximum number of tokens allowed for friend data.
	MaxFriendDataTokens = 400
)

// Generate the JSON schema at initialization time to avoid repeated generation.
var (
	flaggedUsersSchema = utils.GenerateSchema[FlaggedUsers]() //nolint:gochecknoglobals
	flaggedUserSchema  = utils.GenerateSchema[FlaggedUser]()  //nolint:gochecknoglobals
)

// FlaggedUsers holds a list of users that the AI has identified as inappropriate.
// The JSON schema is used to ensure consistent responses from the AI.
type FlaggedUsers struct {
	Users []FlaggedUser `json:"users" jsonschema_description:"List of flagged users"`
}

// FlaggedUser contains the AI's analysis results for a single user.
// The confidence score and flagged content help moderators make decisions.
type FlaggedUser struct {
	Name           string   `json:"name"           jsonschema_description:"Exact username of the flagged user"`
	Reason         string   `json:"reason"         jsonschema_description:"Clear explanation of why the user was flagged"`
	FlaggedContent []string `json:"flaggedContent" jsonschema_description:"Exact content that was flagged"`
	Confidence     float64  `json:"confidence"     jsonschema_description:"Confidence level of the AI's assessment"`
}

// TranslationResult contains the result of translating a user's description.
type TranslationResult struct {
	UserInfo       *fetcher.Info
	TranslatedDesc string
	Err            error
}

// AIChecker handles AI-based content analysis by sending user data to OpenAI.
type AIChecker struct {
	openAIClient *openai.Client
	minify       *minify.M
	tke          *tiktoken.Tiktoken
	translator   *translator.Translator
	logger       *zap.Logger
}

// NewAIChecker creates an AIChecker with a minifier for JSON optimization,
// translator for handling non-English content, and the provided OpenAI client and logger.
func NewAIChecker(openAIClient *openai.Client, translator *translator.Translator, logger *zap.Logger) *AIChecker {
	m := minify.New()
	m.AddFunc("application/json", json.Minify)

	tke, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		logger.Fatal("Failed to get tiktoken encoding", zap.Error(err))
	}

	return &AIChecker{
		openAIClient: openAIClient,
		minify:       m,
		tke:          tke,
		translator:   translator,
		logger:       logger,
	}
}

// ProcessUsers sends user information to OpenAI for analysis after translating descriptions.
// Returns validated users and IDs of users that failed validation for retry.
// The process involves:
// 1. Translating user descriptions to proper English
// 2. Sending translated content to OpenAI for analysis
// 3. Validating AI responses against translated content
// 4. Creating validated users with original descriptions.
func (a *AIChecker) ProcessUsers(userInfos []*fetcher.Info) ([]*database.User, []uint64, error) {
	// Create a struct for user summaries for AI analysis
	type UserSummary struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	// Translate all descriptions concurrently
	translatedInfos, originalInfos := a.prepareUserInfos(userInfos)

	// Convert map to slice for OpenAI request
	userInfosWithoutID := make([]UserSummary, 0, len(translatedInfos))
	for _, userInfo := range translatedInfos {
		userInfosWithoutID = append(userInfosWithoutID, UserSummary{
			Name:        userInfo.Name,
			Description: userInfo.Description,
		})
	}

	// Minify JSON to reduce token usage
	userInfoJSON, err := sonic.Marshal(userInfosWithoutID)
	if err != nil {
		a.logger.Error("Error marshaling user info", zap.Error(err))
		return nil, nil, err
	}

	userInfoJSON, err = a.minify.Bytes("application/json", userInfoJSON)
	if err != nil {
		a.logger.Error("Error minifying user info", zap.Error(err))
		return nil, nil, err
	}

	a.logger.Info("Sending user info to AI for analysis")

	// Configure OpenAI request with schema enforcement
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("flaggedUsers"),
		Description: openai.F("List of flagged users"),
		Schema:      openai.F(flaggedUsersSchema),
		Strict:      openai.Bool(true),
	}

	// Send request to OpenAI
	chatCompletion, err := a.openAIClient.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(ReviewSystemPrompt),
			openai.UserMessage(string(userInfoJSON)),
		}),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schemaParam),
			},
		),
		Model:       openai.F(openai.ChatModelGPT4oMini2024_07_18),
		Temperature: openai.F(0.2),
	})
	if err != nil {
		a.logger.Error("Error calling OpenAI API", zap.Error(err))
		return nil, nil, err
	}

	// Parse AI response
	var flaggedUsers FlaggedUsers
	err = sonic.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &flaggedUsers)
	if err != nil {
		a.logger.Error("Error unmarshaling flagged users", zap.Error(err))
		return nil, nil, err
	}

	a.logger.Info("Received AI response",
		zap.Int("totalUsers", len(userInfos)),
		zap.Int("flaggedUsers", len(flaggedUsers.Users)))

	// Validate AI responses against translated content but use original descriptions for storage
	validatedUsers, failedValidationIDs := a.validateFlaggedUsers(flaggedUsers, translatedInfos, originalInfos)

	return validatedUsers, failedValidationIDs, nil
}

// GenerateFriendReason uses AI to analyze a user's friend list and generate a detailed reason
// for flagging based on the patterns found in their friends' reasons.
func (a *AIChecker) GenerateFriendReason(userInfo *fetcher.Info, confirmedFriends, flaggedFriends map[uint64]*database.User) (string, error) {
	// Create a summary of friend data for AI analysis
	type FriendSummary struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
		Type   string `json:"type"`
	}

	// Collect friend summaries with token counting
	friendSummaries := make([]FriendSummary, 0, len(confirmedFriends)+len(flaggedFriends))
	currentTokens := 0

	// Helper function to add friend if within token limit
	addFriend := func(friend *database.User, friendType string) bool {
		summary := FriendSummary{
			Name:   friend.Name,
			Reason: friend.Reason,
			Type:   friendType,
		}

		// Convert to JSON to count tokens accurately
		summaryJSON, err := sonic.Marshal(summary)
		if err != nil {
			a.logger.Warn("Failed to marshal friend summary",
				zap.String("username", friend.Name),
				zap.Error(err))
			return false
		}

		tokens := a.tke.Encode(string(summaryJSON), nil, nil)
		tokenCount := len(tokens)

		if currentTokens+tokenCount > MaxFriendDataTokens {
			return false
		}

		friendSummaries = append(friendSummaries, summary)
		currentTokens += tokenCount
		return true
	}

	// Add confirmed friends first (they're usually more important)
	for _, friend := range confirmedFriends {
		if !addFriend(friend, database.UserTypeConfirmed) {
			a.logger.Debug("Reached token limit while adding confirmed friends",
				zap.Int("currentTokens", currentTokens),
				zap.Int("totalConfirmed", len(confirmedFriends)))
			break
		}
	}

	// Add flagged friends if there's room
	for _, friend := range flaggedFriends {
		if !addFriend(friend, database.UserTypeFlagged) {
			a.logger.Debug("Reached token limit while adding flagged friends",
				zap.Int("currentTokens", currentTokens),
				zap.Int("totalFlagged", len(flaggedFriends)))
			break
		}
	}

	// Convert to JSON for the AI request
	friendDataJSON, err := sonic.Marshal(friendSummaries)
	if err != nil {
		return "", fmt.Errorf("failed to marshal friend data: %w", err)
	}

	// Minify JSON to reduce token usage
	friendDataJSON, err = a.minify.Bytes("application/json", friendDataJSON)
	if err != nil {
		return "", fmt.Errorf("failed to minify friend data: %w", err)
	}

	// Configure OpenAI request with schema enforcement
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("flaggedUser"),
		Description: openai.F("Flagged user information"),
		Schema:      openai.F(flaggedUserSchema),
		Strict:      openai.Bool(true),
	}

	// Send request to OpenAI
	chatCompletion, err := a.openAIClient.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(FriendSystemPrompt),
			openai.UserMessage(fmt.Sprintf(FriendUserPrompt, userInfo.Name, string(friendDataJSON))),
		}),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schemaParam),
			},
		),
		Model:       openai.F(openai.ChatModelGPT4oMini2024_07_18),
		Temperature: openai.F(0.0),
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	// Parse AI response
	var flaggedUser FlaggedUser
	err = sonic.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &flaggedUser)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal AI response: %w", err)
	}

	reason := flaggedUser.Reason
	a.logger.Debug("Generated friend network reason",
		zap.String("username", userInfo.Name),
		zap.Int("confirmedFriends", len(confirmedFriends)),
		zap.Int("flaggedFriends", len(flaggedFriends)),
		zap.Int("includedFriends", len(friendSummaries)),
		zap.Int("totalTokens", currentTokens),
		zap.String("generatedReason", reason))

	return reason, nil
}

// validateFlaggedUsers validates the flagged users against the translated content
// but uses original descriptions when creating validated users. It checks if at least
// 50% of the flagged words are found in the translated content to confirm the AI's findings.
func (a *AIChecker) validateFlaggedUsers(flaggedUsers FlaggedUsers, translatedInfos map[string]*fetcher.Info, originalInfos map[string]*fetcher.Info) ([]*database.User, []uint64) {
	var validatedUsers []*database.User
	var failedValidationIDs []uint64

	for _, flaggedUser := range flaggedUsers.Users {
		normalizedName := utils.NormalizeString(flaggedUser.Name)

		// Check if the flagged user exists in both maps
		translatedInfo, exists := translatedInfos[normalizedName]
		originalInfo, hasOriginal := originalInfos[normalizedName]

		if exists && hasOriginal {
			// Split all flagged content into words
			var allFlaggedWords []string
			for _, content := range flaggedUser.FlaggedContent {
				allFlaggedWords = append(allFlaggedWords, strings.Fields(content)...)
			}

			// Count how many flagged words are found in the translated content
			foundWords := 0
			for _, word := range allFlaggedWords {
				if utils.ContainsNormalized(translatedInfo.Name, word) || utils.ContainsNormalized(translatedInfo.Description, word) {
					foundWords++
				}
			}

			// Check if at least 50% of the flagged words are found
			isValid := float64(foundWords) >= 0.5*float64(len(allFlaggedWords))

			// If the flagged user is correct, add it using original info
			if isValid {
				validatedUsers = append(validatedUsers, &database.User{
					ID:             originalInfo.ID,
					Name:           originalInfo.Name,
					DisplayName:    originalInfo.DisplayName,
					Description:    originalInfo.Description,
					CreatedAt:      originalInfo.CreatedAt,
					Reason:         flaggedUser.Reason,
					Groups:         originalInfo.Groups,
					Friends:        originalInfo.Friends,
					FlaggedContent: flaggedUser.FlaggedContent,
					Confidence:     flaggedUser.Confidence,
					LastUpdated:    originalInfo.LastUpdated,
				})
			} else {
				failedValidationIDs = append(failedValidationIDs, originalInfo.ID)
				a.logger.Warn("AI flagged content did not pass validation",
					zap.Uint64("userID", originalInfo.ID),
					zap.String("flaggedUsername", flaggedUser.Name),
					zap.String("username", originalInfo.Name),
					zap.String("description", originalInfo.Description),
					zap.Strings("flaggedContent", flaggedUser.FlaggedContent),
					zap.Float64("matchPercentage", float64(foundWords)/float64(len(allFlaggedWords))*100))
			}
		} else {
			a.logger.Warn("AI flagged non-existent user", zap.String("username", flaggedUser.Name))
		}
	}

	return validatedUsers, failedValidationIDs
}

// prepareUserInfos translates user descriptions and maintains maps of both translated
// and original user infos for validation. If translation fails for any description,
// it falls back to using the original content. Returns maps using normalized usernames
// as keys.
func (a *AIChecker) prepareUserInfos(userInfos []*fetcher.Info) (map[string]*fetcher.Info, map[string]*fetcher.Info) {
	var wg sync.WaitGroup
	resultsChan := make(chan TranslationResult, len(userInfos))

	// Create maps for both original and translated infos
	originalInfos := make(map[string]*fetcher.Info)
	translatedInfos := make(map[string]*fetcher.Info)

	// Initialize maps and spawn translation goroutines
	for _, info := range userInfos {
		normalizedName := utils.NormalizeString(info.Name)
		originalInfos[normalizedName] = info

		wg.Add(1)
		go func(info *fetcher.Info) {
			defer wg.Done()

			// Skip empty descriptions
			if info.Description == "" {
				resultsChan <- TranslationResult{
					UserInfo:       info,
					TranslatedDesc: "",
				}
				return
			}

			// Translate the description
			translated, err := a.translator.Translate(
				context.Background(),
				info.Description,
				"auto", // Auto-detect source language
				"en",   // Translate to English
			)

			resultsChan <- TranslationResult{
				UserInfo:       info,
				TranslatedDesc: translated,
				Err:            err,
			}
		}(info)
	}

	// Close results channel when all translations are complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Process results
	for result := range resultsChan {
		normalizedName := utils.NormalizeString(result.UserInfo.Name)
		if result.Err != nil {
			// Use original userInfo if translation fails
			translatedInfos[normalizedName] = result.UserInfo
			a.logger.Error("Translation failed, using original description",
				zap.String("username", result.UserInfo.Name),
				zap.Error(result.Err))
			continue
		}

		// Create new Info with translated description
		translatedInfo := *result.UserInfo
		if translatedInfo.Description != result.TranslatedDesc {
			translatedInfo.Description = result.TranslatedDesc
			a.logger.Debug("Translated description", zap.String("username", translatedInfo.Name))
		}
		translatedInfos[normalizedName] = &translatedInfo
	}

	return translatedInfos, originalInfos
}
