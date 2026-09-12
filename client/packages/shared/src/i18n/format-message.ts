// format-message.ts is a deliberate mirror of shared/i18n/format.go.
//
// The same catalog entry is rendered by the Go services and by the browser, so the two
// formatters have to agree exactly: a message that reads "3 shipments" from the API and
// "3 shipment" in the UI is a bug that no test on either side alone would catch. Keeping
// the supported syntax small — positional placeholders and plurals — is what makes that
// agreement checkable by reading both files.
import type { Locale } from "@trenova/shared/i18n/generated/locales";

type PluralForm = "one" | "other";

// Chinese has a single plural form, so a count of 1 must NOT select the English one-form.
const SINGLE_FORM_LOCALES = new Set<Locale>(["zh-TW", "zh-CN"]);

function selectPlural(locale: Locale, count: number): PluralForm {
  if (SINGLE_FORM_LOCALES.has(locale)) return "other";
  return count === 1 ? "one" : "other";
}

function matchBrace(text: string, open: number): number {
  let depth = 0;
  for (let i = open; i < text.length; i++) {
    if (text[i] === "{") depth++;
    else if (text[i] === "}") {
      depth--;
      if (depth === 0) return i;
    }
  }
  return -1;
}

function stringify(value: unknown): string {
  if (typeof value === "string") return value;
  if (value === null || value === undefined) return "";
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  if (value instanceof Date) return value.toISOString();
  // An object reaching a placeholder is a caller mistake; "[object Object]" in the UI hides
  // it, so serialize enough to make it obvious in a screenshot or a bug report.
  return JSON.stringify(value) ?? "";
}

function pluralBranch(forms: string, want: PluralForm): string | null {
  let fallback: string | null = null;

  for (let i = 0; i < forms.length;) {
    const char = forms[i];
    if (char === " " || char === "\t" || char === "\n") {
      i++;
      continue;
    }

    const relativeOpen = forms.slice(i).indexOf("{");
    if (relativeOpen < 0) break;

    const name = forms.slice(i, i + relativeOpen).trim();
    const open = i + relativeOpen;
    const end = matchBrace(forms, open);
    if (end < 0) break;

    const body = forms.slice(open + 1, end);
    if (name === want) return body;
    if (name === "other") fallback = body;

    i = end + 1;
  }

  return fallback;
}

function renderPlaceholder(locale: Locale, body: string, args: unknown[]): string | null {
  const comma = body.indexOf(",");
  const indexPart = comma === -1 ? body : body.slice(0, comma);

  const index = Number.parseInt(indexPart.trim(), 10);
  if (!Number.isInteger(index) || index < 0 || index >= args.length) return null;

  const value = args[index];
  if (comma === -1) return stringify(value);

  const rest = body.slice(comma + 1).trim();
  const kindEnd = rest.indexOf(",");
  if (kindEnd === -1 || rest.slice(0, kindEnd).trim() !== "plural") return stringify(value);
  if (typeof value !== "number" || !Number.isFinite(value)) return stringify(value);

  const chosen = pluralBranch(rest.slice(kindEnd + 1), selectPlural(locale, value));
  if (chosen === null) return stringify(value);

  return chosen.split("#").join(stringify(value));
}

export function formatMessage(locale: Locale, message: string, args: unknown[]): string {
  if (args.length === 0 || !message.includes("{")) return message;

  let out = "";
  for (let i = 0; i < message.length;) {
    if (message[i] !== "{") {
      out += message[i];
      i++;
      continue;
    }

    const end = matchBrace(message, i);
    if (end < 0) {
      out += message[i];
      i++;
      continue;
    }

    const rendered = renderPlaceholder(locale, message.slice(i + 1, end), args);
    out += rendered === null ? message.slice(i, end + 1) : rendered;
    i = end + 1;
  }

  return out;
}
