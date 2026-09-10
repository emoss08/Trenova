package helpers

import (
	"bytes"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const defaultBodyBufferBytes = 512

type RequestTooLargeError struct {
	Limit int64
}

func NewRequestTooLargeError(limit int64) *RequestTooLargeError {
	return &RequestTooLargeError{Limit: limit}
}

func (e *RequestTooLargeError) Error() string {
	return "request body exceeds the " + strconv.FormatInt(e.Limit, 10) + " byte limit"
}

func ReadBoundedRequestBody(c *gin.Context, limit int64) ([]byte, error) {
	if c.Request.ContentLength > limit {
		return nil, NewRequestTooLargeError(limit)
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)

	buf := bytes.NewBuffer(
		make([]byte, 0, bodyBufferCapacity(c.Request.ContentLength, limit)),
	)
	if _, err := buf.ReadFrom(c.Request.Body); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, NewRequestTooLargeError(limit)
		}
		return nil, err
	}

	return buf.Bytes(), nil
}

func IsRequestTooLargeError(err error) bool {
	var tooLargeErr *RequestTooLargeError
	if errors.As(err, &tooLargeErr) {
		return true
	}

	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func bodyBufferCapacity(contentLength, limit int64) int {
	if contentLength <= 0 || contentLength > limit || contentLength > math.MaxInt {
		return defaultBodyBufferBytes
	}

	return int(contentLength)
}
