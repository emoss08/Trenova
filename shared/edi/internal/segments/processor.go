package segments

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/shared/edi/internal/errors"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// SegmentProcessor handles parsing, validation, and transformation of EDI segments
type SegmentProcessor struct {
	registry         *SegmentRegistry
	customerOverlay  *CustomerRequirements
	errorCollector   *errors.ErrorCollector
	transformers     []SegmentTransformer
	validators       []SegmentValidator
	validationConfig ValidationConfig
	mu               sync.RWMutex
}

// CustomerRequirements defines customer-specific EDI requirements
type CustomerRequirements struct {
	PartnerID       string                       `json:"partner_id"`
	Version         string                       `json:"version"`
	TransactionType string                       `json:"transaction_type"`
	SegmentRules    map[string]SegmentOverlay    `json:"segment_rules"`
	LoopRules       map[string]LoopRequirement   `json:"loop_rules"`
	Conditionals    []ConditionalRule            `json:"conditional_rules"`
	CustomMappings  map[string]MappingDefinition `json:"custom_mappings"`
}

// SegmentOverlay defines customer-specific overrides for a segment
type SegmentOverlay struct {
	SegmentID   string                 `json:"segment_id"`
	Required    *bool                  `json:"required,omitempty"`
	MinOccurs   *int                   `json:"min_occurs,omitempty"`
	MaxOccurs   *int                   `json:"max_occurs,omitempty"`
	Elements    map[int]ElementOverlay `json:"elements"`
	CustomRules []ValidationRule       `json:"custom_rules"`
}

// ElementOverlay defines customer-specific overrides for an element
type ElementOverlay struct {
	Required     *bool    `json:"required,omitempty"`
	DefaultValue string   `json:"default_value,omitempty"`
	AllowedCodes []string `json:"allowed_codes,omitempty"`
	MinLength    *int     `json:"min_length,omitempty"`
	MaxLength    *int     `json:"max_length,omitempty"`
	Transform    string   `json:"transform,omitempty"` // e.g., "uppercase", "trim", "pad_left:10"
}

// LoopRequirement defines requirements for a loop/group
type LoopRequirement struct {
	LoopID    string   `json:"loop_id"`
	Required  bool     `json:"required"`
	MinOccurs int      `json:"min_occurs"`
	MaxOccurs int      `json:"max_occurs"`
	Segments  []string `json:"segments"` // Ordered segment IDs in loop
}

// ConditionalRule defines cross-segment validation rules
type ConditionalRule struct {
	ID          string      `json:"id"`
	Description string      `json:"description"`
	When        Condition   `json:"when"`
	Then        Requirement `json:"then"`
	Severity    string      `json:"severity"`
}

// Condition defines when a rule applies
type Condition struct {
	Segment  string `json:"segment"`
	Element  int    `json:"element"`
	Operator string `json:"operator"` // "equals", "not_equals", "contains", "exists"
	Value    string `json:"value"`
}

// Requirement defines what must be true when condition is met
type Requirement struct {
	Segment string `json:"segment"`
	Element int    `json:"element"`
	MustBe  string `json:"must_be"` // "present", "absent", "equal_to"
	Value   string `json:"value,omitempty"`
}

// MappingDefinition defines how to map segments to business objects
type MappingDefinition struct {
	SourceSegment string            `json:"source_segment"`
	TargetObject  string            `json:"target_object"`
	FieldMappings map[string]string `json:"field_mappings"` // element -> object field
	Transforms    map[string]string `json:"transforms"`     // field -> transform function
}

// SegmentTransformer transforms segment data
type SegmentTransformer interface {
	Transform(ctx context.Context, seg *ProcessedSegment) error
}

// SegmentValidator validates segment data
type SegmentValidator interface {
	Validate(ctx context.Context, seg *ProcessedSegment) []errors.EDIError
}

// ProcessedSegment represents a parsed and validated segment
type ProcessedSegment struct {
	Raw        x12.Segment       `json:"raw"`
	Schema     *SegmentSchema    `json:"schema"`
	Data       map[string]any    `json:"data"`
	Metadata   map[string]any    `json:"metadata"`
	Errors     []errors.EDIError `json:"errors"`
	Position   SegmentPosition   `json:"position"`
	CustomerID string            `json:"customer_id"`
}

