import type { TriageExceptionKind } from "@/lib/graphql/inbox";

/**
 * A delivery exception as the words a driver or a BOL put it — "2 pallets
 * short", "1 case damaged" — read into the parts the exception is logged
 * with. What cannot be read is left for the person to fill in.
 */
export type ExceptionReading = {
  kind: TriageExceptionKind | null;
  quantity: number | null;
  /** "pallets", "cases"; blank when the words do not say. */
  unit: string;
};

const KINDS: ReadonlyArray<[RegExp, TriageExceptionKind]> = [
  [/\bshort(?:age|ed|-shipped)?\b/i, "Shortage"],
  [/\bover(?:age)?\b/i, "Overage"],
  [/\bdamage[ds]?\b/i, "Damage"],
];
const COUNT = /\b(\d{1,6})\b(?:\s+([a-z]+))?/i;
const KIND_WORDS = /^(short|shortage|over|overage|damaged?|in|of)$/i;

export function readException(text: string): ExceptionReading {
  const kind = KINDS.find(([pattern]) => pattern.test(text))?.[1] ?? null;
  const count = COUNT.exec(text);
  const quantity = count === null ? null : Number(count[1]);
  const word = count?.[2] ?? "";
  const unit = word === "" || KIND_WORDS.test(word) ? "" : word.toLowerCase();

  return { kind, quantity, unit };
}
