import { describe, expect, it } from "vitest";
import { matchSuggestions, slashQuery } from "../composer-commands";

const suggestions = [
  { label: "Where is a shipment right now?", prompt: "Where is PRO S12345 right now?" },
  { label: "What is picking up today?", prompt: "Which shipments pick up today?" },
  { label: "Is a driver available?", prompt: "Is Maria Ortiz free tomorrow?" },
];

describe("slashQuery", () => {
  it("reads a leading slash as a request for the starter questions", () => {
    expect(slashQuery("/")).toBe("");
    expect(slashQuery("/ship")).toBe("ship");
    expect(slashQuery("/  Driver ")).toBe("driver");
  });

  it("ignores a slash that is part of a sentence or a multi-line draft", () => {
    expect(slashQuery("what about 12/24?")).toBeNull();
    expect(slashQuery("/ship\nmore")).toBeNull();
    expect(slashQuery("")).toBeNull();
  });
});

describe("matchSuggestions", () => {
  it("offers everything for an empty query", () => {
    expect(matchSuggestions("", suggestions)).toHaveLength(3);
  });

  it("matches on the label or the prompt, ignoring case", () => {
    expect(matchSuggestions("SHIP", suggestions).map((s) => s.label)).toEqual([
      "Where is a shipment right now?",
      "What is picking up today?",
    ]);
    expect(matchSuggestions("ortiz", suggestions).map((s) => s.label)).toEqual([
      "Is a driver available?",
    ]);
    expect(matchSuggestions("zzz", suggestions)).toEqual([]);
  });
});

import {
  commandEntries,
  fillCommand,
  parseSlashCommand,
  SLASH_COMMANDS,
} from "../composer-commands";

/**
 * A slash command with slots is typed in one line: `/status S12345` becomes
 * the question about that shipment, and `/quote Dallas Chicago` fills two
 * slots in order. The last slot takes the rest of the line, so a place name
 * with a space is not split. A command missing a slot is not sendable.
 */
describe("parseSlashCommand", () => {
  it("reads the command and fills its slots from what follows", () => {
    const parsed = parseSlashCommand("/status S12345");
    expect(parsed?.command.name).toBe("status");
    expect(parsed?.args).toEqual(["S12345"]);
    expect(parsed?.complete).toBe(true);
  });

  it("gives the last slot the rest of the line", () => {
    const parsed = parseSlashCommand("/quote Dallas, TX Chicago, IL");
    expect(parsed?.args).toEqual(["Dallas,", "TX Chicago, IL"]);

    const twoWords = parseSlashCommand("/quote Dallas Chicago");
    expect(twoWords?.args).toEqual(["Dallas", "Chicago"]);
    expect(twoWords?.complete).toBe(true);
  });

  it("is incomplete while a slot is still empty, and null for a word that is not a command", () => {
    expect(parseSlashCommand("/status")?.complete).toBe(false);
    expect(parseSlashCommand("/status ")?.complete).toBe(false);
    expect(parseSlashCommand("/explain")?.complete).toBe(true);
    expect(parseSlashCommand("/pick")).toBeNull();
    expect(parseSlashCommand("status S1")).toBeNull();
  });
});

describe("fillCommand", () => {
  it("writes the prompt with each slot in place", () => {
    const status = SLASH_COMMANDS.find((command) => command.name === "status")!;
    expect(fillCommand(status, ["S12345"])).toBe(
      "What is the status of shipment S12345 right now, and is anything holding it up?",
    );
    const quote = SLASH_COMMANDS.find((command) => command.name === "quote")!;
    expect(fillCommand(quote, ["Dallas", "Chicago, IL"])).toContain("from Dallas to Chicago, IL");
  });
});

describe("commandEntries", () => {
  it("lists the commands before the starter questions for an empty slash", () => {
    const entries = commandEntries("", suggestions);
    expect(entries[0].kind).toBe("command");
    expect(entries.filter((entry) => entry.kind === "question")).toHaveLength(3);
  });

  it("matches a command on its name and a question on its words", () => {
    expect(commandEntries("quo", suggestions).map((entry) => entry.label)).toEqual(["/quote"]);
    expect(commandEntries("ortiz", suggestions).map((entry) => entry.kind)).toEqual(["question"]);
  });

  it("offers only the command being typed once its first slot begins", () => {
    const entries = commandEntries("status S1", suggestions);
    expect(entries).toHaveLength(1);
    expect(entries[0].label).toBe("/status");
  });
});
