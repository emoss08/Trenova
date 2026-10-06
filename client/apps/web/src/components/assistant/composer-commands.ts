import type { Suggestion } from "./suggestions";

/**
 * A slot in a slash command: what goes there, in the words shown while it is
 * empty.
 */
export type CommandSlot = {
  name: string;
  hint: string;
};

/**
 * A slash command with typed slots. The template writes the question the
 * agent actually receives, so a person types `/status S12345` and the agent
 * is asked a full sentence about that shipment.
 */
export type SlashCommand = {
  name: string;
  description: string;
  slots: readonly CommandSlot[];
  template: string;
  /**
   * Splits what follows the command into its slots, for a command whose
   * slots are not one word each. Without it, words fill the slots in order.
   */
  splitSlots?: (rest: string) => string[];
};

const SCHEDULE_UNITS =
  "weekday|day|morning|monday|tuesday|wednesday|thursday|friday|saturday|sunday|week";

/**
 * A message that asks for a schedule rather than an answer: "/schedule …", or
 * one that opens with a cadence the server reads, such as "every weekday at
 * 7:30am, …". The server reads the same pattern
 * (conversationschedule.ParseRequest); a message that only starts with
 * "every", like "every time I open the queue…", is an ordinary question.
 */
const SCHEDULE_REQUEST = new RegExp(
  `^(?:/schedule\\s+)?(?:every|each)\\s+(?:${SCHEDULE_UNITS})\\w*(?:\\s+at\\s+[\\d:]+(?:\\s*(?:am|pm)\\b)?)?[,:]?\\s*(.*)$`,
  "is",
);

/** The cadence at the start of a schedule's text: its "when" slot. */
const SCHEDULE_WHEN = new RegExp(
  `^((?:every|each)\\s+(?:${SCHEDULE_UNITS})\\w*(?:\\s+at\\s+[\\d:]+(?:\\s*(?:am|pm)\\b)?)?)[,:]?\\s*(.*)$`,
  "is",
);

/** Whether a message, as sent, schedules a request rather than asking one. */
export function isScheduleRequest(text: string): boolean {
  const trimmed = text.trim();

  return /^\/schedule(?:\s|$)/i.test(trimmed) || SCHEDULE_REQUEST.test(trimmed);
}

/**
 * The /schedule command's slots: the cadence, however many words it takes,
 * then the request. Until a cadence is read the whole text is the "when".
 */
export function splitScheduleSlots(rest: string): string[] {
  const match = SCHEDULE_WHEN.exec(rest.trim());
  if (!match) {
    return [rest.trim(), ""];
  }

  const request = match[2]?.trim() ?? "";
  // "every Monday at" and "every Monday at 8 a" are a time still being typed.
  if (/^at(?:\s+[\d:]*\s*[ap]?m?)?$/i.test(request)) {
    return [rest.trim(), ""];
  }

  return [match[1]?.trim() ?? "", request];
}

export const SLASH_COMMANDS: readonly SlashCommand[] = [
  {
    name: "schedule",
    description: "Run a request on a schedule and post the results here",
    slots: [
      { name: "when", hint: "every Monday at 8am" },
      { name: "request", hint: "what to ask" },
    ],
    template: "{when}, {request}",
    splitSlots: splitScheduleSlots,
  },
  {
    name: "status",
    description: "Where a shipment is and what is holding it up",
    slots: [{ name: "shipment", hint: "PRO or shipment number" }],
    template: "What is the status of shipment {shipment} right now, and is anything holding it up?",
  },
  {
    name: "quote",
    description: "What we would charge for a lane",
    slots: [
      { name: "origin", hint: "origin city" },
      { name: "destination", hint: "destination city" },
    ],
    template:
      "Quote a truckload shipment from {origin} to {destination}. Say which rate applied and what it is made of.",
  },
  {
    name: "report",
    description: "Run a saved report and show the results",
    slots: [{ name: "report", hint: "report name" }],
    template: "Run the report {report} and show me the results.",
  },
  {
    name: "explain",
    description: "Explain what is on this page",
    slots: [],
    template:
      "Explain what I am looking at on this page: what the filters mean, what the figures say, and what stands out.",
  },
];

/**
 * /compact summarizes the conversation's older turns. It asks the agent
 * nothing, so it is offered only where a composer can compact, which handles
 * the command itself rather than sending it.
 */
export const COMPACT_COMMAND: SlashCommand = {
  name: "compact",
  description: "Summarize older turns to free up context",
  slots: [],
  template: "/compact",
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

/** Writes the command's prompt with each slot in place. */
export function fillCommand(command: SlashCommand, args: readonly string[]): string {
  return command.slots.reduce(
    (prompt, slot, index) => prompt.replaceAll(`{${slot.name}}`, (args[index] ?? "").trim()),
    command.template,
  );
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

/** The hint shown while a command's slots are being filled. */
export function slotHint(parsed: ParsedSlashCommand): string {
  return parsed.command.slots
    .map((slot, index) => (parsed.args[index] ? "" : `{${slot.hint}}`))
    .filter((hint) => hint !== "")
    .join(" ");
}
