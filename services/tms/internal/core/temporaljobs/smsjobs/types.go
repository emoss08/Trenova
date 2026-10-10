package smsjobs

type SendSMSResult struct {
	MessageID string `json:"messageId,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}
