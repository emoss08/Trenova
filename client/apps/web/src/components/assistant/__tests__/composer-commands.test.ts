import { registerCatalogSource, setLocale } from "@trenova/shared/i18n/runtime";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
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
  COMPACT_COMMAND,
  fillCommand,
  isScheduleRequest,
  parseSlashCommand,
  SLASH_COMMANDS,
  slotHint,
  splitScheduleSlots,
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

/*
A message that opens with a cadence is a schedule, not a question, and the
composer reads it with the same pattern the server does. "Every time…" is a
question.
*/
describe("isScheduleRequest", () => {
  it("reads a cadence at the start as a schedule", () => {
    expect(isScheduleRequest("every weekday at 7:30am, what's blocking the billing queue?")).toBe(
      true,
    );
    expect(isScheduleRequest("Each Monday summarize detention")).toBe(true);
    expect(isScheduleRequest("every week, margin by lane")).toBe(true);
    expect(isScheduleRequest("/schedule every day check loads")).toBe(true);
    expect(isScheduleRequest("/schedule hourly check loads")).toBe(true);
  });

  it("leaves ordinary questions alone", () => {
    expect(isScheduleRequest("Every time I open the queue it is slow, why?")).toBe(false);
    expect(isScheduleRequest("every hour check the board")).toBe(false);
    expect(isScheduleRequest("What happens every Monday?")).toBe(false);
    expect(isScheduleRequest("/schedules")).toBe(false);
  });
});

describe("/schedule", () => {
  it("fills its when slot with the whole cadence and its request with the rest", () => {
    const parsed = parseSlashCommand("/schedule every Monday at 8am summarize detention");
    expect(parsed?.command.name).toBe("schedule");
    expect(parsed?.args).toEqual(["every Monday at 8am", "summarize detention"]);
    expect(parsed?.complete).toBe(true);
    expect(fillCommand(parsed!.command, parsed!.args)).toBe(
      "every Monday at 8am, summarize detention",
    );
    expect(isScheduleRequest(fillCommand(parsed!.command, parsed!.args))).toBe(true);
  });

  it("waits while the cadence or its time is still being typed", () => {
    expect(parseSlashCommand("/schedule every Mon")?.complete).toBe(false);
    expect(parseSlashCommand("/schedule every Monday at")?.args).toEqual(["every Monday at", ""]);
    expect(parseSlashCommand("/schedule every Monday at 8am")?.complete).toBe(false);
  });

  it("is listed first among the commands", () => {
    expect(commandEntries("sch", suggestions).map((entry) => entry.label)).toEqual(["/schedule"]);
    expect(splitScheduleSlots("each morning, list unassigned loads")).toEqual([
      "each morning",
      "list unassigned loads",
    ]);
  });
});

/*
The composer reads the same cadences the server does, in English, Spanish and
Chinese whatever the language on screen. These mirror the cases in
conversationschedule/cadence_test.go.
*/
describe("isScheduleRequest in Spanish and Chinese", () => {
  it("reads a Spanish cadence at the start as a schedule", () => {
    for (const text of [
      "cada día laborable a las 7:30, ¿qué está bloqueando la cola de facturación?",
      "todos los dias habiles a las 8 am: cargas sin asignar",
      "/schedule todos los lunes a las 8 a. m., resume la detención por cliente",
      "cada día revisa las cargas atrasadas",
      "todas las mañanas a las 6:15, por favor lista las cargas sin asignar",
      "cada semana a las 9, ¿cómo nos fue con el margen?",
      "todos los viernes a las 4 y media de la tarde cierra la semana",
      "todos los sabados a la 1 pm, ¿qué vence el lunes?",
      "cada miércoles a las 12 de la noche: cierre",
      "Todos Los Domingos a las 17:45 quién sigue en ruta",
      "cada martes a las 7 y cuarto de la mañana, informe de antigüedad",
    ]) {
      expect(isScheduleRequest(text), text).toBe(true);
    }
  });

  it("reads a Chinese cadence with a time or a separator as a schedule", () => {
    for (const text of [
      "每天8点，汇总延误的运单",
      "每天早上7点半汇总未分配的货物",
      "每个工作日上午7:30，计费队列卡在哪里？",
      "每個工作日 17：45：誰還在路上？",
      "/schedule 每周一上午 8 点, 请汇总上周各客户的滞留",
      "每週五下午4點30分 總結本週",
      "每周：利润怎么样？",
      "每星期天晚上九点、下周到期的有哪些？",
      "每个星期六中午12点，结算本周",
      "/schedule 每日汇总运单",
      "工作日每天20点 谁还在路上",
    ]) {
      expect(isScheduleRequest(text), text).toBe(true);
    }
  });

  it("leaves questions that only open with cada or 每天 alone", () => {
    for (const text of [
      "cada vez que abro la cola va lento, ¿por qué?",
      "cada hora revisa el tablero",
      "¿Qué pasa cada lunes?",
      "cada diario cuenta",
      "todos los clientes, ¿quién debe más?",
      "每天有多少票货？",
      "每天早上有多少票货？",
      "每周一次汇总",
      "每日报告在哪里？",
      "每个司机每天跑多少？",
      "今天每天8点",
    ]) {
      expect(isScheduleRequest(text), text).toBe(false);
    }
  });
});

