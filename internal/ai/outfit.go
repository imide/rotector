package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"sync"

	"github.com/HugoSmits86/nativewebp"
	"github.com/bytedance/sonic"
	"github.com/jaxron/axonet/pkg/client"
	"github.com/jaxron/roapi.go/pkg/api/resources/thumbnails"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/openai/openai-go"
	"github.com/robalyx/rotector/internal/database/types"
	"github.com/robalyx/rotector/internal/database/types/enum"
	"github.com/robalyx/rotector/internal/roblox/fetcher"
	"github.com/robalyx/rotector/internal/setup"
	"github.com/robalyx/rotector/pkg/utils"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	// FlaggedOutfitSystemPrompt provides detailed instructions to the AI model for analyzing user outfits for violations.
	FlaggedOutfitSystemPrompt = `You are a Roblox moderator analyzing outfits for predatory behavior targeting minors.

Each outfit image is provided as a separate image part along with this prompt.
The first image (if present) is the user's current outfit, followed by their outfit images.
You will receive a list of outfit names that correspond to the images in order.

Output format:
{
  "username": "string",
  "reason": "Clear explanation in one sentence",
  "evidence": ["exact outfit name 1", "exact outfit name 2"],
  "confidence": 0.0-1.0
}

Confidence levels:
0.0: No violations
0.1-0.3: One inappropriate outfit
0.4-0.6: Two inappropriate outfits
0.7-0.8: Three inappropriate outfits
0.9-1.0: Four+ inappropriate outfits

Key rules:
1. Return ONLY users with violations
2. Use "the outfit" instead of outfit names in reasons
3. Include exact outfit names in evidence ONLY if their images violate guidelines
4. Each evidence entry must be a single, complete outfit name - do not combine names
5. NEVER flag an outfit based on its name alone - analyze only the visual content
6. Ensure outfit names in evidence match exactly with the provided names
7. Do not modify, combine, or alter outfit names in any way

Flag outfits containing:

REVEALING OR INAPPROPRIATE CLOTHING:
- Extreme crop tops showing underboob or excessive skin
- Deep v-necks or plunging necklines emphasizing cleavage
- Transparent or semi-transparent clothing textures
- Inappropriate or overly revealing swimwear, including microkinis
- Anatomical outlining or exaggerated features (nipples, genitalia)
- Thongs, g-strings, or very short shorts exposing the buttocks
- Fishnet or see-through leggings/stockings with suggestive detailing
- Low-rise pants that imply nudity or expose hip areas
- Lingerie, underwear sets, or "clubwear" (even if cartoonish)
- Stripper-themed outfits or pole-dancing props
- Nudist or "censored nudity" looks with strategic coverings (pixelation, bars, stickers)
- Bodycon dresses or clothing that excessively emphasizes body contours
- Provocative bodysuits (especially those with cutouts or exposed areas)
- Decorative or intentional cleavage holes (heart-shaped, keyholes, etc.)

SUGGESTIVE AVATAR FEATURES:
- Skintight bodysuits designed to mimic nudity
- Hyper-curvy bodies with exaggerated bust/hips/buttocks
- Oversized or emphasized chest/breast areas
- Unrealistic body proportions meant to be suggestive
- Bare skin textures that simulate nudity or use "clever" illusions
- Body meshes with breast bulges or buttocks clearly modeled in 3D
- Invisible clothing or skin-colored meshes that simulate nakedness
- Outfits with suggestive protrusions or attachments mimicking genitalia
- Overly detailed or emphasized realistic belly buttons
- Ultra-slim waist avatars promoting unrealistic body standards
- NSFW parody bodies with strategic coverings or "nude" base models
- Inflated or exaggerated anatomical features
- Succubus-themed womb tattoos or similar suggestive body markings
- Sexualized chest scars (particularly those fetishizing trans surgeries)

ADULT-THEMED OUTFITS AND ACCESSORIES:
- Bondage, BDSM, or fetish-themed clothing and accessories
- Latex catsuits with exaggerated body outlines
- Collar and leash combinations, ball gags, or chains
- Chokers with inappropriate phrases ("Daddy", "Owned", etc.)
- Explicit roleplay outfits (e.g., suggestive maid, nurse, etc.)
- Outfits designed to simulate sexual acts or positions
- Revealing "cow girl" or similar animal costumes with fetish elements (udders, etc.)
- Provocative furry or anthropomorphic designs
- School uniforms with suggestive modifications (extremely short skirts, open blouses)
- Accessories like whips, cuffs, or props with sexual connotations
- Clothing with explicit text, bypassed inappropriate words, or suggestive messaging
- Characters recreated from adult games, shows, or media
- Petplay elements (ears, tails, collars, leashes in suggestive contexts)
- Diaper or "little" cosplays with sexualized elements

UNDERAGE-SEXUALIZED CONTENT:
- Childlike characters or avatars in revealing outfits
- Sexualized "loli" anime character recreations
- Adult accessories or themes paired with childish elements
- Suggestive schoolgirl outfits with inappropriate modifications
- Content combining youthful appearance with sexual themes
- Outfits featuring childlike proportions with revealing elements
- "Baby" or infant-styled outfits with inappropriate additions

Do NOT Flag:
- Outfits based on their names alone
- Regular, non-revealing casual clothing or fashion items
- Default Roblox clothing and official outfits
- Visible bra straps or tank top straps
- Standard or proportionate body types and avatars
- Single minor violations that do not suggest inappropriate intent
- Accessories that are clearly weapons, tools, or non-sexual objects`

	// FlaggedOutfitRequestPrompt provides a reminder to follow system guidelines for outfit violation analysis.
	FlaggedOutfitRequestPrompt = `Analyze these outfits for inappropriate content.

Remember:
1. Each image part corresponds to the outfit name at the same position in the list
2. The first image (if present) is always the current outfit
3. Use exact outfit names when providing evidence
4. Include only outfits whose IMAGES clearly violate the guidelines
5. DO NOT flag outfits based on their names - analyze only the visual content

Outfits to analyze (in order of corresponding images):
`

	// OutfitThemePrompt provides instructions for analyzing outfit themes without looking for violations.
	OutfitThemePrompt = `You are a Roblox outfit analyzer detecting specific outfit themes.

Each outfit image is provided as a separate image part along with this prompt.
The first image (if present) is the user's current outfit, followed by their outfit images.
You will receive a list of outfit names that correspond to the images in order.

Output format:
{
  "username": "string",
  "themes": [
    {
      "outfitName": "exact outfit name",
      "theme": "identified theme",
      "confidence": 0.0-1.0
    }
  ]
}

Identify these specific themes and categories (only if clearly present):

SEXUAL/ADULT THEMES:
- Stripper/pole dancer outfits
- Lingerie/underwear models
- Sexualized maid outfits (especially with fishnets, cleavage)
- Bunny girl outfits (lingerie versions)
- Latex catsuits/dominatrix outfits
- Fetishwear (bondage elements, suggestive accessories)
- Nudist/censored nudity looks (with pixelation, censor bars, stickers)
- Revealing swimsuit/microkini outfits (especially with exaggerated anatomy)
- Provocative bodysuits with cutouts or revealing elements
- Thongs/g-strings or outfits emphasizing exposed buttocks
- Outfits with intentional cleavage cutouts or revealing holes (heart-shaped, keyholes)
- Succubus-themed outfits (especially with womb tattoos or markings)

BODY/FIGURE-FOCUSED THEMES:
- Exaggerated curvy/thicc avatars
- Hourglass figure avatars (especially with suggestive poses)
- Inflated chest/butt avatars
- Bodycon dress models
- Ultra-slim waist avatars
- NSFW parody bodies (like "nude" base models with strategic coverings)
- Bodies with sexualized scars or markings (including sexualized chest scars)

BDSM/KINK/FETISH PARODIES:
- Bondage sets (chains, gags, collars)
- Petplay (ears, collars, leashes in suggestive context)
- Slave-themed outfits (with chains, torn clothing)
- Leather harnesses/latex corsets
- "Cow girl" outfits (NOT cowboy/western, but fetish-themed outfits with cow print, udders, or animal-sexualization elements)

PEDOBAIT/UNDERAGE-SEXUALIZED LOOKS:
- Childlike characters in revealing outfits
- Diaper or "little" cosplays with sexual tones
- Adult accessories paired with childish elements
- Sexualized "loli" anime characters
- Suggestive schoolgirl outfits

Theme confidence levels:
0.0-0.3: Possible but unclear theme elements
0.4-0.7: Clear but modest theme elements
0.8-1.0: Explicit or obvious theme elements

Key rules:
1. Include the exact outfit name
2. Only identify themes if they are clearly visible in the image
3. Do not flag inappropriate content - focus only on theme identification
4. Return empty themes array if no specified themes are detected
5. Each theme detection should include the full outfit name, identified theme, and confidence level
6. Be specific about the theme category when possible (e.g., "BDSM/Fetish: Latex catsuit" rather than just "Latex catsuit")`

	// OutfitThemeRequestPrompt provides a reminder to focus on theme identification.
	OutfitThemeRequestPrompt = `Identify specific themes in these outfits.

Remember:
1. Each image part corresponds to the outfit name at the same position in the list
2. The first image (if present) is always the current outfit
3. Focus only on identifying themes, not on flagging inappropriate content
4. Only include outfits that clearly match one of the specified themes
5. Return the exact outfit name in your analysis

Outfits to analyze (in order of corresponding images):
`
)

