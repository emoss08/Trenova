package variable

import "errors"

var (
	ErrQueryCannotBeEmpty               = errors.New("query cannot be empty")
	ErrQueryMustBeSelect                = errors.New("query must be a select statement")
	ErrSemicolonsNotAllowed             = errors.New("semicolons are not allowed in queries")
	ErrCommentsNotAllowed               = errors.New("comments are not allowed in queries")
	ErrNEXTVALNotAllowed                = errors.New("NEXTVAL is not allowed")
	ErrInvalidSubquery                  = errors.New("invalid subquery: expected SELECT statement")
	ErrFormatSQLCannotBeEmpty           = errors.New("format SQL cannot be empty")
	ErrFormatSemicolonsNotAllowed       = errors.New("semicolons are not allowed in format SQL")
	ErrFormatCommentsNotAllowed         = errors.New("comments are not allowed in format SQL")
	ErrUnbalancedParentheses            = errors.New("unbalanced parentheses in format SQL")
	ErrSubqueriesNotAllowed             = errors.New("subqueries are not allowed in format SQL")
	ErrFormatValuePlaceholderNotAllowed = errors.New("format SQL must contain :value placeholder")
)
