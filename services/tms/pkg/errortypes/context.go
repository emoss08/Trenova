package errortypes

type ErrorContext struct {
	RequestID string            `json:"requestId,omitempty"`
	UserID    string            `json:"userId,omitempty"`
	TraceID   string            `json:"traceId,omitempty"`
	SpanID    string            `json:"spanId,omitempty"`
	Extra     map[string]string `json:"extra,omitempty"`
}

func NewErrorContext() *ErrorContext {
	return &ErrorContext{}
}

func (c *ErrorContext) WithRequestID(id string) *ErrorContext {
	c.RequestID = id
	return c
}

func (c *ErrorContext) WithUserID(id string) *ErrorContext {
	c.UserID = id
	return c
}

func (c *ErrorContext) WithTraceID(id string) *ErrorContext {
	c.TraceID = id
	return c
}

func (c *ErrorContext) WithSpanID(id string) *ErrorContext {
	c.SpanID = id
	return c
}

func (c *ErrorContext) WithExtra(key, value string) *ErrorContext {
	if c.Extra == nil {
		c.Extra = make(map[string]string)
	}

	c.Extra[key] = value
	return c
}

type LogFields map[string]any

func (c *ErrorContext) LogFields() LogFields {
	fields := make(LogFields)
	if c.RequestID != "" {
		fields["request_id"] = c.RequestID
	}

	if c.UserID != "" {
		fields["user_id"] = c.UserID
	}

	if c.TraceID != "" {
		fields["trace_id"] = c.TraceID
	}

	if c.SpanID != "" {
		fields["span_id"] = c.SpanID
	}

	for k, v := range c.Extra {
		fields[k] = v
	}
	return fields
}