const (
	MaxOutfits = 15
)

var (
	ErrNoViolations        = errors.New("no violations found in outfits")
	ErrNoOutfits           = errors.New("no outfit images downloaded successfully")
	ErrInvalidThumbnailURL = errors.New("invalid thumbnail URL")
)

// OutfitAnalysis contains the AI's analysis results for a user's outfits.
type OutfitAnalysis struct {
	Username   string   `json:"username"`
	Reason     string   `json:"reason"`
	Evidence   []string `json:"evidence"`
	Confidence float64  `json:"confidence"`
}

// OutfitThemeAnalysis contains the AI's theme detection results for a user's outfits.
type OutfitThemeAnalysis struct {
	Username string        `json:"username"`
	Themes   []OutfitTheme `json:"themes"`
}

// OutfitTheme represents a detected theme for a single outfit.
type OutfitTheme struct {
	OutfitName string  `json:"outfitName"`
	Theme      string  `json:"theme"`
	Confidence float64 `json:"confidence"`
}

// OutfitAnalysisSchema is the JSON schema for the outfit analysis response.
var OutfitAnalysisSchema = utils.GenerateSchema[OutfitAnalysis]()

// OutfitThemeAnalysisSchema is the JSON schema for the outfit theme analysis response.
var OutfitThemeAnalysisSchema = utils.GenerateSchema[OutfitThemeAnalysis]()