// SegmentPosition tracks segment location in document
type SegmentPosition struct {
	Index         int    `json:"index"`
	LineNumber    int    `json:"line_number"`
	LoopID        string `json:"loop_id"`
	LoopIteration int    `json:"loop_iteration"`
	ParentLoop    string `json:"parent_loop"`
	Transaction   string `json:"transaction"`
	Functional    string `json:"functional_group"`
	Interchange   string `json:"interchange"`
}


// NewSegmentProcessor creates a new processor with default validation config
func NewSegmentProcessor(registry *SegmentRegistry) *SegmentProcessor {
	return &SegmentProcessor{
		registry:         registry,
		errorCollector:   errors.NewErrorCollector(100, false),
		transformers:     []SegmentTransformer{},
		validators:       []SegmentValidator{},
		validationConfig: GetDefaultValidationConfig(),
	}
}

// NewSegmentProcessorWithValidation creates a new processor with custom validation config
func NewSegmentProcessorWithValidation(
	registry *SegmentRegistry,
	config ValidationConfig,
) *SegmentProcessor {
	return &SegmentProcessor{
		registry:         registry,
		errorCollector:   errors.NewErrorCollector(100, false),
		transformers:     []SegmentTransformer{},
		validators:       []SegmentValidator{},
		validationConfig: config,
	}
}

// SetValidationConfig updates the validation configuration
func (p *SegmentProcessor) SetValidationConfig(config ValidationConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.validationConfig = config
}

// SetCustomerRequirements sets customer-specific requirements
func (p *SegmentProcessor) SetCustomerRequirements(req *CustomerRequirements) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.customerOverlay = req
}

// AddTransformer adds a segment transformer
func (p *SegmentProcessor) AddTransformer(t SegmentTransformer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transformers = append(p.transformers, t)
}

// AddValidator adds a segment validator
func (p *SegmentProcessor) AddValidator(v SegmentValidator) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.validators = append(p.validators, v)
}

// ProcessSegments processes raw segments into structured data
func (p *SegmentProcessor) ProcessSegments(
	ctx context.Context,
	segments []x12.Segment,
	version string,
) ([]*ProcessedSegment, error) {
	processed := make([]*ProcessedSegment, 0, len(segments))
	position := &positionTracker{
		loops:       make(map[string]int),
		currentLoop: "",
		transaction: "",
		functional:  "",
		interchange: "",
	}

	for i, seg := range segments {
		position.update(seg)

		ps, err := p.processSegment(ctx, seg, version, i, position)
		if err != nil {
			p.errorCollector.Add(errors.NewError(
				errors.ErrorTypeStructure,
				fmt.Sprintf("SEG_%s_%d", seg.Tag, i),
				fmt.Sprintf("Failed to process segment %s at position %d", seg.Tag, i),
			).WithCause(err).Build())
			continue
		}

		processed = append(processed, ps)
	}

	if err := p.validateCrossSegments(ctx, processed); err != nil {
		return processed, err
	}

	if p.errorCollector.HasErrors() {
		return processed, fmt.Errorf("processing completed with %s", p.errorCollector.Summary())
	}

	return processed, nil
}