describe("/schedule in Spanish and Chinese", () => {
  it("fills its when slot with the whole cadence and its request with the rest", () => {
    expect(
      parseSlashCommand("/schedule todos los lunes a las 8 a. m. resume la detención")?.args,
    ).toEqual(["todos los lunes a las 8 a. m.", "resume la detención"]);
    expect(parseSlashCommand("/schedule 每周一上午 8 点 汇总滞留")?.args).toEqual([
      "每周一上午 8 点",
      "汇总滞留",
    ]);
    expect(parseSlashCommand("/schedule 每天汇总运单")?.args).toEqual(["每天", "汇总运单"]);

    const parsed = parseSlashCommand("/schedule 每週五下午4點半 總結本週")!;
    expect(parsed.complete).toBe(true);
    expect(isScheduleRequest(fillCommand(parsed.command, parsed.args))).toBe(true);
  });

  it("waits while the time is still being typed", () => {
    expect(parseSlashCommand("/schedule cada lunes a las")?.complete).toBe(false);
    expect(parseSlashCommand("/schedule cada lunes a las 8 de la")?.complete).toBe(false);
    expect(parseSlashCommand("/schedule 每周一下午")?.complete).toBe(false);
    expect(parseSlashCommand("/schedule 每周一 8")?.complete).toBe(false);
    expect(parseSlashCommand("/schedule 每周一 8点")?.complete).toBe(false);
  });
});

/**
 * A command's prompt is a whole message in the catalog, so a person reading
 * Spanish asks the agent in Spanish.
 */
describe("slash commands in another language", () => {
  beforeAll(async () => {
    await registerCatalogSource({
      es: async () => ({
        "What is the status of shipment {0} right now, and is anything holding it up?":
          "¿Cuál es el estado del envío {0} ahora mismo y hay algo que lo retenga?",
        "Quote a truckload shipment from {0} to {1}. Say which rate applied and what it is made of.":
          "Cotiza un envío de carga completa de {0} a {1}. Indica qué tarifa se aplicó y de qué se compone.",
        "PRO or shipment number": "Número PRO o de envío",
      }),
    });
    await setLocale("es");
  });

  afterAll(async () => {
    await setLocale("en");
  });

  it("writes the prompt in Spanish with each slot in place", () => {
    const status = SLASH_COMMANDS.find((command) => command.name === "status")!;
    expect(fillCommand(status, [" S12345 "])).toBe(
      "¿Cuál es el estado del envío S12345 ahora mismo y hay algo que lo retenga?",
    );
    const quote = SLASH_COMMANDS.find((command) => command.name === "quote")!;
    expect(fillCommand(quote, ["Dallas", "Chicago, IL"])).toBe(
      "Cotiza un envío de carga completa de Dallas a Chicago, IL. Indica qué tarifa se aplicó y de qué se compone.",
    );
  });

  it("shows the empty slots' hints in Spanish", () => {
    expect(slotHint(parseSlashCommand("/status ")!)).toBe("{Número PRO o de envío}");
  });

  it("keeps /compact as the command it is", () => {
    expect(fillCommand(COMPACT_COMMAND, [])).toBe("/compact");
  });
});