// OutfitAnalyzer handles AI-based outfit analysis using OpenAI models.
type OutfitAnalyzer struct {
	httpClient       *client.Client
	openAIClient     *openai.Client
	thumbnailFetcher *fetcher.ThumbnailFetcher
	analysisSem      *semaphore.Weighted
	logger           *zap.Logger
	model            string
}

// DownloadResult contains the result of a single outfit image download.
type DownloadResult struct {
	img  image.Image
	name string
}

// NewOutfitAnalyzer creates an OutfitAnalyzer instance.
func NewOutfitAnalyzer(app *setup.App, logger *zap.Logger) *OutfitAnalyzer {
	return &OutfitAnalyzer{
		httpClient:       app.RoAPI.GetClient(),
		openAIClient:     app.OpenAIClient,
		thumbnailFetcher: fetcher.NewThumbnailFetcher(app.RoAPI, logger),
		analysisSem:      semaphore.NewWeighted(int64(app.Config.Worker.BatchSizes.OutfitAnalysis)),
		logger:           logger.Named("ai_outfit"),
		model:            app.Config.Common.OpenAI.Model,
	}
}

// ProcessOutfits analyzes outfit images for a batch of users.
func (a *OutfitAnalyzer) ProcessOutfits(userInfos []*types.User, reasonsMap map[uint64]types.Reasons[enum.UserReasonType]) {
	// Separate users into flagged and non-flagged groups
	var flaggedInfos, unflaggedInfos []*types.User
	for _, info := range userInfos {
		if _, isFlagged := reasonsMap[info.ID]; isFlagged {
			flaggedInfos = append(flaggedInfos, info)
		} else {
			unflaggedInfos = append(unflaggedInfos, info)
		}
	}

	// Process flagged users (looking for violations)
	if len(flaggedInfos) > 0 {
		a.processFlaggedUsersOutfits(flaggedInfos, reasonsMap)
	}

	// Process non-flagged users (looking for specific themes)
	if len(unflaggedInfos) > 0 {
		a.processUnflaggedUsersOutfits(unflaggedInfos, reasonsMap)
	}
}