// processSegment processes a single segment
func (p *SegmentProcessor) processSegment(
	ctx context.Context,
	seg x12.Segment,
	version string,
	index int,
	pos *positionTracker,
) (*ProcessedSegment, error) {
	schema, err := p.registry.GetSegment(version, seg.Tag)
	if err != nil {
		schema = &SegmentSchema{
			ID:      seg.Tag,
			Name:    fmt.Sprintf("Unknown Segment %s", seg.Tag),
			Version: version,
		}
	}

	ps := &ProcessedSegment{
		Raw:    seg,
		Schema: schema,
		Data:   make(map[string]any),
		Metadata: map[string]any{
			"version":   version,
			"raw_index": index,
			"timestamp": ctx.Value("timestamp"),
		},
		Position: SegmentPosition{
			Index:         index,
			LineNumber:    index + 1, // Assuming 1 segment per line
			LoopID:        pos.currentLoop,
			LoopIteration: pos.loops[pos.currentLoop],
			ParentLoop:    pos.parentLoop,
			Transaction:   pos.transaction,
			Functional:    pos.functional,
			Interchange:   pos.interchange,
		},
	}

	if err := p.parseElements(ps); err != nil {
		ps.Errors = append(ps.Errors, *errors.NewError(
			errors.ErrorTypeSyntax,
			"PARSE_ERROR",
			fmt.Sprintf("Failed to parse elements: %v", err),
		).Build())
	}

	if p.customerOverlay != nil {
		p.applyCustomerOverlay(ps)
	}

	if errs := p.validateSegment(ctx, ps); len(errs) > 0 {
		ps.Errors = append(ps.Errors, errs...)
	}

	for _, transformer := range p.transformers {
		if err := transformer.Transform(ctx, ps); err != nil {
			ps.Errors = append(ps.Errors, *errors.NewError(
				errors.ErrorTypeStructure,
				"TRANSFORM_ERROR",
				fmt.Sprintf("Transformation failed: %v", err),
			).Build())
		}
	}

	return ps, nil
}

// parseElements parses segment elements using schema
func (p *SegmentProcessor) parseElements(ps *ProcessedSegment) error {
	schema := ps.Schema
	elements := ps.Raw.Elements

	ps.Data["_segment_id"] = schema.ID
	ps.Data["_segment_name"] = schema.Name

	// ! If no schema elements defined, parse raw elements directly
	if len(schema.Elements) == 0 {
		// ! For unknown segments or segments without full schema,
		// ! parse elements based on position
		for i, elementData := range elements {
			if len(elementData) > 0 {
				key := fmt.Sprintf("%s%02d", schema.ID, i+1)
				if len(elementData) > 1 {
					compMap := make(map[string]any)
					for j, comp := range elementData {
						compKey := fmt.Sprintf("C%02d", j+1)
						compMap[compKey] = comp
					}
					ps.Data[key] = compMap
				} else {
					ps.Data[key] = elementData[0]
				}
			}
		}
		return nil
	}

	for i, elemSchema := range schema.Elements {
		if i >= len(elements) {
			if elemSchema.Required || strings.ToLower(elemSchema.Requirement) == "mandatory" {
				if p.validationConfig.Elements.EnforceMandatory {
					return fmt.Errorf(
						"required element %s-%02d missing",
						schema.ID,
						elemSchema.Position,
					)
				}
				ps.Errors = append(ps.Errors, *errors.NewError(
					errors.ErrorTypeFormat,
					"MISSING_REQUIRED_ELEMENT",
					fmt.Sprintf("Required element %s-%02d is missing", schema.ID, elemSchema.Position),
				).WithLocation(&errors.Location{
					SegmentTag:   schema.ID,
					ElementIndex: elemSchema.Position,
				}).Build())
			}
			continue
		}

		elementData := elements[i]
		if len(elementData) == 0 || (len(elementData) == 1 && elementData[0] == "") {
			if elemSchema.Required || strings.ToLower(elemSchema.Requirement) == "mandatory" {
				if p.validationConfig.Elements.EnforceMandatory {
					return fmt.Errorf(
						"required element %s-%02d is empty",
						schema.ID,
						elemSchema.Position,
					)
				}
				ps.Errors = append(ps.Errors, *errors.NewError(
					errors.ErrorTypeFormat,
					"EMPTY_REQUIRED_ELEMENT",
					fmt.Sprintf("Required element %s-%02d is empty", schema.ID, elemSchema.Position),
				).WithLocation(&errors.Location{
					SegmentTag:   schema.ID,
					ElementIndex: elemSchema.Position,
				}).Build())
			}
			continue
		}

		key := fmt.Sprintf("%s%02d", schema.ID, elemSchema.Position)

		if len(elemSchema.Components) > 0 && len(elementData) > 1 {
			compMap := make(map[string]any)
			for j, comp := range elemSchema.Components {
				if j < len(elementData) {
					compKey := fmt.Sprintf("C%02d", comp.Position)
					compMap[compKey] = elementData[j]
					compMap[compKey+"_name"] = comp.Name
				}
			}
			ps.Data[key] = compMap
		} else {
			value := ""
			if len(elementData) > 0 {
				value = elementData[0]
			}

			ps.Data[key] = value
			ps.Data[key+"_name"] = elemSchema.Name

			if strings.Contains(strings.ToUpper(elemSchema.Type), "ID") {
				for _, code := range elemSchema.Codes {
					if code.Code == value {
						ps.Data[key+"_description"] = code.Description
						break
					}
				}
			}
		}
	}

	return nil
}

