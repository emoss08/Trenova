import { stubLayout } from "@/test/layout";
import type { ToolCatalogEntry, ToolTrust } from "@/types/assistant";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { describeTrust } from "../track-record";
import { TrackRecord } from "../track-record";

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

function row(overrides: Partial<ToolTrust>): ToolTrust {
  return {
    id: "att_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    agentDefinitionId: "agdef_1",
    toolName: "assign_move",
    streak: 0,
    approvals: 0,
    modifications: 0,
    rejections: 0,
    executionFailures: 0,
    earnedTier: null,
    lastDecisionAt: null,
    promotedAt: null,
    demotedAt: null,
    version: 0,
    createdAt: 0,
    updatedAt: 0,
    ...overrides,
  };
}

function tool(overrides: Partial<ToolCatalogEntry>): ToolCatalogEntry {
  return {
    name: "assign_move",
    description: "Assigns a driver",
    parameters: null,
    kind: "action",
    resource: "shipment_move",
    operation: "assign",
    defaultAutonomyTier: "ActWithApproval",
    reversible: true,
    core: false,
    prerequisites: [],
    extension: "",
    grantedToEveryAgent: false,
    ...overrides,
  };
}

/**
 * The ledger is read by an administrator deciding whether to trust an agent
 * more. Each state names what stands between the tool and the next tier, so
 * the page never shows a streak that can go nowhere as if it were progress.
 */
describe("describeTrust", () => {
  const settings = { earnedAutonomy: true, threshold: 10, ceiling: "AutoExecute" as const };

  it("counts toward the threshold while the organization allows promotion", () => {
    const summary = describeTrust(row({ streak: 4 }), "ActWithApproval", settings);

    expect(summary).toEqual({ state: "counting", remaining: 6, nextTier: "AutoExecute" });
  });

  it("says the tier was earned when the ledger granted the current one", () => {
    const summary = describeTrust(
      row({ streak: 2, earnedTier: "AutoExecute", promotedAt: 1_700_000_000 }),
      "AutoExecute",
      settings,
    );

    expect(summary.state).toBe("earned");
  });

  it("stops at the agent's ceiling instead of counting toward a tier it cannot reach", () => {
    const summary = describeTrust(row({ streak: 9 }), "ActWithApproval", {
      ...settings,
      ceiling: "ActWithApproval",
    });

    expect(summary.state).toBe("at-ceiling");
  });

  it("stops at the top tier", () => {
    const summary = describeTrust(row({ streak: 9 }), "AutoExecute", settings);

    expect(summary.state).toBe("at-ceiling");
  });

  it("reports the switch as off rather than a count that will never promote", () => {
    const summary = describeTrust(row({ streak: 9 }), "ActWithApproval", {
      ...settings,
      earnedAutonomy: false,
    });

    expect(summary.state).toBe("off");
  });
});

describe("TrackRecord", () => {
  it("says so when nothing has been decided yet", () => {
    render(
      <TrackRecord
        rows={[]}
        tools={[tool({})]}
        tiers={{}}
        ceiling="AutoExecute"
        earnedAutonomy
        threshold={10}
      />,
    );

    expect(screen.getByText(/no decisions yet/i)).toBeInTheDocument();
  });

  it("shows each tool's streak against the threshold and its totals", () => {
    render(
      <TrackRecord
        rows={[row({ streak: 4, approvals: 12, modifications: 1, rejections: 2 })]}
        tools={[tool({})]}
        tiers={{}}
        ceiling="AutoExecute"
        earnedAutonomy
        threshold={10}
      />,
    );

    expect(screen.getByRole("progressbar", { name: /assign move/i })).toHaveAttribute(
      "aria-valuenow",
      "4",
    );
    expect(screen.getByText("6 more clean approvals to Automatic")).toBeInTheDocument();
    expect(screen.getByText("12 approved · 1 changed · 2 rejected · 0 failed")).toBeInTheDocument();
  });

  it("marks a tier the ledger granted", () => {
    render(
      <TrackRecord
        rows={[row({ earnedTier: "AutoExecute", promotedAt: 1_700_000_000 })]}
        tools={[tool({})]}
        tiers={{ assign_move: "AutoExecute" }}
        ceiling="AutoExecute"
        earnedAutonomy
        threshold={10}
      />,
    );

    expect(screen.getByText(/earned/i)).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });
});