// processFlaggedUsersOutfits analyzes outfit images for flagged users.
func (a *OutfitAnalyzer) processFlaggedUsersOutfits(
	flaggedInfos []*types.User, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
) {
	// Get all outfit thumbnails organized by user
	userOutfits, userThumbnails := a.getOutfitThumbnails(context.Background(), flaggedInfos)

	// Process each user's outfits concurrently
	var (
		ctx = context.Background()
		p   = pool.New().WithContext(ctx)
		mu  sync.Mutex
	)

	for _, userInfo := range flaggedInfos {
		// Skip if user has no outfits
		outfits, hasOutfits := userOutfits[userInfo.ID]
		if !hasOutfits {
			continue
		}

		thumbnails := userThumbnails[userInfo.ID]

		p.Go(func(ctx context.Context) error {
			// Analyze user's outfits
			err := a.analyzeFlaggedUserOutfits(ctx, userInfo, &mu, reasonsMap, outfits, thumbnails)
			if err != nil && !errors.Is(err, ErrNoViolations) {
				a.logger.Error("Failed to analyze outfits",
					zap.Error(err),
					zap.Uint64("userID", userInfo.ID))
				return err
			}
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := p.Wait(); err != nil {
		a.logger.Error("Error during outfit analysis for flagged users", zap.Error(err))
		return
	}

	a.logger.Info("Received AI outfit analysis for flagged users",
		zap.Int("totalUsers", len(flaggedInfos)))
}

// processUnflaggedUsersOutfits analyzes outfit images for non-flagged users.
func (a *OutfitAnalyzer) processUnflaggedUsersOutfits(
	unflaggedInfos []*types.User, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
) {
	// Get all outfit thumbnails organized by user
	userOutfits, userThumbnails := a.getOutfitThumbnails(context.Background(), unflaggedInfos)

	// Process each user's outfits concurrently
	var (
		ctx = context.Background()
		p   = pool.New().WithContext(ctx)
		mu  sync.Mutex
	)

	for _, userInfo := range unflaggedInfos {
		// Skip if user has no outfits
		outfits, hasOutfits := userOutfits[userInfo.ID]
		if !hasOutfits {
			continue
		}

		thumbnails := userThumbnails[userInfo.ID]

		p.Go(func(ctx context.Context) error {
			// Analyze user's outfits for themes
			err := a.analyzeUnflaggedUserOutfits(ctx, userInfo, &mu, reasonsMap, outfits, thumbnails)
			if err != nil && !errors.Is(err, ErrNoViolations) {
				a.logger.Error("Failed to analyze outfit themes",
					zap.Error(err),
					zap.Uint64("userID", userInfo.ID))
				return err
			}
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := p.Wait(); err != nil {
		a.logger.Error("Error during outfit theme analysis for non-flagged users", zap.Error(err))
		return
	}

	a.logger.Info("Received AI outfit theme analysis for non-flagged users",
		zap.Int("totalUsers", len(unflaggedInfos)))
}

// getOutfitThumbnails fetches thumbnail URLs for outfits and organizes them by user.
func (a *OutfitAnalyzer) getOutfitThumbnails(
	ctx context.Context, userInfos []*types.User,
) (map[uint64][]*apiTypes.Outfit, map[uint64]map[uint64]string) {
	userOutfits := make(map[uint64][]*apiTypes.Outfit)
	requests := thumbnails.NewBatchThumbnailsBuilder()

	// Organize outfits by user and build thumbnail requests
	for _, userInfo := range userInfos {
		// Limit outfits per user
		outfits := userInfo.Outfits
		if len(outfits) > MaxOutfits {
			outfits = outfits[:MaxOutfits]
		}

		userOutfits[userInfo.ID] = outfits

		// Add thumbnail requests for each outfit
		for _, outfit := range outfits {
			requests.AddRequest(apiTypes.ThumbnailRequest{
				Type:      apiTypes.OutfitType,
				TargetID:  outfit.ID,
				RequestID: strconv.FormatUint(outfit.ID, 10),
				Size:      apiTypes.Size150x150,
				Format:    apiTypes.WEBP,
			})
		}
	}

	// Get thumbnails for all outfits
	thumbnailMap := a.thumbnailFetcher.ProcessBatchThumbnails(ctx, requests)

	// Create user thumbnail map
	userThumbnails := make(map[uint64]map[uint64]string)
	for userID, outfits := range userOutfits {
		thumbnails := make(map[uint64]string)
		for _, outfit := range outfits {
			if url, ok := thumbnailMap[outfit.ID]; ok {
				thumbnails[outfit.ID] = url
			}
		}
		userThumbnails[userID] = thumbnails
	}

	return userOutfits, userThumbnails
}

// analyzeOutfits is a generic helper method that handles the common analysis steps for both flagged and unflagged users.
func (a *OutfitAnalyzer) analyzeOutfits(
	ctx context.Context, info *types.User, outfits []*apiTypes.Outfit, thumbnailMap map[uint64]string,
	systemPrompt string, requestPrompt string, promptFormat string, schema any,
	resultHandler func(
		content string, info *types.User, mu *sync.Mutex, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
	) error,
	mu *sync.Mutex, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
) error {
	// Acquire semaphore before making AI request
	if err := a.analysisSem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer a.analysisSem.Release(1)

	// Download and process each outfit image
	downloads, err := a.downloadOutfitImages(ctx, info, outfits, thumbnailMap)
	if err != nil {
		if errors.Is(err, ErrNoOutfits) {
			return ErrNoViolations
		}
		return fmt.Errorf("failed to download outfit images: %w", err)
	}

	// Process each downloaded image and add as user message parts
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	outfitNames := make([]string, 0, len(downloads))
	for _, result := range downloads {
		// Convert image to base64
		buf := new(bytes.Buffer)
		if err := nativewebp.Encode(buf, result.img, nil); err != nil {
			continue
		}
		base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())

		// Add image as a user message
		imagePart := openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
			URL: "data:image/webp;base64," + base64Image,
		})
		messages = append(messages, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{imagePart}))

		// Store outfit name
		outfitNames = append(outfitNames, result.name)
	}

	// Add final user message with outfit names
	prompt := fmt.Sprintf(
		promptFormat,
		requestPrompt,
		info.Name,
		strings.Join(outfitNames, ", "),
	)
	messages = append(messages, openai.UserMessage(prompt))

	// Generate outfit analysis
	resp, err := a.openAIClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        "outfitAnalysis",
					Description: openai.String("Analysis of user outfits"),
					Schema:      schema,
					Strict:      openai.Bool(true),
				},
			},
		},
		Model:       a.model,
		Temperature: openai.Float(0.2),
		TopP:        openai.Float(0.1),
	})
	if err != nil {
		return fmt.Errorf("openai API error: %w", err)
	}

	// Check for empty response
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.Content) == 0 {
		return fmt.Errorf("%w: no response from model", ErrModelResponse)
	}

	// Use the result handler to process the response
	return resultHandler(resp.Choices[0].Message.Content, info, mu, reasonsMap)
}

