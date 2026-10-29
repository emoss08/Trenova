package inboundreplyservice

import (
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundreplyports"
)

const (
	placeholderFirst    = "first"
	placeholderPro      = "pro"
	placeholderStatus   = "status"
	placeholderETA      = "eta"
	placeholderLane     = "lane"
	placeholderCustomer = "customer"
	placeholderPO       = "po"
	placeholderSender   = "sender"
)

var placeholderPattern = regexp.MustCompile(`\{([A-Za-z]+)\}`)

var placeholderFallbacks = map[string]string{
	placeholderFirst:    "there",
	placeholderPro:      "your load",
	placeholderStatus:   "moving",
	placeholderETA:      "to follow",
	placeholderLane:     "the load",
	placeholderCustomer: "your team",
	placeholderPO:       "your order",
	placeholderSender:   "you",
}

func RenderTemplate(body string, values *inboundreplyports.ReplyContext) string {
	known := map[string]string{
		placeholderFirst:    values.SenderFirstName,
		placeholderPro:      values.ProNumber,
		placeholderStatus:   values.Status,
		placeholderETA:      values.ETA,
		placeholderLane:     values.Lane,
		placeholderCustomer: values.CustomerName,
		placeholderPO:       values.PONumber,
		placeholderSender:   values.SenderAddress,
	}

	return placeholderPattern.ReplaceAllStringFunc(body, func(match string) string {
		key := strings.ToLower(match[1 : len(match)-1])
		value, ok := known[key]
		if !ok {
			return match
		}
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}

		return placeholderFallbacks[key]
	})
}
