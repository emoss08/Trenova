package moderation

import (
	"context"
	"embed"
	"strings"

	"github.com/emoss08/trenova/pkg/utils"
	"github.com/openai/openai-go/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//go:embed en.txt
var wordsFS embed.FS

type Params struct {
	fx.In

	Logger       *zap.Logger
	OpenAIClient openai.Client
}

type Service struct {
	l                *zap.Logger
	openAIClient     openai.Client
	prohibitedWords  map[string]struct{}
	prohibitedLoaded bool
}

//nolint:gocritic // This is a constructor
func NewService(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.moderation"),
		openAIClient:    p.OpenAIClient,
		prohibitedWords: make(map[string]struct{}),
	}
}

type Result struct {
	Flagged        bool     `json:"flagged"`
	Reason         string   `json:"reason"`
	Categories     []string `json:"categories,omitempty"`
	ProhibitedWord bool     `json:"prohibited_word"`
	MatchedWords   []string `json:"matched_words,omitempty"`
}

func (s *Service) ModerateText(
	ctx context.Context,
	text string,
) (*Result, error) {
	log := s.l.With(
		zap.String("operation", "ModerateText"),
		zap.String("text", text),
	)

	matchedWords, hasProhibitedWord := s.detectProhibitedWords(text)

	result, err := s.openAIClient.Moderations.New(ctx, openai.ModerationNewParams{
		Model: openai.ModerationModelOmniModerationLatest,
		Input: openai.ModerationNewParamsInputUnion{
			OfString: openai.String(text),
		},
	})
	if err != nil {
		log.Error("failed to moderate text", zap.Error(err))
		return nil, err
	}

	flaggedByAI, categories, aiReason := s.parseOpenAIResponse(result)

	isFlagged := flaggedByAI || hasProhibitedWord
	reason := s.buildReason(hasProhibitedWord, flaggedByAI, aiReason)

	return &Result{
		Flagged:        isFlagged,
		Reason:         reason,
		Categories:     categories,
		ProhibitedWord: hasProhibitedWord,
		MatchedWords:   matchedWords,
	}, nil
}

func (s *Service) parseOpenAIResponse(
	result *openai.ModerationNewResponse,
) (flagged bool, categories []string, reason string) {
	if len(result.Results) == 0 {
		return false, nil, ""
	}

	moderationResult := result.Results[0]
	categories = s.extractFlaggedCategories(&moderationResult)

	var aiReason string
	if moderationResult.Flagged && len(categories) > 0 {
		var builder strings.Builder
		builder.WriteString("Content flagged for: ")
		builder.WriteString(strings.Join(categories, ", "))
		aiReason = builder.String()
	}

	return moderationResult.Flagged, categories, aiReason
}

func (s *Service) extractFlaggedCategories(result *openai.Moderation) []string {
	var categories []string

	categoryMap := map[bool]string{
		result.Categories.Hate:                  "hate",
		result.Categories.HateThreatening:       "hate/threatening",
		result.Categories.Harassment:            "harassment",
		result.Categories.HarassmentThreatening: "harassment/threatening",
		result.Categories.SelfHarm:              "self-harm",
		result.Categories.SelfHarmIntent:        "self-harm/intent",
		result.Categories.SelfHarmInstructions:  "self-harm/instructions",
		result.Categories.Sexual:                "sexual",
		result.Categories.SexualMinors:          "sexual/minors",
		result.Categories.Violence:              "violence",
		result.Categories.ViolenceGraphic:       "violence/graphic",
	}

	for flagged, category := range categoryMap {
		if flagged {
			categories = append(categories, category)
		}
	}

	return categories
}

func (s *Service) buildReason(hasProhibitedWord, flaggedByAI bool, aiReason string) string {
	switch {
	case hasProhibitedWord && flaggedByAI:
		var builder strings.Builder
		builder.WriteString("Contains prohibited words and ")
		builder.WriteString(aiReason)
		return builder.String()
	case hasProhibitedWord:
		return "Contains prohibited words"
	case flaggedByAI:
		return aiReason
	default:
		return ""
	}
}

// detectProhibitedWords checks if the text contains any prohibited words.
//
// Algorithm Complexity:
// - Time: O(n + m*k) where n = length of text, m = number of prohibited words, k = average word length
// - Space: O(w) where w = number of unique words in text
//
// The algorithm uses a two-pass approach for optimal performance:
//
// Pass 1 - Fast Word Lookup (O(n)):
//   - Tokenizes input text into words
//   - Performs O(1) hash map lookup for each word
//   - Catches most common cases instantly
//
// Pass 2 - Phrase Detection (O(m*k)):
//   - Scans for multi-word phrases (e.g., "bad word")
//   - Handles compound words (e.g., "badword")
//   - Uses word boundary detection to avoid false positives
//   - Only checks words not already found in Pass 1
//
// Returns matched words and whether any were found.
func (s *Service) detectProhibitedWords(text string) ([]string, bool) {
	if err := s.ensureProhibitedWordsLoaded(); err != nil {
		s.l.Error("failed to load prohibited words", zap.Error(err))
		return nil, false
	}

	normalizedText := strings.ToLower(text)
	matchedWords := make(map[string]struct{})

	words := s.tokenizeText(normalizedText)
	for _, word := range words {
		if _, exists := s.prohibitedWords[word]; exists {
			matchedWords[word] = struct{}{}
		}
	}

	for prohibitedWord := range s.prohibitedWords {
		if len(prohibitedWord) == 1 {
			continue
		}

		if _, alreadyFound := matchedWords[prohibitedWord]; alreadyFound {
			continue
		}

		if s.findWordBoundaryMatch(normalizedText, prohibitedWord) {
			matchedWords[prohibitedWord] = struct{}{}
		}
	}

	return s.mapKeysToSlice(matchedWords), len(matchedWords) > 0
}

func (s *Service) ensureProhibitedWordsLoaded() error {
	if s.prohibitedLoaded {
		return nil
	}

	wordsData, err := wordsFS.ReadFile("en.txt")
	if err != nil {
		return err
	}

	for line := range strings.SplitSeq(string(wordsData), "\n") {
		word := strings.TrimSpace(strings.ToLower(line))
		if word == "" || strings.HasPrefix(word, "#") { // Support comments
			continue
		}
		s.prohibitedWords[word] = struct{}{}
	}

	s.prohibitedLoaded = true
	s.l.Info("loaded prohibited words", zap.Int("count", len(s.prohibitedWords)))
	return nil
}

func (s *Service) tokenizeText(text string) []string {
	var words []string
	var currentWord strings.Builder

	for _, r := range text {
		if utils.IsLetter(r) || r == '\'' {
			currentWord.WriteRune(r)
		} else if currentWord.Len() > 0 {
			words = append(words, currentWord.String())
			currentWord.Reset()
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

func (s *Service) findWordBoundaryMatch(text, word string) bool {
	searchPos := 0
	textLen := len(text)
	wordLen := len(word)

	for searchPos < textLen {
		idx := strings.Index(text[searchPos:], word)
		if idx == -1 {
			return false
		}

		absPos := searchPos + idx

		hasStartBoundary := absPos == 0 || !utils.IsLetter(rune(text[absPos-1]))
		hasEndBoundary := absPos+wordLen >= textLen || !utils.IsLetter(rune(text[absPos+wordLen]))

		if hasStartBoundary && hasEndBoundary {
			return true
		}

		searchPos = absPos + wordLen
	}

	return false
}

func (s *Service) mapKeysToSlice(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}

	result := make([]string, 0, len(m))
	for key := range m {
		result = append(result, key)
	}
	return result
}
