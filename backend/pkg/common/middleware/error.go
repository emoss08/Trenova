package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIErrorResponse struct {
	Type    string       `json:"type"`
	Message string       `json:"message,omitempty"` // User-friendly error message
	Errors  []FieldError `json:"errors"`
}

type FieldError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Attr   string `json:"attr,omitempty"` // Attribute name, if applicable
}

func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // process request

		if len(c.Errors) > 0 {
			var fieldErrors []FieldError

			for _, e := range c.Errors {
				log.Printf("Error: %v", e.Err) // Logging the error for internal monitoring

				// Error response for the client
				fieldError := FieldError{
					Code:   "InternalError",
					Detail: "An internal error occurred.",
				}

				// If the error is a binding error, we can customize the error response
				if e.Type == gin.ErrorTypeBind {
					fieldError.Code = "ValidationError"
					fieldError.Detail = e.Err.Error()
					fieldError.Attr = e.Meta.(string)
				}

				// Append the error to the list of errors
				fieldErrors = append(fieldErrors, fieldError)
			}

			c.JSON(http.StatusBadRequest, APIErrorResponse{
				Type:    "error",
				Message: "Request could not be processed.",
				Errors:  fieldErrors,
			})
			c.Abort()
			return
		}
	}
}
