/**
 * A navigation path in one spelling: a leading slash, no trailing one, and
 * null for an entry that leads nowhere ("", "#").
 */
export function normalizePath(path: string): string | null {
  if (!path || path === "#") {
    return null;
  }

  const trimmed = path.trim();
  if (!trimmed || trimmed === "#") {
    return null;
  }

  const withLeadingSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  const normalized = withLeadingSlash.replace(/\/+$/, "");

  return normalized === "" ? "/" : normalized;
}
