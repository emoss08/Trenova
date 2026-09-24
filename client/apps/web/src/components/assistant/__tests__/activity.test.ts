import { translate } from "@trenova/shared/i18n/runtime";
import type { AssistantMessage, ToolEffect } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  currentActivity,
  describeActivity,
  groupActivity,
  reportSummary,
  stepsFromExchanges,
  summaryCount,
  toolEffect,
  type ToolStep,
} from "../activity";
import type { ToolExchange } from "../thread-view";

const t = translate;

function step(overrides: Partial<ToolStep> & { name: string }): ToolStep {
  return {
    id: overrides.id ?? `call_${overrides.name}_${Math.random().toString(36).slice(2, 8)}`,
    arguments: {},
    status: "done",
    content: "",
    effect: toolEffect(overrides.name, overrides.effect),
    summary: "",
    durationSeconds: null,
    ...overrides,
  };
}

function line(steps: ToolStep[]) {
  const groups = groupActivity(steps);
  expect(groups).toHaveLength(1);
  return describeActivity(groups[0], t);
}

/**
 * The server sends `effect` on every call it knows. A thread saved before
 * that, or a tool it retired, arrives without one, and the name decides —
 * with anything unrecognised read as a change, because a write shown as a
 * lookup is the defect this exists to stop.
 */
describe("toolEffect", () => {
  it("uses the server's effect when it sent one", () => {
    expect(toolEffect("list_customers", "change")).toBe("change");
  });

  it.each<[string, ToolEffect]>([
    ["open_page", "navigate"],
    ["find_tools", "discover"],
    ["find_in_trenova", "discover"],
    ["run_report", "present"],
    ["publish_artifact", "present"],
    ["compare_report_runs", "present"],
    ["compose_table_view", "present"],
    ["ask_user", "ask"],
    ["list_customers", "lookup"],
    ["get_shipment", "lookup"],
    ["search_worker", "lookup"],
    ["describe_report_dataset", "lookup"],
    ["preview_report", "lookup"],
    ["create_report", "change"],
    ["cancel_shipment", "change"],
    ["something_new", "change"],
  ])("derives %s as %s when the effect is missing", (name, effect) => {
    expect(toolEffect(name)).toBe(effect);
    expect(toolEffect(name, null)).toBe(effect);
  });
});

/**
 * A list's summary is the server's English ("3 customers"). The count is
 * read out of it and said in the reader's language; the noun never is.
 */
describe("summaryCount", () => {
  it("reads the count forms the server writes", () => {
    expect(summaryCount("3 customers")).toEqual({ count: 3, more: false });
    expect(summaryCount("25+ shipments")).toEqual({ count: 25, more: true });
    expect(summaryCount("No customers")).toEqual({ count: 0, more: false });
    expect(summaryCount("1 result")).toEqual({ count: 1, more: false });
  });

  it("does not mistake a name for a count", () => {
    expect(summaryCount("Peak Distributing")).toBeNull();
    expect(summaryCount("PRO-1042")).toBeNull();
    expect(summaryCount("3M Company")).toBeNull();
    expect(summaryCount("")).toBeNull();
  });
});

describe("reportSummary", () => {
  it("splits a finished run into its name and rows", () => {
    expect(reportSummary("Late loads · 42 rows")).toEqual({ name: "Late loads", rows: 42 });
    expect(reportSummary("Late loads · 1 row")).toEqual({ name: "Late loads", rows: 1 });
  });

  it("keeps a name with no rows as the name", () => {
    expect(reportSummary("Late loads")).toEqual({ name: "Late loads", rows: null });
  });
});

describe("groupActivity", () => {
  // The report this came from: an open_page folded into "Looked up 1 record".
  it("never folds an action into the reads around it", () => {
    const groups = groupActivity([
      step({ name: "find_in_trenova" }),
      step({ name: "open_page" }),
      step({ name: "get_customer" }),
      step({ name: "list_shipments" }),
      step({ name: "create_report" }),
      step({ name: "run_report" }),
    ]);

    expect(groups.map((group) => [group.effect, group.steps.length])).toEqual([
      ["discover", 1],
      ["navigate", 1],
      ["lookup", 2],
      ["change", 1],
      ["present", 1],
    ]);
  });

  it("keeps consecutive actions on lines of their own", () => {
    const groups = groupActivity([step({ name: "open_page" }), step({ name: "open_page" })]);

    expect(groups).toHaveLength(2);
  });
});

