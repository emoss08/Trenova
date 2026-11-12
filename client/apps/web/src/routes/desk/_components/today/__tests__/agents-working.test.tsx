import type { DeskAttentionGroup, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import type { WatchtowerAgentWork } from "@/lib/graphql/watchtower";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import { watchtowerItem } from "../../watchtower/__tests__/item-fixture";
import type { WorkingRun } from "@/lib/graphql/ai-control";
import { AgentsWorking, runOrigin, runSubject, workingItems, workingRows } from "../agents-working";

function agent(id: string, name: string): NonNullable<WorkingRun["agent"]> {
  return { id, name, icon: "", accent: "", template: null };
}

/** A run of an organization-wide agent that has said nothing yet, as one in flight is. */
function run(overrides: Partial<WorkingRun> = {}): WorkingRun {
  return {
    id: "ar_1",
    agentDefinitionId: "agdef_1",
    status: "Diagnosing",
    summary: "",
    trigger: "Manual",
    subjectType: "Organization",
    subjectLabel: "",
    startedAt: 1_800_000_000,
    createdAt: 1_800_000_000,
    agent: null,
    ...overrides,
  };
}

function work(overrides: Partial<WatchtowerAgentWork> = {}): WatchtowerAgentWork {
  return {
    agentId: "agdef_1",
    agent: null,
    runId: "ar_1",
    status: "Working",
    step: 0,
    steps: ["Read", "Decide", "Approve"],
    note: "",
    failed: false,
    handedAt: 1_800_000_000,
    handedById: null,
    handedBy: null,
    ...overrides,
  } as WatchtowerAgentWork;
}

function item(id: string, agentWork: WatchtowerAgentWork | null): DeskAttentionItem {
  return { ...watchtowerItem({ id }), deskLens: "Watchtower", aboutId: "shp_1", agentWork };
}

function group(items: DeskAttentionItem[]): DeskAttentionGroup {
  return {
    key: "shp_1",
    record: null,
    severity: "Warning",
    dueAt: null,
    dueLabel: null,
    unseen: false,
    items,
    ownership: null,
  };
}

/**
 * "Agents are working on" lists a run only while it runs. A run that finished,
 * failed or is waiting on a person has stopped working, whatever it wrote.
 */
describe("what agents are working on", () => {
  it("lists a run that is reading or deciding, and no other", () => {
    const groups = [
      group([
        item("wt_reading", work({ step: 0 })),
        item("wt_deciding", work({ step: 1 })),
        item("wt_waiting", work({ status: "NeedsYou", step: 2, note: "Proposed a hold" })),
        item("wt_done", work({ status: "Done", step: 3, note: "Found nothing to change" })),
        item("wt_failed", work({ status: "NeedsYou", step: 1, failed: true })),
        item("wt_nobody", null),
      ]),
    ];

    expect(workingItems(groups).map((row) => row.id)).toEqual(["wt_reading", "wt_deciding"]);
  });

  it("draws a run on an item as one line: what it is on, who has it and what it is doing", () => {
    const row = {
      ...item(
        "wt_1",
        work({ step: 1, agent: { id: "agdef_1", name: "Billing exceptions" } as never }),
      ),
      title: "POD missing on S-240412",
    };

    render(
      <MemoryRouter>
        <AgentsWorking rows={workingRows(null, [row], () => "/desk/r/shipment/shp_1")} />
      </MemoryRouter>,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/desk/r/shipment/shp_1");
    expect(link).toHaveTextContent("POD missing on S-240412");
    expect(link).toHaveTextContent("Billing exceptions · Working out what to do");
  });

  it("lists every working run, as AI control does, and shows the one on an item as that item", () => {
    const onItem = {
      ...item("wt_1", work({ runId: "ar_1", step: 0 })),
      title: "POD missing on S-1",
    };
    const rows = workingRows(
      [
        run({ id: "ar_1", status: "GatheringContext" }),
        run({ id: "ar_2", agent: agent("agdef_2", "Carrier risk desk") }),
      ],
      [onItem],
      () => "/desk/r/shipment/shp_1",
    );

    expect(rows.map((row) => [row.key, row.to])).toEqual([
      ["wt_1", "/desk/r/shipment/shp_1"],
      ["ar_2", "/admin/agent-control?tab=activity&view=runs&run=ar_2"],
    ]);
    expect(rows[0].item?.title).toBe("POD missing on S-1");
    expect(rows[1].agent?.name).toBe("Carrier risk desk");
  });

  it("names the run's own agent, though it is not one the reader may ask", () => {
    const rows = workingRows(
      [run({ id: "ar_9", trigger: "Scheduled", agent: agent("agdef_9", "Load monitor") })],
      [],
      () => "/desk/needs",
    );

    render(
      <MemoryRouter>
        <AgentsWorking rows={rows} />
      </MemoryRouter>,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveTextContent("Load monitor");
    expect(link).toHaveTextContent("On its schedule · Working out what to do");
    expect(link).not.toHaveTextContent("An agent");
  });

  it("says what a run is on in its own words, else the record by name, else nothing", () => {
    expect(runSubject(run({ summary: "  Checking carrier insurance  " }))).toBe(
      "Checking carrier insurance",
    );
    expect(runSubject(run({ subjectType: "Shipment", subjectLabel: "S-240412" }))).toBe("S-240412");
    expect(runSubject(run())).toBe("");
  });

  it("says how a run came to be going", () => {
    const t = ((text: string) => text) as never;

    expect(runOrigin(t, run({ subjectType: "AssistantThread", trigger: "Chat" }))).toBe(
      "Answering in a conversation",
    );
    expect(runOrigin(t, run({ trigger: "Manual" }))).toBe("Started by a person");
    expect(runOrigin(t, run({ trigger: "Event" }))).toBe("Started by an event");
  });

  it("gives a reader who may not read runs the items alone", () => {
    const rows = workingRows(null, [item("wt_1", work())], () => "/desk/needs");

    expect(rows.map((row) => row.key)).toEqual(["wt_1"]);
  });
});
