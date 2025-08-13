package services

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// ProfileParser handles EDI parsing with partner profile configurations
type ProfileParser struct {
	profileManager *profiles.ProfileManager
}

// NewProfileParser creates a new profile-aware parser
func NewProfileParser(profileManager *profiles.ProfileManager) *ProfileParser {
	return &ProfileParser{
		profileManager: profileManager,
	}
}

// ParseWithProfile parses EDI content using partner profile configuration
func (p *ProfileParser) ParseWithProfile(content []byte, partnerID string) (*x12.Document, error) {
	// Load partner profile
	profile, err := p.profileManager.GetProfile(partnerID)
	if err != nil {
		return p.parseWithDefaults(content)
	}

	delims := profile.GetDelimiters()

	parser := x12.NewParserWithProfile(partnerID, delims)

	doc, err := parser.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse EDI with profile %s: %w", partnerID, err)
	}

	return doc, nil
}

// ParseWithProfileReader parses EDI content from a reader using partner profile
func (p *ProfileParser) ParseWithProfileReader(
	r io.Reader,
	partnerID string,
) (*x12.Document, error) {
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, fmt.Errorf("failed to read EDI content: %w", err)
	}

	return p.ParseWithProfile(buf.Bytes(), partnerID)
}

// DetectPartnerFromContent attempts to detect the partner from EDI content
func (p *ProfileParser) DetectPartnerFromContent(content []byte) (string, error) {
	contentStr := string(content)

	isaIndex := strings.Index(contentStr, "ISA")
	if isaIndex == -1 {
		return "", fmt.Errorf("no ISA segment found")
	}

	commonDelimiters := []rune{'*', '|', '^', '`'}

	for range commonDelimiters {
		if isaIndex+3 < len(contentStr) {
			possibleDelim := rune(contentStr[isaIndex+3])

			isaEnd := strings.IndexAny(contentStr[isaIndex:], "~\n\r")
			if isaEnd == -1 {
				isaEnd = len(contentStr) - isaIndex
			} else {
				isaEnd += isaIndex
			}

			isaSegment := contentStr[isaIndex:isaEnd]
			elements := strings.Split(isaSegment, string(possibleDelim))

			if len(elements) >= 16 {
				senderID := strings.TrimSpace(elements[6])

				entries := p.profileManager.ListProfiles()
				for _, entry := range entries {
					if strings.EqualFold(entry.PartnerID, senderID) ||
						strings.Contains(
							strings.ToUpper(senderID),
							strings.ToUpper(entry.PartnerID),
						) ||
						strings.Contains(
							strings.ToUpper(entry.PartnerID),
							strings.ToUpper(senderID),
						) {
						return entry.PartnerID, nil
					}
				}

				break
			}
		}
	}

	return "", fmt.Errorf("unable to detect partner from content")
}

// parseWithDefaults parses EDI with default settings
func (p *ProfileParser) parseWithDefaults(content []byte) (*x12.Document, error) {
	parser := x12.NewParser()
	return parser.Parse(bytes.NewReader(content))
}

// ValidateWithProfile validates EDI document using partner profile rules
func (p *ProfileParser) ValidateWithProfile(doc *x12.Document, partnerID string) error {
	profile, err := p.profileManager.GetProfile(partnerID)
	if err != nil {
		return doc.Validate()
	}

	if err := doc.Validate(); err != nil {
		return err
	}

	if profile.ValidationConfig.Strictness == "strict" {
		if profile.ValidationConfig.EnforceSegmentOrder {
			if err := p.validateSegmentOrder(doc, profile); err != nil {
				return err
			}
		}

		if profile.ValidationConfig.EnforceRequiredElements {
			if err := p.validateRequiredElements(doc, profile); err != nil {
				return err
			}
		}

		for _, rule := range profile.ValidationConfig.CustomRules {
			if err := p.applyValidationRule(doc, rule); err != nil {
				if rule.Severity == "error" {
					return err
				}
			}
		}
	}

	return nil
}

// validateSegmentOrder checks segment order against profile requirements
func (p *ProfileParser) validateSegmentOrder(
	doc *x12.Document,
	profile *profiles.PartnerProfile,
) error {
	transSupport := new(profiles.TransactionSupport)

	for _, ts := range profile.SupportedTransactions {
		if ts.TransactionType == doc.Metadata.TransactionType {
			transSupport = &ts
			break
		}
	}

	if transSupport == nil {
		return fmt.Errorf(
			"transaction type %s not supported by partner profile",
			doc.Metadata.TransactionType,
		)
	}

	// Check required segments are present
	requiredFound := make(map[string]bool)
	for _, required := range transSupport.RequiredSegments {
		requiredFound[required] = false
	}

	for _, seg := range doc.Segments {
		if _, ok := requiredFound[seg.Tag]; ok {
			requiredFound[seg.Tag] = true
		}
	}

	for seg, found := range requiredFound {
		if !found {
			return fmt.Errorf("required segment %s not found", seg)
		}
	}

	return nil
}

// validateRequiredElements checks that required elements are present
func (p *ProfileParser) validateRequiredElements(
	doc *x12.Document,
	profile *profiles.PartnerProfile,
) error {
	// ! This would check element-level requirements based on segment schemas
	// ! For now, just ensure critical segments have required elements

	for _, seg := range doc.Segments {
		switch seg.Tag {
		case "ISA":
			if len(seg.Elements) < 16 {
				return fmt.Errorf("ISA segment missing required elements")
			}
		case "GS":
			if len(seg.Elements) < 8 {
				return fmt.Errorf("GS segment missing required elements")
			}
		case "ST":
			if len(seg.Elements) < 2 {
				return fmt.Errorf("ST segment missing required elements")
			}
		}
	}

	return nil
}

// applyValidationRule applies a custom validation rule
func (p *ProfileParser) applyValidationRule(doc *x12.Document, rule profiles.ValidationRule) error {
	for _, seg := range doc.Segments {
		if seg.Tag == rule.SegmentTag {
			if rule.ElementPosition > 0 && rule.ElementPosition <= len(seg.Elements) {
				element := seg.Elements[rule.ElementPosition-1]
				if len(element) > 0 {
					if rule.Condition != "" {
						if re, err := regexp.Compile(rule.Condition); err == nil {
							if !re.MatchString(element[0]) {
								return fmt.Errorf("%s: %s", rule.Name, rule.ErrorMessage)
							}
						} else {
							// Fall back to simple string matching
							if !strings.Contains(element[0], rule.Condition) {
								return fmt.Errorf("%s: %s", rule.Name, rule.ErrorMessage)
							}
						}
					}
				}
			}
		}
	}

	return nil
}
