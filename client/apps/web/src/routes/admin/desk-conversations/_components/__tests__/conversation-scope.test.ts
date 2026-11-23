import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import type { DeskTranscriptExportRow } from "@/lib/graphql/desk-transcripts";
import {
  DEFAULT_CONVERSATION_SCOPE,
  TIMEFRAME_PRESETS,
  parseConversationScope,
  scopeFilter,
  scopeKey,
  timeframeLabel,
} from "../conversation-scope";
import {
  describeCoverage,
  describeOwners,
  exportSummary,
  selectionLabel,
} from "../export-coverage";

const t = translate;

/*
The presets as services/tms/internal/api/graphql/schema/desk_transcript.graphqls
declares them, less Custom, which the picker offers as its calendar.
*/
const SCHEMA_PRESETS = ["Last7Days", "Last30Days", "Last90Days", "Last12Months", "AllTime"];

function coverage(
  fields: Partial<DeskTranscriptExportRow["coverage"]>,
): DeskTranscriptExportRow["coverage"] {
  return {
    kind: "Filter",
    timeframe: { preset: "Last30Days", fromDate: null, toDate: null },
    personName: null,
    agentName: null,
    origins: [],
    selectedCount: 0,
    ownerNames: [],
    ownerCount: 0,
    ...fields,
  };
}

describe("conversation scope", () => {
  it("offers every preset the server resolves, in one list the list and the export share", () => {
    expect([...TIMEFRAME_PRESETS]).toEqual(SCHEMA_PRESETS);
  });

  it("sends a preset as the server's filter and lets the server resolve it", () => {
    expect(scopeFilter(DEFAULT_CONVERSATION_SCOPE)).toEqual({
      timeframe: { preset: "Last30Days" },
      userId: null,
      agentId: null,
      origins: null,
    });
  });

  it("sends a custom range as calendar days with the person, agent and origin", () => {
    expect(
      scopeFilter({
        timeframe: { preset: "Custom", fromDate: "2026-03-01", toDate: "2026-03-31" },
        personId: "usr_1",
        agent: { id: "agd_1", name: "Billing exceptions" },
        origin: "Watchtower",
      }),
    ).toEqual({
      timeframe: { preset: "Custom", fromDate: "2026-03-01", toDate: "2026-03-31" },
      userId: "usr_1",
      agentId: "agd_1",
      origins: ["Watchtower"],
    });
  });

  it("reads back what it wrote to the address and refuses a backwards range", () => {
    const scope = {
      timeframe: { preset: "Last7Days" as const },
      personId: "usr_1",
      agent: null,
      origin: null,
    };
    expect(parseConversationScope(scope)).toEqual(scope);
    expect(
      parseConversationScope({
        ...scope,
        timeframe: { preset: "Custom", fromDate: "2026-03-31", toDate: "2026-03-01" },
      }),
    ).toBeNull();
  });

  it("gives every narrowing its own key, so the table starts over when the filters change", () => {
    expect(scopeKey(DEFAULT_CONVERSATION_SCOPE)).not.toEqual(
      scopeKey({ ...DEFAULT_CONVERSATION_SCOPE, personId: "usr_1" }),
    );
  });

  it("names each preset in words", () => {
    expect(TIMEFRAME_PRESETS.map((preset) => timeframeLabel(t, { preset }))).toEqual([
      "Last 7 days",
      "Last 30 days",
      "Last 90 days",
      "Last 12 months",
      "All time",
    ]);
  });
});

describe("export words", () => {
  it("says what a timeframe export covered", () => {
    expect(describeCoverage(t, coverage({}))).toBe("Last 30 days, everyone");
    expect(
      describeCoverage(t, coverage({ personName: "Dana Reyes", agentName: "Billing exceptions" })),
    ).toBe("Last 30 days, Dana Reyes, with Billing exceptions");
  });

  it("says how many conversations a selection held", () => {
    expect(
      describeCoverage(t, coverage({ kind: "Selection", timeframe: null, selectedCount: 12 })),
    ).toBe("12 selected conversations");
    expect(
      describeCoverage(t, coverage({ kind: "Selection", timeframe: null, selectedCount: 1 })),
    ).toBe("1 selected conversation");
  });

  it("names the owners, then how many more", () => {
    expect(describeOwners(t, coverage({ ownerNames: ["Dana Reyes"], ownerCount: 1 }))).toBe(
      "Dana Reyes",
    );
    expect(describeOwners(t, coverage({ ownerNames: ["Dana Reyes"], ownerCount: 3 }))).toBe(
      "Dana Reyes and 2 others",
    );
  });

  it("names the selection action for one and for many, and refuses past the bound", () => {
    expect(selectionLabel(t, 1)).toBe("Download 1 transcript");
    expect(selectionLabel(t, 12)).toBe("Download 12 transcripts");
    expect(selectionLabel(t, 201)).toBe("Select at most 200 to download");
  });

  it("says before an export starts what it will hold, and when it is too much", () => {
    const scope = DEFAULT_CONVERSATION_SCOPE;
    expect(exportSummary(t, { count: 41, exact: true, limit: 5000 }, scope)).toBe(
      "41 conversations will be exported as one zip: a Markdown transcript for each, and conversations.jsonl with all of them.",
    );
    expect(exportSummary(t, { count: 5001, exact: false, limit: 5000 }, scope)).toContain(
      "More than 5,000 conversations match",
    );
    expect(exportSummary(t, { count: 0, exact: true, limit: 5000 }, scope)).toBe(
      "No conversation matches. Choose a longer timeframe, or another person or agent.",
    );
  });
});
