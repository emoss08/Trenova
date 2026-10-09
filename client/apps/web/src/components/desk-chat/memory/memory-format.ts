import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** A calendar day in the person's zone, for telling today and yesterday apart. */
function dayOf(seconds: number, timezone: string): string {
  return new Date(seconds * 1000).toLocaleDateString("en-CA", { timeZone: timezone });
}

/** "Sep 18", in the person's zone and in English, as the design writes dates. */
function shortDate(seconds: number, timezone: string): string {
  return new Date(seconds * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    timeZone: timezone,
  });
}

/** Today, Yesterday, or the date: when something happened, in a word where one will do. */
export function memoryDay(
  seconds: number,
  timezone: string,
  t: TranslateFn,
  nowSeconds = Math.floor(Date.now() / 1000),
): string {
  const day = dayOf(seconds, timezone);
  if (day === dayOf(nowSeconds, timezone)) {
    return t("Today");
  }
  if (day === dayOf(nowSeconds - 86_400, timezone)) {
    return t("Yesterday");
  }
  return shortDate(seconds, timezone);
}

type SourceOf = {
  source: string;
  sourceTitle: string;
  scope: string;
};

/**
 * Where a memory came from, in a few words: the conversation it was saved
 * from when there is one, otherwise how it was recorded.
 */
export function memorySource(memory: SourceOf, t: TranslateFn): string {
  if (memory.sourceTitle !== "") {
    return memory.sourceTitle;
  }
  switch (memory.source) {
    case "User":
      return memory.scope === "User" ? t("Added by you") : t("Added by hand");
    case "Decision":
      return t("A decision on a change");
    case "Feedback":
      return t("Ratings of an agent's work");
    case "Reflection":
      return t("An agent looking back over its work");
    default:
      return t("A conversation");
  }
}

/** "Saved Sep 18 from “Acme rate review”". */
export function savedLine(
  memory: SourceOf & { createdAt: number },
  timezone: string,
  t: TranslateFn,
  nowSeconds?: number,
): string {
  return t(
    "Saved {0} from “{1}”",
    memoryDay(memory.createdAt, timezone, t, nowSeconds),
    memorySource(memory, t),
  );
}

/** "Used 14× · last today", "Paused", or "Not used yet". */
export function usageLine(
  memory: { status: string; useCount: number; lastUsedAt?: number | null },
  timezone: string,
  t: TranslateFn,
  nowSeconds?: number,
): string {
  if (memory.status === "Paused") {
    return t("Paused");
  }
  if (memory.useCount === 0 || !memory.lastUsedAt) {
    return t("Not used yet");
  }
  const when = memoryDay(memory.lastUsedAt, timezone, t, nowSeconds);
  const lower = when === t("Today") || when === t("Yesterday") ? when.toLowerCase() : when;

  return t("Used {0}× · last {1}", memory.useCount, lower);
}

/** Who a memory is kept for, in the page's words. */
export function scopeLabel(
  memory: { scope: string; roleName?: string | null },
  t: TranslateFn,
): string {
  switch (memory.scope) {
    case "User":
      return t("Just you");
    case "Role":
      return memory.roleName || t("Your team");
    case "Agent":
      return t("Everyone using this agent");
    default:
      return t("Organization");
  }
}

/**
 * Who a memory is kept for, as the one word a sentence names them by: "just
 * you", the team's name, or "everyone". At the start of a line it opens with a
 * capital: "Just you", "Everyone".
 */
export function scopeWord(
  memory: { scope: string; roleName?: string | null },
  t: TranslateFn,
  opening = false,
): string {
  switch (memory.scope) {
    case "User":
      return opening ? t("Just you") : t("just you");
    case "Role":
      return memory.roleName || (opening ? t("Your team") : t("your team"));
    case "Agent":
      return opening ? t("Everyone") : t("everyone");
    default:
      return opening ? t("Organization") : t("the whole organization");
  }
}

type WhyOf = {
  reason?: string | null;
  quotes?: readonly string[] | null;
  replaces?: { content: string } | null;
  replacedBy?: { content: string } | null;
};

export type MemoryWhyLine = {
  key: "reason" | "quote" | "replaces" | "replacedBy";
  label: string;
  text: string;
};

/** The most of what was said that a memory row quotes. */
const MAX_WHY_QUOTES = 2;

/**
 * Why a memory is kept and what it stands in for, as short labelled lines:
 * the reason an agent gave, a little of what was said, the memory it
 * replaced and the one that has since replaced it. A memory a person wrote
 * down has none of these, and shows nothing.
 */
export function memoryWhy(memory: WhyOf, t: TranslateFn): MemoryWhyLine[] {
  const lines: MemoryWhyLine[] = [];
  const reason = memory.reason?.trim() ?? "";
  if (reason !== "") {
    lines.push({ key: "reason", label: t("Why"), text: reason });
  }
  for (const quote of (memory.quotes ?? []).slice(0, MAX_WHY_QUOTES)) {
    if (quote.trim() !== "") {
      lines.push({ key: "quote", label: t("Said"), text: `“${quote.trim()}”` });
    }
  }
  if (memory.replaces) {
    lines.push({ key: "replaces", label: t("Replaces"), text: `“${memory.replaces.content}”` });
  }
  if (memory.replacedBy) {
    lines.push({
      key: "replacedBy",
      label: t("Replaced by"),
      text: `“${memory.replacedBy.content}”`,
    });
  }

  return lines;
}

const STEP_NUMBER = /\s*\d+\.\s+/;
const LEADING_STEP_NUMBER = /^\d+\.\s+/;

/**
 * A memory's steps: one per line when it is written on lines, otherwise split
 * at its numbers ("1. Search… 2. Then…"). Text that does not split comes back
 * as one entry, which reads as a paragraph rather than a list of one.
 */
export function splitSteps(content: string): string[] {
  const text = content.trim();
  if (text === "") {
    return [];
  }

  const parts = text.includes("\n")
    ? text.split(/\r?\n/).map((line) => line.trim().replace(LEADING_STEP_NUMBER, ""))
    : text.split(STEP_NUMBER);

  return parts.map((part) => part.trim()).filter((part) => part !== "");
}

/**
 * Steps edited one per line, written back the way the memory was written:
 * on lines when it was, numbered on one line when it was numbered, and as
 * plain text otherwise.
 */
export function joinSteps(lines: readonly string[], original: string): string {
  const steps = lines.map((line) => line.trim()).filter((line) => line !== "");
  if (original.includes("\n") || !STEP_NUMBER.test(original)) {
    return steps.join("\n");
  }

  return steps.map((step, index) => `${index + 1}. ${step}`).join(" ");
}
