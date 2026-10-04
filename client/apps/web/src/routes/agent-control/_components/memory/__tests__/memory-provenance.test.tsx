import type { AgentMemoryRow } from "@/lib/graphql/agent-memories";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { MemoryProvenance } from "../memory-provenance";

afterEach(cleanup);

function memory(overrides: Partial<AgentMemoryRow>): AgentMemoryRow {
  return {
    id: "amem_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    kind: "Procedure",
    source: "Reflection",
    status: "Active",
    subjectType: null,
    subjectId: null,
    subjectLabel: "",
    toolName: "",
    content: "Read the move before assigning it.",
    agentDefinitionId: "agdef_1",
    tainted: false,
    sourceRunId: null,
    sourceProposalId: null,
    sourceThreadId: null,
    reflectionId: "arfl_1",
    supersedesId: null,
    supersedes: null,
    replacedBy: null,
    createdByUserId: null,
    retiredByUserId: null,
    retiredAt: null,
    expiresAt: null,
    useCount: 0,
    lastUsedAt: null,
    evidence: null,
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    ...overrides,
  } as AgentMemoryRow;
}

function renderProvenance(row: AgentMemoryRow | null) {
  return render(
    <MemoryRouter>
      <MemoryProvenance memory={row} />
    </MemoryRouter>,
  );
}

/*
An agent keeps lessons on its own. Whoever reviews one in AI Control has to see
why it was kept and what it stands in for, or they cannot judge it.
*/
describe("MemoryProvenance", () => {
  it("draws nothing for a memory being created", () => {
    const { container } = renderProvenance(null);

    expect(container).toBeEmptyDOMElement();
  });

  it("shows why a learned memory was kept, what made the agent look and what was said", () => {
    renderProvenance(
      memory({
        sourceThreadId: "athr_1",
        evidence: {
          feedbackIds: [],
          patternKey: "",
          ratingCount: 0,
          distinctUsers: 0,
          distinctThreads: 0,
          reason: "Assigning failed until the move was read.",
          quotes: ["Read it first next time"],
          firstRatedAt: 0,
          lastRatedAt: 0,
          signals: ["ToolRecovered", "PersonCorrected"],
        },
      }),
    );

    expect(screen.getByText("Learned from work")).toBeInTheDocument();
    expect(screen.getByText("Assigning failed until the move was read.")).toBeInTheDocument();
    const signals = screen.getByRole("list", { name: "What made the agent look back" });
    expect(within(signals).getByText("A tool worked after failing")).toBeInTheDocument();
    expect(within(signals).getByText("A person corrected it")).toBeInTheDocument();
    expect(screen.getByText("Read it first next time")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open the conversation" })).toHaveAttribute(
      "href",
      "/desk/t/athr_1",
    );
    expect(screen.queryByRole("link", { name: "Open the run" })).not.toBeInTheDocument();
    expect(screen.queryByText("Ratings")).not.toBeInTheDocument();
  });

  it("names the memory it replaced and the one that replaced it", () => {
    renderProvenance(
      memory({
        supersedesId: "amem_old",
        supersedes: {
          id: "amem_old",
          content: "Assign the move straight away.",
          status: "Retired",
        },
        replacedBy: {
          id: "amem_new",
          content: "Read the move and its stops first.",
          status: "Active",
          createdAt: 1_790_100_000,
        },
      }),
    );

    expect(screen.getByText("Replaces")).toBeInTheDocument();
    expect(screen.getByText("Assign the move straight away.")).toBeInTheDocument();
    expect(screen.getByText("Replaced by")).toBeInTheDocument();
    expect(screen.getByText("Read the move and its stops first.")).toBeInTheDocument();
  });

  it("links a memory drawn from a run to that run, and marks outside content", () => {
    renderProvenance(memory({ source: "Agent", sourceRunId: "arun_1", tainted: true }));

    expect(screen.getByRole("link", { name: "Open the run" }).getAttribute("href")).toContain(
      "/admin/agent-control",
    );
    expect(screen.getByText("Read outside content")).toBeInTheDocument();
    expect(screen.queryByText("Why it was kept")).not.toBeInTheDocument();
    expect(screen.queryByText("Replaces")).not.toBeInTheDocument();
  });

  it("counts the ratings behind a memory drawn from feedback", () => {
    renderProvenance(
      memory({
        source: "Feedback",
        reflectionId: null,
        evidence: {
          feedbackIds: ["afb_1", "afb_2", "afb_3"],
          patternKey: "wrong-bill-to",
          ratingCount: 3,
          distinctUsers: 2,
          distinctThreads: 3,
          reason: "",
          quotes: [],
          firstRatedAt: 1_789_000_000,
          lastRatedAt: 1_790_000_000,
          signals: [],
        },
      }),
    );

    expect(screen.getByText(/3 ratings/)).toBeInTheDocument();
    expect(screen.getByText(/2 people/)).toBeInTheDocument();
    expect(
      screen.queryByRole("list", { name: "What made the agent look back" }),
    ).not.toBeInTheDocument();
  });
});
