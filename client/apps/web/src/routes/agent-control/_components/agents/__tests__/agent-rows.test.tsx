import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AgentRow } from "../agent-rows";

/**
 * The row's controls are what an operator uses to run this thing, so the test
 * asks for them the way a person does: by their accessible name.
 */
function agentFixture(overrides: Partial<AgentDefinitionRow> = {}): AgentDefinitionRow {
  return {
    id: "agdef_01JAGENT",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    name: "Night dispatch helper",
    description: "Looks up shipments and drivers.",
    template: "DispatchAssistant",
    icon: "",
    accent: "",
    instructions: "",
    guardrails: [],
    toolNames: ["get_shipment"],
    toolTiers: {},
    autonomyCeiling: "Propose",
    dataAccessCeiling: "Internal",
    enabled: true,
    shadowMode: false,
    decisionTimeoutSeconds: 86400,
    triggerMode: "Chat",
    cronExpression: "",
    cronTimezone: "",
    eventKinds: [],
    intervalSeconds: 0,
    endsAt: null,
    maxConcurrentRuns: 1,
    runTimeoutSeconds: 600,
    maxToolCalls: 12,
    contextProviders: [],
    outputMode: "Conversational",
    preferredProviderId: "",
    systemKey: "",
    delegateIds: [],
    delegates: [],
    accessMode: "Everyone",
    accessRoles: [],
    lastRunAt: null,
    nextRunAt: null,
    pendingProposals: 0,
    openRuns: 0,
    version: 1,
    createdAt: 1_758_000_000,
    updatedAt: 1_758_000_100,
    ...overrides,
  } as AgentDefinitionRow;
}

function renderRow(overrides: Partial<AgentDefinitionRow> = {}) {
  const onDelete = vi.fn();
  const onEdit = vi.fn();
  const onRunNow = vi.fn();
  const onToggleEnabled = vi.fn();

  render(
    <ul>
      <AgentRow
        agent={agentFixture(overrides)}
        templates={[]}
        actions={{
          canUpdate: true,
          canDelete: true,
          canRun: true,
          isToggling: () => false,
          isRunning: () => false,
          onEdit,
          onToggleEnabled,
          onRunNow,
          onDelete,
        }}
      />
    </ul>,
  );

  return { onDelete, onEdit, onRunNow, onToggleEnabled };
}

describe("AgentRow", () => {
  /**
   * The control used to render as an empty square: the icon was passed as a
   * child of the tooltip trigger, which replaced the button's own children, so
   * a person saw a blank button where a bin belongs. Asking only for the role
   * would not have caught that, because the button itself was always there.
   */
  it("offers a remove control that is visible and fires", async () => {
    const { onDelete } = renderRow();

    const remove = screen.getByRole("button", { name: /remove agent/i });
    expect(remove.querySelector("svg"), "the remove control renders no icon").not.toBeNull();

    await userEvent.click(remove);

    expect(onDelete).toHaveBeenCalledOnce();
  });

  it("locks removal for an agent the platform owns", () => {
    renderRow({ systemKey: "billing_exception" });

    expect(screen.getByRole("button", { name: /remove agent/i })).toBeDisabled();
  });
});

describe("the access badge", () => {
  it("says Everyone for an agent open to everyone", () => {
    renderRow();

    expect(screen.getByText("Everyone")).toBeInTheDocument();
  });

  it("counts the roles an agent is limited to", () => {
    renderRow({
      accessMode: "Roles",
      accessRoles: [
        { id: "role_a", name: "Dispatch" },
        { id: "role_b", name: "Billing" },
      ],
    });

    expect(screen.getByText("2 roles")).toBeInTheDocument();
    expect(screen.queryByText("Everyone")).toBeNull();
  });

  it("says one role in the singular", () => {
    renderRow({ accessMode: "Roles", accessRoles: [{ id: "role_a", name: "Dispatch" }] });

    expect(screen.getByText("1 role")).toBeInTheDocument();
  });

  // Restricted with nobody granted is usable by nobody, which is worth a
  // second look rather than a quiet "0 roles".
  it("flags an agent limited to roles with none granted", () => {
    renderRow({ accessMode: "Roles", accessRoles: [] });

    expect(screen.getByText("No roles")).toBeInTheDocument();
  });

  // The server treats a system agent as open whatever its row says.
  it("says Everyone for a system agent whatever its row says", () => {
    renderRow({ systemKey: "billing_exception", accessMode: "Roles", accessRoles: [] });

    expect(screen.getByText("Everyone")).toBeInTheDocument();
  });
});