// applyCustomerOverlay applies customer-specific rules
func (p *SegmentProcessor) applyCustomerOverlay(ps *ProcessedSegment) {
	overlay, exists := p.customerOverlay.SegmentRules[ps.Schema.ID]
	if !exists {
		return
	}

	for pos, elemOverlay := range overlay.Elements {
		key := fmt.Sprintf("%s%02d", ps.Schema.ID, pos)

		if val, exists := ps.Data[key]; (!exists || val == "" || val == nil) &&
			elemOverlay.DefaultValue != "" {
			ps.Data[key] = elemOverlay.DefaultValue
		}

		if elemOverlay.Transform != "" && ps.Data[key] != nil {
			ps.Data[key] = p.applyTransform(ps.Data[key], elemOverlay.Transform)
		}
	}

	ps.CustomerID = p.customerOverlay.PartnerID
}

// validateSegment validates a segment against schema and customer rules
func (p *SegmentProcessor) validateSegment(
	ctx context.Context,
	ps *ProcessedSegment,
) []errors.EDIError {
	var errs []errors.EDIError

	if p.validationConfig.Level == ValidationLevelNone {
		return errs
	}

	for _, elemSchema := range ps.Schema.Elements {
		key := fmt.Sprintf("%s%02d", ps.Schema.ID, elemSchema.Position)
		value, exists := ps.Data[key]
		valueStr := fmt.Sprintf("%v", value)

		if (elemSchema.Required || strings.ToLower(elemSchema.Requirement) == "mandatory") &&
			(!exists || value == "" || value == nil) {
			if p.validationConfig.Elements.EnforceMandatory {
				severity := p.validationConfig.GetValidationSeverity("mandatory")
				if severity == "error" {
					errs = append(errs, *errors.NewRequiredFieldError(
						ps.Schema.ID,
						elemSchema.Position,
						elemSchema.Name,
					))
				} else {
					errs = append(errs, *errors.NewError(
						errors.ErrorTypeRequired,
						fmt.Sprintf("MISSING_%s_%02d", ps.Schema.ID, elemSchema.Position),
						fmt.Sprintf("Recommended field %s is missing", elemSchema.Name),
					).WithSeverity(errors.SeverityWarning).Build())
				}
			}
		}

		if exists && value != "" && len(elemSchema.Codes) > 0 {
			shouldValidate := p.validationConfig.ShouldValidateCode(
				valueStr,
				len(elemSchema.Codes) > 0,
			)

			if shouldValidate {
				valid := false
				for _, code := range elemSchema.Codes {
					if p.validationConfig.IsCodeMatch(valueStr, code.Code) {
						valid = true
						break
					}
				}

				if !valid {
					severity := p.validationConfig.GetValidationSeverity("code")

					if severity != "ignore" &&
						p.validationConfig.Codes.InvalidCodeHandling != CodeHandlingIgnore {
						allowedCodes := make([]string, len(elemSchema.Codes))
						for i, c := range elemSchema.Codes {
							allowedCodes[i] = c.Code
						}

						if severity == "warning" ||
							p.validationConfig.Codes.InvalidCodeHandling == CodeHandlingWarning {
							errs = append(errs, *errors.NewError(
								errors.ErrorTypeValue,
								fmt.Sprintf("INVALID_CODE_%s_%02d", ps.Schema.ID, elemSchema.Position),
								fmt.Sprintf("Value '%s' is not in recommended codes for %s", valueStr, elemSchema.Name),
							).WithSeverity(errors.SeverityWarning).Build())
						} else {
							errs = append(errs, *errors.NewError(
								errors.ErrorTypeValue,
								"VALUE_INVALID",
								fmt.Sprintf("Invalid %s value '%s' in %s-%02d", elemSchema.Name, valueStr, ps.Schema.ID, elemSchema.Position),
							).WithDetails(fmt.Sprintf("Allowed values: %s", strings.Join(allowedCodes, ", "))).
								WithLocation(&errors.Location{
									SegmentTag:   ps.Schema.ID,
									ElementIndex: elemSchema.Position,
								}).Build())
						}
					}
				}
			}
		}
	}

	if p.customerOverlay != nil {
		if overlay, exists := p.customerOverlay.SegmentRules[ps.Schema.ID]; exists {
			for _, rule := range overlay.CustomRules {
				if !p.evaluateRule(ps, rule) {
					errs = append(errs, *errors.NewError(
						errors.ErrorTypeBusinessRule,
						fmt.Sprintf("CUSTOM_%s", rule.Type),
						rule.Message,
					).WithSeverity(p.getSeverity(rule.Severity)).Build())
				}
			}
		}
	}

	for _, validator := range p.validators {
		errs = append(errs, validator.Validate(ctx, ps)...)
	}

	return errs
}

