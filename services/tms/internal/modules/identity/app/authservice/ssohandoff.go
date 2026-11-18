package authservice

import (
	"net/url"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
)

func ssoStartHandoff(
	redirectURL string,
	req *tenantports.StartSSOLoginRequest,
	allowedOrigins []string,
) (string, bool, error) {
	if err := validateReturnToURL(req.ReturnTo, allowedOrigins); err != nil {
		return "", false, err
	}
	handoff, ok := ssoHandoffURL(redirectURL, req.ReturnTo, req.StartPath, allowedOrigins)

	return handoff, ok, nil
}

func ssoHandoffURL(
	redirectURL string,
	returnTo string,
	startPath string,
	allowedOrigins []string,
) (string, bool) {
	target, ok := originOf(redirectURL)
	if !ok || !slices.Contains(allowedOrigins, target) {
		return "", false
	}
	from, ok := originOf(returnTo)
	if !ok || from == target || !plainPath(startPath) {
		return "", false
	}
	query := url.Values{"returnTo": {returnToOn(target, returnTo)}}

	return target + startPath + "?" + query.Encode(), true
}

func originOf(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	return parsed.Scheme + "://" + parsed.Host, true
}

func returnToOn(origin, returnTo string) string {
	parsed, err := url.Parse(strings.TrimSpace(returnTo))
	if err != nil {
		return origin + "/"
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	if !plainPath(path) {
		return origin + "/"
	}
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}

	return origin + path
}

func plainPath(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") &&
		!strings.Contains(path, `\`)
}