// analyzeFlaggedUserOutfits handles the analysis of a single flagged user's outfits for violations.
func (a *OutfitAnalyzer) analyzeFlaggedUserOutfits(
	ctx context.Context, info *types.User, mu *sync.Mutex, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
	outfits []*apiTypes.Outfit, thumbnailMap map[uint64]string,
) error {
	resultHandler := func(
		content string, info *types.User, mu *sync.Mutex, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
	) error {
		var result *OutfitAnalysis
		if err := sonic.Unmarshal([]byte(content), &result); err != nil {
			return fmt.Errorf("JSON unmarshal error: %w", err)
		}

		// Skip results with no violations
		if result.Confidence < 0.1 || result.Reason == "NO_VIOLATIONS" {
			return nil
		}

		// Validate confidence level
		if result.Confidence > 1.0 {
			a.logger.Debug("AI flagged user with invalid confidence",
				zap.String("username", info.Name),
				zap.Float64("confidence", result.Confidence))
			return nil
		}

		// Process evidence to handle potential newlines
		processedEvidence := utils.SplitLines(result.Evidence)

		// If analysis is successful and violations found, update reasons map
		mu.Lock()
		if _, exists := reasonsMap[info.ID]; !exists {
			reasonsMap[info.ID] = make(types.Reasons[enum.UserReasonType])
		}
		reasonsMap[info.ID].Add(enum.UserReasonTypeOutfit, &types.Reason{
			Message:    result.Reason,
			Confidence: result.Confidence,
			Evidence:   processedEvidence,
		})
		mu.Unlock()

		a.logger.Info("AI flagged user with outfit violations",
			zap.Uint64("userID", info.ID),
			zap.String("username", info.Name),
			zap.String("reason", result.Reason),
			zap.Float64("confidence", result.Confidence))

		return nil
	}

	return a.analyzeOutfits(
		ctx,
		info,
		outfits,
		thumbnailMap,
		FlaggedOutfitSystemPrompt,
		FlaggedOutfitRequestPrompt,
		"%s\n\nAnalyze outfits for user %q.\nOutfit names: %s",
		OutfitAnalysisSchema,
		resultHandler,
		mu,
		reasonsMap,
	)
}