// validateCrossSegments validates relationships between segments
func (p *SegmentProcessor) validateCrossSegments(
	ctx context.Context,
	segments []*ProcessedSegment,
) error {
	if p.customerOverlay == nil {
		return nil
	}

	for _, rule := range p.customerOverlay.Conditionals {
		if err := p.evaluateConditionalRule(segments, rule); err != nil {
			p.errorCollector.Add(errors.NewError(
				errors.ErrorTypeBusinessRule,
				rule.ID,
				err.Error(),
			).WithSeverity(p.getSeverity(rule.Severity)).Build())
		}
	}

	for _, loopReq := range p.customerOverlay.LoopRules {
		if err := p.validateLoopRequirement(segments, loopReq); err != nil {
			p.errorCollector.Add(errors.NewError(
				errors.ErrorTypeStructure,
				fmt.Sprintf("LOOP_%s", loopReq.LoopID),
				err.Error(),
			).Build())
		}
	}

	return nil
}

// Helper methods

func (p *SegmentProcessor) applyTransform(value any, transform string) any {
	valueStr := fmt.Sprintf("%v", value)

	switch {
	case transform == "uppercase":
		return strings.ToUpper(valueStr)
	case transform == "lowercase":
		return strings.ToLower(valueStr)
	case transform == "trim":
		return strings.TrimSpace(valueStr)
	case strings.HasPrefix(transform, "pad_left:"):
		parts := strings.Split(transform, ":")
		if len(parts) == 2 {
			return valueStr // Simplified
		}
	}

	return value
}

func (p *SegmentProcessor) evaluateRule(ps *ProcessedSegment, rule ValidationRule) bool {
	// Implement rule evaluation logic
	// This would check the rule conditions against segment data
	return true // Simplified
}

func (p *SegmentProcessor) evaluateConditionalRule(
	segments []*ProcessedSegment,
	rule ConditionalRule,
) error {
	// Find segments matching the condition
	conditionMet := false
	for _, seg := range segments {
		if seg.Schema.ID == rule.When.Segment {
			// Check condition
			key := fmt.Sprintf("%s%02d", rule.When.Segment, rule.When.Element)
			if value, exists := seg.Data[key]; exists {
				valueStr := fmt.Sprintf("%v", value)
				switch rule.When.Operator {
				case "equals":
					if valueStr == rule.When.Value {
						conditionMet = true
					}
				case "not_equals":
					if valueStr != rule.When.Value {
						conditionMet = true
					}
				case "contains":
					if strings.Contains(valueStr, rule.When.Value) {
						conditionMet = true
					}
				case "exists":
					conditionMet = true
				}
			}
		}
	}

	// If condition is met, check requirement
	if conditionMet {
		requirementMet := false
		for _, seg := range segments {
			if seg.Schema.ID == rule.Then.Segment {
				key := fmt.Sprintf("%s%02d", rule.Then.Segment, rule.Then.Element)
				value, exists := seg.Data[key]

				switch rule.Then.MustBe {
				case "present":
					if exists && value != "" {
						requirementMet = true
					}
				case "absent":
					if !exists || value == "" {
						requirementMet = true
					}
				case "equal_to":
					if exists && fmt.Sprintf("%v", value) == rule.Then.Value {
						requirementMet = true
					}
				}
			}
		}

		if !requirementMet {
			return fmt.Errorf("%s: condition met but requirement not satisfied", rule.Description)
		}
	}

	return nil
}

