package rptmetavalidator

import (
	"regexp"
	"strings"

	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/rptmeta"
	"go.uber.org/fx"
)

const (
	maxSQLLength = 10000 // Maximum length of the SQL query
)

// SQL injection patterns to check for
var sqlInjectionPatterns = []string{
	`(?i);\s*DROP\s+TABLE`,
	`(?i);\s*DELETE\s+FROM`,
	`(?i);\s*UPDATE\s+.*\s+SET`,
	`(?i);\s*INSERT\s+INTO`,
	`(?i)EXEC\s*\(`,
	`(?i)EXECUTE\s*\(`,
	`(?i)UNION\s+ALL\s+SELECT`,
	`(?i)UNION\s+SELECT`,
}

type MetadataValidatorParams struct {
	fx.In

	VariableValidator *VariableValidator
}

type MetadataValidator struct {
	variableValidator *VariableValidator
}

func NewMetadataValidator(p MetadataValidatorParams) *MetadataValidator {
	return &MetadataValidator{
		variableValidator: p.VariableValidator,
	}
}

func (mv *MetadataValidator) Validate(metadata *rptmeta.Metadata) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Core validation
	metadata.Validate(multiErr)

	// SQL structure check
	mv.validateSQLStructure(metadata.SQL, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (mv *MetadataValidator) validateSQLStructure(sql string, multiErr *errors.MultiError) {
	if len(sql) > maxSQLLength {
		multiErr.Add("sql", errors.ErrInvalid, "SQL query exceeds maximum length")
	}

	// Check for basic SQL structure
	if !strings.Contains(strings.ToUpper(sql), "SELECT") {
		multiErr.Add("sql", errors.ErrInvalid, "SQL query must contain a SELECT statement")
	}

	if !hasBalancedParentheses(sql) {
		multiErr.Add("sql", errors.ErrInvalid, "SQL query has unbalanced parentheses")
	}

	// SQL injection check
	mv.checkSQLInjection(sql, multiErr)
}

func (mv *MetadataValidator) checkSQLInjection(sql string, multiErr *errors.MultiError) {
	for _, pattern := range sqlInjectionPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(sql) {
			multiErr.Add("sql", errors.ErrInvalid, "Potential SQL injection detected")
		}
	}
}

func hasBalancedParentheses(sql string) bool {
	count := 0
	for _, char := range sql {
		switch char {
		case '(':
			count++
		case ')':
			count--
			if count < 0 {
				return false
			}
		}
	}
	return count == 0
}
