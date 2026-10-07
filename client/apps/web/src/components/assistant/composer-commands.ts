import { translate } from "@trenova/shared/i18n/runtime";
import type { Suggestion } from "./suggestions";

/**
 * A slot in a slash command: what goes there, in the words shown while it is
 * empty. The hint is the English source; whatever shows it translates it.
 */
export type CommandSlot = {
  name: string;
  hint: string;
};

/**
 * A slash command with typed slots. `prompt` writes the question the agent
 * actually receives, in the language on screen, so a person types
 * `/status S12345` and the agent is asked a full sentence about that
 * shipment. The description is the English source, translated where shown.
 */
export type SlashCommand = {
  name: string;
  description: string;
  slots: readonly CommandSlot[];
  prompt: (args: readonly string[]) => string;
  /**
   * Splits what follows the command into its slots, for a command whose
   * slots are not one word each. Without it, words fill the slots in order.
   */
  splitSlots?: (rest: string) => string[];
};

/**
 * What `\s` means in the cadence patterns: the same set the server reads
 * (conversationschedule.cadencePattern), with a no-break and an ideographic
 * space as well as ASCII whitespace.
 */
const SPACE_CLASS = "[\\t\\n\\f\\r \u00a0\u3000]";

function cadencePattern(source: string, flags: string): RegExp {
  return new RegExp(source.replaceAll(String.raw`\s`, SPACE_CLASS), flags);
}

/**
 * The cadences the server reads, one pattern per language, each the same
 * source as its Go twin in conversationschedule/cadence.go. Groups: 1 the
 * /schedule prefix, 2 the whole cadence, 3 its unit, 4 its time, 5 the
 * separator after it, 6 the request. Chinese opens ordinary questions with
 * 每天 too, so without the /schedule command a Chinese cadence counts only
 * with a time or a separator right after it.
 */
const SCHEDULE_GRAMMARS: readonly { pattern: RegExp; needsMarker: boolean }[] = [
  {
    pattern: cadencePattern(
      String.raw`^(/schedule\s+)?((?:every|each)\s+` +
        String.raw`(weekday|day|morning|monday|tuesday|wednesday|thursday|friday|saturday|sunday|week)\w*` +
        String.raw`(?:\s+at\s+([\d:]+(?:\s*(?:am|pm)\b)?))?)([,:]?)\s*(.*)$`,
      "is",
    ),
    needsMarker: false,
  },
  {
    pattern: cadencePattern(
      String.raw`^(/schedule\s+)?((?:cada|todos\s+los|todas\s+las)\s+` +
        String.raw`(d[ií]as?\s+(?:laborables?|h[aá]bil(?:es)?)|d[ií]as?|ma[nñ]anas?|lunes|martes|` +
        String.raw`mi[eé]rcoles|jueves|viernes|s[aá]bados?|domingos?|semanas?)\b` +
        String.raw`(?:\s+a\s+las?\s+([\d:]+(?:\s+y\s+(?:media|cuarto)\b)?` +
        String.raw`(?:\s*(?:a\.\s?m\.|p\.\s?m\.|(?:am|pm)\b)|\s+de\s+la\s+(?:ma[nñ]ana|tarde|noche)\b)?))?)` +
        String.raw`([,:]?)\s*(.*)$`,
      "is",
    ),
    needsMarker: false,
  },
  {
    pattern: cadencePattern(
      String.raw`^(/schedule\s+)?((每(?:天|日)(?:早上|上午|早晨)|每(?:个|個)?工作(?:日|天)|工作日每天|` +
        String.raw`每(?:个|個)?(?:周|週|星期|礼拜|禮拜)[一二三四五六日天]|每(?:个|個)?(?:周|週|星期|礼拜|禮拜)|每(?:天|日))` +
        String.raw`(?:\s*((?:早上|上午|早晨|凌晨|中午|下午|晚上|傍晚)?\s*` +
        String.raw`(?:\d{1,2}\s*[:：]\s*\d{2}|(?:\d{1,2}|十[一二]?|[一二三四五六七八九两兩])\s*[点點]` +
        String.raw`(?:\s*(?:半|\d{1,2}(?:\s*分)?|[钟鐘整]))?)))?)\s*([,:，：、]?)\s*(.*)$`,
      "is",
    ),
    needsMarker: true,
  },
];

type CadenceMatch = { when: string; request: string };

