package cdc

import "errors"

var (
	ErrOperationFieldNotFound = errors.New("operation field not found or not a string")
	ErrSourceFieldNotFound    = errors.New("source field not found or not a map")
	ErrMessageTooShort        = errors.New("message too short for Avro format")
	ErrDecodeAvroMessage      = errors.New("failed to decode Avro message")
)
