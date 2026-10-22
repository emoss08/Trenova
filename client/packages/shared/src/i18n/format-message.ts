// format-message.ts is a deliberate mirror of shared/i18n/format.go.
//
// The same catalog entry is rendered by the Go services and by the browser, so the two
// formatters have to agree exactly: a message that reads "3 shipments" from the API and
// "3 shipment" in the UI is a bug that no test on either side alone would catch. Keeping
// the supported syntax small — positional placeholders and plurals — is what makes that
// agreement checkable by reading both files.
import type { Locale } from "@trenova/shared/i18n/generated/locales";
import { intlTag } from "@trenova/shared/i18n/intl-tags";

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
    const char = text[i];
    if (char === "'") {
      const next = text[i + 1];
      if (next === "{" || next === "}") {
        const closing = text.indexOf("'", i + 1);
        if (closing >= 0) i = closing;
      }
    } else if (char === "{") depth++;
    else if (char === "}") {
      depth--;
      if (depth === 0) return i;
    }
  }
  return -1;
}

// ICU quoting, as format.go's writeQuoted: '' is one apostrophe, and an apostrophe before a
// brace quotes up to the next apostrophe, so '{' is a literal brace. Any other apostrophe is
// itself.
function readQuoted(text: string, i: number): [string, number] {
  const next = text[i + 1];
  if (next === "'") return ["'", i + 2];
  if (next === "{" || next === "}") {
    const closing = text.indexOf("'", i + 1);
    if (closing < 0) return [text.slice(i + 1), text.length];
    return [text.slice(i + 1, closing), closing + 1];
  }
  return ["'", i + 1];
}

function unquote(text: string): string {
  if (!text.includes("'")) return text;
  let out = "";
  for (let i = 0; i < text.length; ) {
    if (text[i] === "'") {
      const [literal, next] = readQuoted(text, i);
      out += literal;
      i = next;
      continue;
    }
    out += text[i];
    i++;
  }
  return out;
}

function stringify(value: unknown): string {
  if (typeof value === "string") return value;
  if (value === null || value === undefined) return "";
  if (
    typeof value === "number" ||
    typeof value === "boolean" ||
    typeof value === "bigint"
  ) {
    return String(value);
  }
  if (value instanceof Date) return value.toISOString();
  // An object reaching a placeholder is a caller mistake; "[object Object]" in the UI hides
  // it, so serialize enough to make it obvious in a screenshot or a bug report.
  return JSON.stringify(value) ?? "";
}

// A plural's # is the count as the locale writes numbers, grouped and with at most three
// decimals, as ICU does; format.go's formatCount writes the same digits.
const countFormatters = new Map<Locale, Intl.NumberFormat>();

function formatCount(locale: Locale, value: number): string {
  let formatter = countFormatters.get(locale);
  if (formatter === undefined) {
    formatter = new Intl.NumberFormat(intlTag(locale), {
      maximumFractionDigits: 3,
    });
    countFormatters.set(locale, formatter);
  }
  return formatter.format(value);
}

function pluralBranch(forms: string, want: PluralForm): string | null {
  let fallback: string | null = null;

  for (let i = 0; i < forms.length; ) {
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

function renderPlaceholder(
  locale: Locale,
  body: string,
  args: unknown[],
): string | null {
  const comma = body.indexOf(",");
  const indexPart = comma === -1 ? body : body.slice(0, comma);

  const index = Number.parseInt(indexPart.trim(), 10);
  if (!Number.isInteger(index) || index < 0 || index >= args.length)
    return null;

  const value = args[index];
  if (comma === -1) return stringify(value);

  const rest = body.slice(comma + 1).trim();
  const kindEnd = rest.indexOf(",");
  if (kindEnd === -1 || rest.slice(0, kindEnd).trim() !== "plural")
    return stringify(value);
  if (typeof value !== "number" || !Number.isFinite(value))
    return stringify(value);

  const chosen = pluralBranch(
    rest.slice(kindEnd + 1),
    selectPlural(locale, value),
  );
  if (chosen === null) return stringify(value);

  return unquote(chosen).split("#").join(formatCount(locale, value));
}

export function formatMessage(
  locale: Locale,
  message: string,
  args: unknown[],
): string {
  const quoted = message.includes("'");
  if (!quoted && (args.length === 0 || !message.includes("{"))) return message;

  let out = "";
  for (let i = 0; i < message.length; ) {
    if (message[i] === "'") {
      const [literal, next] = readQuoted(message, i);
      out += literal;
      i = next;
      continue;
    }
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