/**
 * The cadence at the start of a text and the request after it, read the way
 * the server reads it. `explicit` is a text already under /schedule.
 */
function matchCadence(text: string, explicit: boolean): CadenceMatch | null {
  for (const { pattern, needsMarker } of SCHEDULE_GRAMMARS) {
    const match = pattern.exec(text);
    if (!match) {
      continue;
    }
    if (needsMarker && !explicit && !match[1] && !match[4] && !match[5]) {
      return null;
    }

    return { when: match[2] ?? "", request: match[6] ?? "" };
  }

  return null;
}

/**
 * A message that asks for a schedule rather than an answer: "/schedule …", or
 * one that opens with a cadence the server reads, such as "every weekday at
 * 7:30am, …", "cada lunes a las 8, …" or "每天8点，…". The server reads the
 * same patterns (conversationschedule.ParseRequest); a message that only
 * starts with "every", like "every time I open the queue…", is an ordinary
 * question, and so is "每天有多少票货？".
 */
export function isScheduleRequest(text: string): boolean {
  const trimmed = text.trim();

  return /^\/schedule(?:\s|$)/i.test(trimmed) || matchCadence(trimmed, false) !== null;
}

/**
 * A time still being typed after a cadence: "every Monday at", "cada lunes a
 * las", "a las 8 de la", "每周一 8".
 */
const TIME_IN_PROGRESS = [
  /^at(?:\s+[\d:]*\s*[ap]?m?)?$/i,
  /^a(?:\s+l(?:as?)?)?(?:\s+[\d:]*)?$/i,
  /^(?:de(?:\s+la?)?|y)$/i,
  /^(?:早上|上午|早晨|凌晨|中午|下午|晚上|傍晚)?\s*[\d:：一二三四五六七八九十两兩]*$/,
];

/**
 * The /schedule command's slots: the cadence, however many words it takes,
 * then the request. Until a cadence is read the whole text is the "when".
 */
export function splitScheduleSlots(rest: string): string[] {
  const match = matchCadence(rest.trim(), true);
  if (!match) {
    return [rest.trim(), ""];
  }

  const request = match.request.trim();
  if (request !== "" && TIME_IN_PROGRESS.some((pattern) => pattern.test(request))) {
    return [rest.trim(), ""];
  }

  return [match.when.trim(), request];
}

export const SLASH_COMMANDS: readonly SlashCommand[] = [
  {
    name: "schedule",
    description: "Run a request on a schedule and post the results here",
    slots: [
      { name: "when", hint: "every Monday at 8am" },
      { name: "request", hint: "what to ask" },
    ],
    prompt: ([when = "", request = ""]) => `${when}, ${request}`,
    splitSlots: splitScheduleSlots,
  },
  {
    name: "status",
    description: "Where a shipment is and what is holding it up",
    slots: [{ name: "shipment", hint: "PRO or shipment number" }],
    prompt: ([shipment]) =>
      translate(
        "What is the status of shipment {0} right now, and is anything holding it up?",
        shipment,
      ),
  },
  {
    name: "quote",
    description: "What we would charge for a lane",
    slots: [
      { name: "origin", hint: "origin city" },
      { name: "destination", hint: "destination city" },
    ],
    prompt: ([origin, destination]) =>
      translate(
        "Quote a truckload shipment from {0} to {1}. Say which rate applied and what it is made of.",
        origin,
        destination,
      ),
  },
  {
    name: "report",
    description: "Run a saved report and show the results",
    slots: [{ name: "report", hint: "report name" }],
    prompt: ([report]) => translate("Run the report {0} and show me the results.", report),
  },
  {
    name: "explain",
    description: "Explain what is on this page",
    slots: [],
    prompt: () =>
      translate(
        "Explain what I am looking at on this page: what the filters mean, what the figures say, and what stands out.",
      ),
  },
];

/** What a composer that can compact reads as the /compact command. */
export const COMPACT_TEXT = "/compact";

/**
 * /compact summarizes the conversation's older turns. It asks the agent
 * nothing, so it is offered only where a composer can compact, which handles
 * the command itself rather than sending it.
 */
export const COMPACT_COMMAND: SlashCommand = {
  name: "compact",
  description: "Summarize older turns to free up context",
  slots: [],
  prompt: () => COMPACT_TEXT,
};

/**
 * A draft that opens with a slash is a request for the starter questions,
 * filtered by whatever follows. Null for any other draft, including a slash
 * somewhere later in a sentence and a draft that has grown to a second line.
 */