// analyzeUnflaggedUserOutfits handles the theme analysis of a single non-flagged user's outfits.
func (a *OutfitAnalyzer) analyzeUnflaggedUserOutfits(
	ctx context.Context, info *types.User, mu *sync.Mutex, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
	outfits []*apiTypes.Outfit, thumbnailMap map[uint64]string,
) error {
	resultHandler := func(
		content string, info *types.User, mu *sync.Mutex, reasonsMap map[uint64]types.Reasons[enum.UserReasonType],
	) error {
		var result *OutfitThemeAnalysis
		if err := sonic.Unmarshal([]byte(content), &result); err != nil {
			return fmt.Errorf("JSON unmarshal error: %w", err)
		}

		// Skip results with no themes detected
		if len(result.Themes) == 0 {
			return nil
		}

		// Process themes and collect suspicious themes
		var suspiciousThemes []string
		for _, theme := range result.Themes {
			// Skip themes with low confidence
			if theme.Confidence < 0.4 {
				a.logger.Debug("Skipping theme with low confidence",
					zap.String("username", info.Name),
					zap.String("theme", theme.Theme),
					zap.Float64("confidence", theme.Confidence))
				continue
			}

			suspiciousThemes = append(suspiciousThemes,
				fmt.Sprintf("%s: %s (%.2f)", theme.OutfitName, theme.Theme, theme.Confidence))
		}

		// If no suspicious themes found, return
		if len(suspiciousThemes) == 0 {
			return nil
		}

		// Get the highest confidence theme
		var highestConfidence float64
		for _, theme := range result.Themes {
			if theme.Confidence > highestConfidence {
				highestConfidence = theme.Confidence
			}
		}

		// If themes are suspicious enough, flag the user
		if highestConfidence >= 0.5 {
			mu.Lock()
			if _, exists := reasonsMap[info.ID]; !exists {
				reasonsMap[info.ID] = make(types.Reasons[enum.UserReasonType])
			}
			reasonsMap[info.ID].Add(enum.UserReasonTypeOutfit, &types.Reason{
				Message:    "User has outfits with potentially inappropriate themes",
				Confidence: highestConfidence,
				Evidence:   suspiciousThemes,
			})
			mu.Unlock()

			a.logger.Info("AI flagged previously unflagged user based on outfit themes",
				zap.Uint64("userID", info.ID),
				zap.String("username", info.Name),
				zap.Float64("confidence", highestConfidence),
				zap.Strings("themes", suspiciousThemes))
		}

		return nil
	}

	return a.analyzeOutfits(
		ctx,
		info,
		outfits,
		thumbnailMap,
		OutfitThemePrompt,
		OutfitThemeRequestPrompt,
		"%s\n\nIdentify themes for user %q.\nOutfit names: %s",
		OutfitThemeAnalysisSchema,
		resultHandler,
		mu,
		reasonsMap,
	)
}

