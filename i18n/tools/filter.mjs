// filter.mjs decides whether a captured literal is text a human reads.
//
// One implementation serves the TypeScript extractor, the Go extractor's output and the
// codemod, so a string can never be translated in one pass and skipped in another.
//
// Every rejection carries a reason. `i18n report --rejected` prints them grouped, which is
// how the denylist gets audited instead of trusted — a filter nobody can inspect quietly
// deletes UI strings from the product.

// Props whose string values are prose, derived by frequency-counting real attribute usage
// across apps/web rather than guessed.
export const TEXT_PROPS = new Set([
  "label", "description", "placeholder", "title", "subtitle", "heading", "caption",
  "loadingText", "hint", "help", "helper", "message", "successMessage", "errorMessage",
  "emptyMessage", "emptyLabel", "empty", "noResultsMessage", "searchPlaceholder",
  "confirmLabel", "confirmText", "cancelLabel", "cancelText", "submitLabel",
  "saveButtonContent", "hrefLabel", "sideText", "tooltip", "alt", "ariaLabel", "aria-label",
  "summary", "detail", "footer", "body", "text", "sub",
]);

// Props that look textual but carry identifiers, enum wire values or vector data. `d` is an
// SVG path; `name` is a form field key bound to a schema, not a caption.
export const NEVER_TEXT_PROPS = new Set([
  "d", "name", "currency", "status", "category", "type", "variant", "size", "value", "id",
  "key", "href", "src", "to", "path", "className", "class", "role", "target", "rel",
  "stopType", "targetStatus", "method", "mode", "align", "side", "position", "color",
  "icon", "format", "locale", "timezone", "accept", "pattern", "autoComplete", "testId",
  "data-testid",
]);

const URL_LIKE = /^(https?:\/\/|\/|\.\/|\.\.\/|@\/|mailto:|tel:)/;
const SCREAMING = /^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$/;
const SINGLE_LOWER_TOKEN = /^[a-z][a-z0-9]*$/;
const CAMEL_TOKEN = /^[a-z][a-zA-Z0-9]*$/;
const HAS_LETTER = /\p{L}/u;
// date-fns / Intl format patterns: only pattern letters, separators and digits.
const DATE_PATTERN = /^[yMdHhmsaGEwWDFkKSzZX\/\-.,:'\s]+$/;
const TAILWIND_TOKEN = /^[a-z0-9:\-\/\[\]().%#\s]+$/;

// reject returns a reason string when the value is not human-readable prose, else null.
/**
 * reject returns a reason string when the value is not human-readable prose, else null.
 *
 * `positional` marks a string the extractor found in a place that is user-facing by
 * construction — a message argument, an enum's Label() return. There the identifier-shaped
 * heuristics do more harm than good: "suspension" and "termination" are real disciplinary
 * labels, not variable names, and only their position can tell them apart.
 */
export function reject(value, { prop = null, positional = false } = {}) {
  if (prop !== null && NEVER_TEXT_PROPS.has(prop)) return "non-text prop";

  const text = value.trim();
  if (text === "") return "blank";
  if (!HAS_LETTER.test(text)) return "no letters";
  if (text.length === 1) return "single character";
  if (URL_LIKE.test(text)) return "url or path";
  // Underscored or long all-caps tokens are wire values (ACTIVE, PENDING_REVIEW). Short
  // ones are legitimate labels and must survive: OK, ID, PO, MC, BOL.
  if (!positional && SCREAMING.test(text) && (text.includes("_") || text.length >= 4)) {
    return "enum wire value";
  }

  const words = text.split(/\s+/);

  if (words.length === 1 && !positional) {
    // A lone identifier-shaped token is a variant, key or slug. A lone capitalised word
    // ("Shipments", "Draft") is a real label and must survive.
    if (SINGLE_LOWER_TOKEN.test(text)) return "lowercase identifier";
    if (CAMEL_TOKEN.test(text)) return "camelCase identifier";
    if (text.includes("-") && text === text.toLowerCase()) return "slug";
    if (text.includes("_")) return "identifier with underscore";
  }

  // Checked after the single-token rules so a bare "sm" is reported as the identifier it is
  // rather than as a date pattern that happens to be spelled from pattern letters.
  if (DATE_PATTERN.test(text) && /[\/\-.:]/.test(text)) return "date format pattern";

  // A class list is all-lowercase, carries at least one utility-shaped token
  // ("items-center", "md:flex", "w-[32px]") and never sentence punctuation. Prose stays
  // clear of it: "bill of lading" is lowercase but has no such token.
  if (!positional && words.length > 1 && text === text.toLowerCase() && TAILWIND_TOKEN.test(text)) {
    const hasUtilityToken = words.some((w) => /[-:\/\[]/.test(w));
    const hasSentencePunctuation = /[.,!?;]/.test(text);
    if (hasUtilityToken && !hasSentencePunctuation) return "css class list";
  }

  // Go internals. These describe infrastructure faults to an operator and are deliberately
  // kept in English so logs and support tickets stay searchable.
  if (/^failed to /.test(text)) return "internal go error";

  return null;
}

export function accept(value, opts) {
  return reject(value, opts) === null;
}