describe("describeActivity", () => {
  it("names the record a single lookup found", () => {
    expect(line([step({ name: "get_customer", summary: "Peak Distributing" })]).phrase).toBe(
      "Looked up Peak Distributing",
    );
  });

  it("counts several single-record lookups as records and names the first", () => {
    const result = line([
      step({ name: "get_customer", summary: "Peak Distributing" }),
      step({ name: "get_shipment", summary: "PRO-1042" }),
      step({ name: "get_worker", summary: "Maria Ortiz" }),
    ]);

    expect(result.phrase).toBe("Looked up 3 records");
    expect(result.detail).toBe("Peak Distributing, PRO-1042 +1");
  });

  it("says a list's count in its own words rather than the server's noun", () => {
    expect(line([step({ name: "list_customers", summary: "3 customers" })]).phrase).toBe(
      "Found 3 records",
    );
    expect(line([step({ name: "list_shipments", summary: "25+ shipments" })]).phrase).toBe(
      "Found 25+ records",
    );
    expect(line([step({ name: "list_customers", summary: "No customers" })]).phrase).toBe(
      "Found nothing",
    );
  });

  it("does not call lists records when reads of both kinds fold together", () => {
    expect(
      line([
        step({ name: "list_customers", summary: "3 customers" }),
        step({ name: "get_customer", summary: "Peak Distributing" }),
      ]).phrase,
    ).toBe("Ran 2 lookups");
  });

  it("falls back to the argument it was asked about for a saved lookup with no summary", () => {
    expect(line([step({ name: "get_shipment", arguments: { proNumber: "S1" } })]).phrase).toBe(
      "Looked up S1",
    );
  });

  it("reads an opened page as opened", () => {
    expect(line([step({ name: "open_page", summary: "Report library" })]).phrase).toBe(
      "Opened Report library",
    );
  });

  it("takes an opened page's name from its result when the summary is missing", () => {
    const content =
      'Result from open_page\n<untrusted_data>{"path":"/reports","name":"Report library"}</untrusted_data>';
    expect(line([step({ name: "open_page", content })]).phrase).toBe("Opened Report library");
  });

  it("says the right tools were found", () => {
    expect(line([step({ name: "find_tools", summary: "4 results" })]).phrase).toBe(
      "Found the right tools",
    );
  });

  it("reads a finished run by its report and rows", () => {
    const result = line([step({ name: "run_report", summary: "Late loads · 42 rows" })]);

    expect(result.phrase).toBe("Ran Late loads");
    expect(result.detail).toBe("42 rows");
  });

  it("reads a run that has not finished as started", () => {
    expect(line([step({ name: "run_report", summary: "Late loads" })]).phrase).toBe(
      "Started Late loads",
    );
  });

  it("reads a write that ran by its verb and what it wrote", () => {
    expect(
      line([step({ name: "create_report", summary: "Shipments for Peak Distributing" })]).phrase,
    ).toBe("Saved Shipments for Peak Distributing");
  });

  it("reads a write waiting on approval as proposed, never as done", () => {
    const result = line([
      step({ name: "cancel_shipment", status: "proposed", summary: "PRO-1042" }),
    ]);

    expect(result.phrase).toBe("Proposed a change");
    expect(result.state).toBe("proposed");
    expect(result.detail).toBe("PRO-1042");
  });

  it("reads a question as asked", () => {
    expect(line([step({ name: "ask_user" })]).phrase).toBe("Asked you to choose");
  });

  it("says a failed call failed, with the reason", () => {
    const result = line([
      step({
        name: "open_page",
        status: "failed",
        content: 'Tool "open_page" failed: you may not open that page',
      }),
    ]);

    expect(result.state).toBe("failed");
    expect(result.phrase).toBe("Couldn't open that page");
    expect(result.detail).toBe("you may not open that page");
  });

  it("counts the failures among reads that otherwise worked", () => {
    const result = line([
      step({ name: "get_customer", summary: "Peak Distributing" }),
      step({ name: "get_customer", status: "failed", content: 'Tool "get_customer" failed: x' }),
    ]);

    expect(result.phrase).toBe("Looked up Peak Distributing");
    expect(result.failure).toBe("1 failed");
  });
});

describe("currentActivity", () => {
  it("describes the step under way, not the last one finished", () => {
    const current = currentActivity(
      [
        step({ name: "open_page", summary: "Report library" }),
        step({ name: "run_report", status: "running", arguments: { reportKey: "late-loads" } }),
      ],
      t,
    );

    expect(current?.phrase).toBe("Running a report…");
    expect(current?.state).toBe("running");
  });

  it("is null before any step", () => {
    expect(currentActivity([], t)).toBeNull();
  });
});

function saved(overrides: Partial<AssistantMessage>): AssistantMessage {
  return {
    id: "m",
    threadId: "t",
    sequence: 1,
    kind: "Message",
    role: "Tool",
    content: "",
    toolCalls: null,
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt: 0,
    ...overrides,
  };
}

/**
 * History carries the effect on the call and on the result, the summary on
 * the result, and a createdAt on each that is when it was produced.
 */
