package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/shared/edi/internal/transport"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-playground/validator/v10"
)

// Validator interface for request validation
type Validator interface {
	Validate(ctx context.Context, request interface{}) error
}

// StructValidator implements Validator using go-playground/validator
type StructValidator struct {
	validator *validator.Validate
}

// NewStructValidator creates a new struct validator
func NewStructValidator() *StructValidator {
	v := validator.New()
	
	// Register custom validations
	v.RegisterValidation("partner_id", validatePartnerID)
	v.RegisterValidation("edi_content", validateEDIContent)
	
	return &StructValidator{
		validator: v,
	}
}

// Validate validates a struct
func (v *StructValidator) Validate(ctx context.Context, request interface{}) error {
	if err := v.validator.Struct(request); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// Custom validation functions
func validatePartnerID(fl validator.FieldLevel) bool {
	partnerID := fl.Field().String()
	// Partner ID must be alphanumeric with optional hyphens/underscores
	if len(partnerID) < 3 || len(partnerID) > 50 {
		return false
	}
	for _, ch := range partnerID {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || 
			 (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func validateEDIContent(fl validator.FieldLevel) bool {
	content := fl.Field().String()
	// Basic EDI validation - must start with ISA
	return len(content) > 0 && strings.HasPrefix(content, "ISA")
}

// formatValidationError converts validator errors to our error type
func formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		fieldErrors := make([]transport.FieldError, 0, len(validationErrors))
		
		for _, e := range validationErrors {
			fieldName := strings.ToLower(e.Field())
			var message string
			
			switch e.Tag() {
			case "required":
				message = fmt.Sprintf("%s is required", fieldName)
			case "min":
				message = fmt.Sprintf("%s must be at least %s characters", fieldName, e.Param())
			case "max":
				message = fmt.Sprintf("%s must be at most %s characters", fieldName, e.Param())
			case "partner_id":
				message = "invalid partner ID format"
			case "edi_content":
				message = "invalid EDI content"
			default:
				message = fmt.Sprintf("%s failed %s validation", fieldName, e.Tag())
			}
			
			fieldErrors = append(fieldErrors, transport.FieldError{
				Field:   fieldName,
				Message: message,
				Code:    e.Tag(),
			})
		}
		
		return transport.ValidationError{Errors: fieldErrors}
	}
	
	return transport.NewServiceError(transport.ErrorTypeValidation, err.Error())
}

// ValidationMiddleware creates an endpoint middleware for request validation
func ValidationMiddleware(v Validator) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			if err := v.Validate(ctx, request); err != nil {
				return nil, err
			}
			return next(ctx, request)
		}
	}
}

// ContentLengthValidation validates request size
func ContentLengthValidation(maxBytes int64) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			// Check if request has content length info
			if req, ok := request.(interface{ GetContentLength() int64 }); ok {
				if req.GetContentLength() > maxBytes {
					return nil, transport.NewServiceError(
						transport.ErrorTypeValidation,
						fmt.Sprintf("request size exceeds maximum of %d bytes", maxBytes),
					)
				}
			}
			return next(ctx, request)
		}
	}
}

// BusinessRuleValidation applies business-specific validation rules
func BusinessRuleValidation() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			// Apply business rules based on request type
			switch req := request.(type) {
			case interface{ GetPartnerID() string }:
				// Validate partner exists, is active, etc.
				partnerID := req.GetPartnerID()
				if partnerID == "" {
					return nil, transport.NewServiceError(
						transport.ErrorTypeValidation,
						"partner ID is required",
					).WithField("partner_id")
				}
				
			case interface{ GetEDIContent() string }:
				// Validate EDI content structure
				content := req.GetEDIContent()
				if len(content) < 100 {
					return nil, transport.NewServiceError(
						transport.ErrorTypeValidation,
						"EDI content appears to be incomplete",
					)
				}
			}
			
			return next(ctx, request)
		}
	}
}