export function slashQuery(draft: string): string | null {
  if (!draft.startsWith("/") || draft.includes("\n")) {
    return null;
  }

  return draft.slice(1).trim().toLowerCase();
}

/**
 * The starter questions matching a slash query, on their label or their
 * prompt. An empty query is every one of them.
 */
export function matchSuggestions(query: string, suggestions: readonly Suggestion[]): Suggestion[] {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return [...suggestions];
  }

  return suggestions.filter(
    (suggestion) =>
      suggestion.label.toLowerCase().includes(needle) ||
      suggestion.prompt.toLowerCase().includes(needle),
  );
}

export type ParsedSlashCommand = {
  command: SlashCommand;
  /** One entry per slot; an empty string where the slot is still empty. */
  args: string[];
  complete: boolean;
};

/**
 * Reads a draft as a slash command with its slots filled in order. Words
 * fill the slots one each, and the last slot takes the rest of the line so
 * a place name with a space in it is not split. Null when the first word is
 * not a command.
 */
export function parseSlashCommand(
  draft: string,
  extra: readonly SlashCommand[] = [],
): ParsedSlashCommand | null {
  if (!draft.startsWith("/") || draft.includes("\n")) {
    return null;
  }
  const body = draft.slice(1);
  const firstSpace = body.search(/\s/);
  const name = (firstSpace === -1 ? body : body.slice(0, firstSpace)).toLowerCase();
  const command = [...SLASH_COMMANDS, ...extra].find((candidate) => candidate.name === name);
  if (!command) {
    return null;
  }

  const rest = firstSpace === -1 ? "" : body.slice(firstSpace + 1).trim();
  if (command.splitSlots) {
    const split = command.splitSlots(rest);
    return {
      command,
      args: split,
      complete: split.length === command.slots.length && split.every((arg) => arg !== ""),
    };
  }
  const args: string[] = [];
  let remaining = rest;
  for (let index = 0; index < command.slots.length; index += 1) {
    const last = index === command.slots.length - 1;
    if (last) {
      args.push(remaining.trim());
      break;
    }
    const cut = remaining.search(/\s/);
    if (cut === -1) {
      args.push(remaining.trim());
      remaining = "";
    } else {
      args.push(remaining.slice(0, cut));
      remaining = remaining.slice(cut + 1).trimStart();
    }
  }

  return {
    command,
    args,
    complete: args.length === command.slots.length && args.every((arg) => arg !== ""),
  };
}

/**
 * Writes the command's prompt in the language on screen, with each slot in
 * place.
 */
export function fillCommand(command: SlashCommand, args: readonly string[]): string {
  return command.prompt(command.slots.map((_, index) => (args[index] ?? "").trim()));
}

/** One row of the list a slash opens: a command to fill, or a question to send. */
export type CommandEntry =
  | { kind: "command"; label: string; description: string; command: SlashCommand }
  | { kind: "question"; label: string; description: string; prompt: string };

/**
 * What a slash offers: the commands first, then the agent's starter
 * questions, both narrowed by the query. Once a command's first slot has
 * begun, only that command is listed, with its slots as the hint.
 */
export function commandEntries(
  query: string,
  suggestions: readonly Suggestion[],
  extra: readonly SlashCommand[] = [],
): CommandEntry[] {
  const needle = query.trim().toLowerCase();
  const typed = parseSlashCommand("/" + query, extra);
  if (typed && /\s/.test(query)) {
    return [commandEntry(typed.command)];
  }

  const commands = [...SLASH_COMMANDS, ...extra]
    .filter((command) => needle === "" || command.name.startsWith(needle))
    .map(commandEntry);
  const questions = matchSuggestions(needle, suggestions).map<CommandEntry>((suggestion) => ({
    kind: "question",
    label: suggestion.label,
    description: "",
    prompt: suggestion.prompt,
  }));

  return [...commands, ...questions];
}

function commandEntry(command: SlashCommand): CommandEntry {
  return {
    kind: "command",
    label: "/" + command.name,
    description: command.description,
    command,
  };
}

/** The hint shown while a command's slots are being filled, in the language on screen. */
export function slotHint(parsed: ParsedSlashCommand): string {
  return parsed.command.slots
    .map((slot, index) => (parsed.args[index] ? "" : `{${translate(slot.hint)}}`))
    .filter((hint) => hint !== "")
    .join(" ");
}