// downloadOutfitImages concurrently downloads outfit images until we have enough.
func (a *OutfitAnalyzer) downloadOutfitImages(
	ctx context.Context, userInfo *types.User, outfits []*apiTypes.Outfit, thumbnailMap map[uint64]string,
) ([]DownloadResult, error) {
	var (
		p         = pool.New().WithContext(ctx)
		mu        sync.Mutex
		downloads []DownloadResult
	)

	// Download current user thumbnail
	thumbnailURL := userInfo.ThumbnailURL
	if thumbnailURL != "" && thumbnailURL != fetcher.ThumbnailPlaceholder {
		p.Go(func(ctx context.Context) error {
			img, ok := a.downloadImage(ctx, thumbnailURL)
			if ok {
				mu.Lock()
				// Add current outfit at the beginning of the array
				downloads = append(downloads, DownloadResult{
					img:  img,
					name: "Current Outfit",
				})
				mu.Unlock()
			}
			return nil
		})
	}

	// Process outfits concurrently
	for _, outfit := range outfits {
		// Check if thumbnail is valid
		thumbnailURL, ok := thumbnailMap[outfit.ID]
		if !ok || thumbnailURL == "" || thumbnailURL == fetcher.ThumbnailPlaceholder {
			continue
		}

		p.Go(func(ctx context.Context) error {
			img, ok := a.downloadImage(ctx, thumbnailURL)
			if !ok {
				return nil
			}

			mu.Lock()
			downloads = append(downloads, DownloadResult{
				img:  img,
				name: outfit.Name,
			})
			mu.Unlock()

			return nil
		})
	}

	// Wait for all downloads to complete
	if err := p.Wait(); err != nil {
		a.logger.Error("Error during outfit downloads", zap.Error(err))
	}

	// Check if we got any successful downloads
	if len(downloads) == 0 {
		return nil, ErrNoOutfits
	}

	return downloads, nil
}

// downloadImage downloads an image from a URL.
func (a *OutfitAnalyzer) downloadImage(ctx context.Context, url string) (image.Image, bool) {
	// Download image
	resp, err := a.httpClient.NewRequest().URL(url).Do(ctx)
	if err != nil {
		a.logger.Warn("Failed to download outfit image",
			zap.Error(err),
			zap.String("url", url))
		return nil, false
	}
	defer resp.Body.Close()

	// Decode image
	img, err := nativewebp.Decode(resp.Body)
	if err != nil {
		return nil, false
	}

	return img, true
}