func (p *SegmentProcessor) validateLoopRequirement(
	segments []*ProcessedSegment,
	req LoopRequirement,
) error {
	// Count loop occurrences
	loopCount := 0
	for _, seg := range segments {
		if seg.Position.LoopID == req.LoopID {
			loopCount++
		}
	}

	if req.Required && loopCount == 0 {
		return fmt.Errorf("required loop %s not found", req.LoopID)
	}

	if req.MinOccurs > 0 && loopCount < req.MinOccurs {
		return fmt.Errorf(
			"loop %s occurs %d times, minimum %d required",
			req.LoopID,
			loopCount,
			req.MinOccurs,
		)
	}

	if req.MaxOccurs > 0 && loopCount > req.MaxOccurs {
		return fmt.Errorf(
			"loop %s occurs %d times, maximum %d allowed",
			req.LoopID,
			loopCount,
			req.MaxOccurs,
		)
	}

	return nil
}

func (p *SegmentProcessor) getSeverity(sev string) errors.Severity {
	switch strings.ToLower(sev) {
	case "info":
		return errors.SeverityInfo
	case "warning", "warn":
		return errors.SeverityWarning
	case "fatal":
		return errors.SeverityFatal
	default:
		return errors.SeverityError
	}
}

// positionTracker tracks position within EDI structure
type positionTracker struct {
	loops       map[string]int
	currentLoop string
	parentLoop  string
	transaction string
	functional  string
	interchange string
}

func (t *positionTracker) update(seg x12.Segment) {
	tag := strings.ToUpper(seg.Tag)

	switch tag {
	case "ISA":
		if len(seg.Elements) >= 13 && len(seg.Elements[12]) > 0 {
			t.interchange = seg.Elements[12][0]
		} else {
			t.interchange = "UNKNOWN"
		}
		t.currentLoop = ""
	case "GS":
		if len(seg.Elements) >= 6 && len(seg.Elements[5]) > 0 {
			t.functional = seg.Elements[5][0]
		} else {
			t.functional = "UNKNOWN"
		}
		t.currentLoop = ""
	case "ST":
		if len(seg.Elements) >= 2 && len(seg.Elements[1]) > 0 {
			t.transaction = seg.Elements[1][0]
		} else {
			t.transaction = "UNKNOWN"
		}
		t.currentLoop = ""
	case "N1": // Party loop starter
		t.currentLoop = "N1"
		if _, exists := t.loops["N1"]; !exists {
			t.loops["N1"] = 0
		}
		t.loops["N1"]++
	case "N3", "N4", "G61": // Segments that belong to N1 loop
		// Keep current loop as N1 if we're in it
		if t.currentLoop != "N1" && t.currentLoop != "S5" {
			// These can appear in N1 or S5 loops
		}
	case "S5": // Stop loop starter
		t.currentLoop = "S5"
		if _, exists := t.loops["S5"]; !exists {
			t.loops["S5"] = 0
		}
		t.loops["S5"]++
	case "G62": // Date/Time - can be in S5 loop
		// Keep current loop
	case "SE", "GE", "IEA": // End markers
		t.currentLoop = ""
	case "LS": // Explicit loop start
		if len(seg.Elements) >= 1 && len(seg.Elements[0]) > 0 {
			t.parentLoop = t.currentLoop
			t.currentLoop = seg.Elements[0][0]
			t.loops[t.currentLoop]++
		}
	case "LE": // Loop end
		t.currentLoop = t.parentLoop
		t.parentLoop = ""
	}
}