describe("stepsFromExchanges", () => {
  it("reads effect, summary and time from the saved call and its result", () => {
    const exchanges: ToolExchange[] = [
      {
        call: { id: "c1", name: "open_page", arguments: { page: "/reports" }, effect: "navigate" },
        result: saved({
          toolCallId: "c1",
          toolName: "open_page",
          content: "{}",
          summary: "Report library",
          effect: "navigate",
          createdAt: 1_700_000_004,
        }),
      },
    ];

    expect(stepsFromExchanges(exchanges, 1_700_000_001)).toEqual([
      {
        id: "c1",
        name: "open_page",
        arguments: { page: "/reports" },
        status: "done",
        content: "{}",
        effect: "navigate",
        summary: "Report library",
        durationSeconds: 3,
      },
    ]);
  });

  it("takes the effect from the result when only the result has one", () => {
    const [only] = stepsFromExchanges(
      [
        {
          call: { id: "c1", name: "custom_tool", arguments: {} },
          result: saved({ toolCallId: "c1", toolName: "custom_tool", effect: "lookup" }),
        },
      ],
      0,
    );

    expect(only.effect).toBe("lookup");
    expect(only.durationSeconds).toBeNull();
  });

  it("marks a recorded proposal and a failure as such", () => {
    const steps = stepsFromExchanges(
      [
        {
          call: { id: "c1", name: "cancel_shipment", arguments: {} },
          result: saved({
            toolCallId: "c1",
            content: 'Recorded a proposal to run "cancel_shipment".',
          }),
        },
        {
          call: { id: "c2", name: "get_shipment", arguments: {} },
          result: saved({ toolCallId: "c2", toolFailed: true }),
        },
      ],
      1,
    );

    expect(steps.map((entry) => entry.status)).toEqual(["proposed", "failed"]);
  });

  // A result whose call is not in view was rebuilt from the result alone;
  // the message it is shown under did not ask for it, so there is no time.
  it("does not time a result whose call is not in view", () => {
    const [only] = stepsFromExchanges(
      [
        {
          call: { id: "c9", name: "get_shipment", arguments: {} },
          result: saved({ toolCallId: "c9", toolName: "get_shipment", createdAt: 1_700_000_009 }),
          orphan: true,
        },
      ],
      1_700_000_001,
    );

    expect(only.durationSeconds).toBeNull();
  });
});

/**
 * Going to the web is said as such — what was searched for and which sites
 * were read — so a reader can tell an answer came from outside Trenova
 * before reading it.
 */
describe("web activity", () => {
  const read = (url: string, status: ToolStep["status"] = "done") =>
    step({ name: "web_read", effect: "lookup", arguments: { url, ref: "r" }, status });
  const search = (query: string, status: ToolStep["status"] = "done") =>
    step({ name: "web_search", effect: "lookup", arguments: { query }, status });

  it("says a search is under way and what it is for", () => {
    expect(line([search("ELD mandate", "running")])).toMatchObject({
      phrase: "Searching the web…",
      detail: "ELD mandate",
      state: "running",
    });
  });

  it("names the site being read", () => {
    expect(
      line([search("ELD mandate"), read("https://www.fmcsa.dot.gov/x", "running")]),
    ).toMatchObject({
      phrase: "Reading fmcsa.dot.gov…",
      state: "running",
    });
  });

  it("says what was searched and read once it is done", () => {
    expect(line([search("ELD mandate")])).toMatchObject({
      phrase: "Searched the web",
      detail: "ELD mandate",
      state: "done",
    });
    expect(line([search("ELD mandate"), search("ELD exemptions")])).toMatchObject({
      phrase: "Searched the web 2 times",
      detail: "ELD mandate, ELD exemptions",
    });
    expect(line([search("ELD mandate"), read("https://ecfr.gov/a")])).toMatchObject({
      phrase: "Searched the web and read 1 page",
      detail: "ELD mandate",
    });
    expect(line([read("https://ecfr.gov/a")])).toMatchObject({ phrase: "Read ecfr.gov" });
  });

  it("says the web could not be reached, with why", () => {
    expect(
      line([
        step({
          name: "web_search",
          effect: "lookup",
          status: "failed",
          content: 'Tool "web_search" failed: the monthly search budget is spent',
        }),
      ]),
    ).toMatchObject({
      phrase: "Couldn't reach the web",
      detail: "the monthly search budget is spent",
      state: "failed",
    });
  });

  it("keeps a web search apart from the lookups in Trenova beside it", () => {
    const groups = groupActivity([
      step({ name: "list_customers", effect: "lookup" }),
      search("ELD mandate"),
    ]);

    expect(groups).toHaveLength(2);
  });
});
