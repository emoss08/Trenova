import type { AskField, AskInput } from "@/components/assistant/ask-requests";
import { mentionToken } from "@/components/assistant/composer-types";
import type { AssistantEntityRef } from "@/types/assistant";
import { fromISODateString } from "@trenova/shared/lib/date";

/**
 * What a person has entered for one value: the text of a box (words, a
 * number, or a day or moment as the browser's control gives it) or the record
 * they picked. Null is nothing entered.
 */
export type AskValue = string | AssistantEntityRef | null;

/** An answer as it leaves: the message's words, and the records they name. */
export type AskAnswer = { text: string; mentions: AssistantEntityRef[] };

/** One value in the words the model reads, and the record it names when it is one. */
type AskValueText = { text: string; mention: AssistantEntityRef | null };

const DATETIME_LOCAL = /^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2})(?::\d{2}(?:\.\d+)?)?$/;

/**
 * A value as the model reads it, or null while it is empty or not yet a value
 * of its type.
 *
 * A day is `YYYY-MM-DD` and a moment `YYYY-MM-DD HH:mm` on the person's own
 * clock, which is what a date and a datetime-local control already hold. A
 * number is kept as typed, so "1,200" is not silently read as 1. A record is
 * written the way the composer writes an @ mention, and the record goes with
 * it, so the model reads its id exactly as it does for a mention.
 */
function askValueText(input: AskInput, value: AskValue): AskValueText | null {
  if (value === null) {
    return null;
  }
  if (typeof value !== "string") {
    if (input.type !== "record" || value.id === "") {
      return null;
    }
    const label = value.label !== "" ? value.label : value.id;
    return { text: mentionToken(label), mention: { ...value, label } };
  }

  const text = value.trim();
  if (text === "") {
    return null;
  }
  switch (input.type) {
    case "number":
      return Number.isFinite(Number(text)) ? { text, mention: null } : null;
    case "date":
      return fromISODateString(text) === null ? null : { text, mention: null };
    case "datetime": {
      const match = DATETIME_LOCAL.exec(text);
      return match && fromISODateString(match[1]) !== null
        ? { text: `${match[1]} ${match[2]}`, mention: null }
        : null;
    }
    case "record":
      return null;
    default:
      return { text, mention: null };
  }
}

/** Whether a value is entered and of its type. */
export function askValueReady(input: AskInput, value: AskValue): boolean {
  return askValueText(input, value) !== null;
}

/** The answer to a question that takes one value of the person's own, or null until it is one. */
export function askInputAnswer(input: AskInput, value: AskValue): AskAnswer | null {
  const entered = askValueText(input, value);
  return entered === null
    ? null
    : { text: entered.text, mentions: entered.mention ? [entered.mention] : [] };
}

/**
 * The answer to a small form: one line per value entered, `Label: value`, in
 * the form's order. An optional value left empty is left out; null until every
 * value the form needs is entered, and for a form with nothing entered at all.
 */
export function askFieldsAnswer(
  fields: readonly AskField[],
  values: Readonly<Record<string, AskValue>>,
): AskAnswer | null {
  const lines: string[] = [];
  const mentions: AssistantEntityRef[] = [];

  for (const field of fields) {
    const entered = askValueText(field, values[field.key] ?? null);
    if (entered === null) {
      if (field.optional) {
        continue;
      }
      return null;
    }
    lines.push(`${field.label}: ${entered.text}`);
    const mention = entered.mention;
    if (mention && !mentions.some((m) => m.type === mention.type && m.id === mention.id)) {
      mentions.push(mention);
    }
  }

  return lines.length === 0 ? null : { text: lines.join("\n"), mentions };
}
