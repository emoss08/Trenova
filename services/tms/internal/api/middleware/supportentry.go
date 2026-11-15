package middleware

// SupportEntryPath is where the control plane's admin posts a Trenova staff member's support
// assertion. The post comes from the control plane's origin by design, so the browser
// provenance guard lets it through; it carries no cookie the guard protects, and the assertion
// in its body is the whole of its authority (managed-mode.md, "Support access").
const SupportEntryPath = "/api/v1/support/enter"

func isSupportEntry(path string) bool {
	return path == SupportEntryPath
}
