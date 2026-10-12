package emailports

import "github.com/emoss08/trenova/internal/modules/workspace/domain/email"

const (
	PostmarkMessageLimitBytes = int64(10 * 1024 * 1024)
	ResendMessageLimitBytes   = int64(40 * 1024 * 1024)
	BodyAllowanceBytes        = int64(16 * 1024)
	smallestMessageLimitBytes = PostmarkMessageLimitBytes
	base64GroupBytes          = 3
	base64EncodedGroupBytes   = 4
)

const MaxAttachmentBytes = (smallestMessageLimitBytes - BodyAllowanceBytes) /
	base64EncodedGroupBytes * base64GroupBytes

func MessageLimitBytes(provider email.Provider) int64 {
	if provider == email.ProviderResend {
		return ResendMessageLimitBytes
	}

	return PostmarkMessageLimitBytes
}
