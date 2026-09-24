import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { RoleAgent, RoleAgents } from "@/lib/graphql/agent-access";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RoleAgentsSection, sortRoleAgents } from "../role-agents-section";

const fetchRoleAgents = vi.fn<(roleId: string) => Promise<RoleAgents | null>>();
const setRoleAgentAccess =
  vi.fn<(roleId: string, agentIds: readonly string[]) => Promise<RoleAgents>>();

vi.mock("@/lib/graphql/agent-access", () => ({
  fetchRoleAgents: (roleId: string) => fetchRoleAgents(roleId),
  fetchSuggestedAgentAudience: vi.fn(),
  setRoleAgentAccess: (roleId: string, agentIds: readonly string[]) =>
    setRoleAgentAccess(roleId, agentIds),
}));

// The picker is its own component with its own tests; here it only has to
// say what it was asked to list and hand back an agent.
const pickerProps = vi.fn();
vi.mock("@/components/assistant/agent-picker", () => ({
  AgentPickerList: (props: {
    source: string;
    hiddenIds: ReadonlySet<string>;
    exclude: (agent: AgentChoice) => boolean;
    onSelect: (agent: AgentChoice) => void;
  }) => {
    pickerProps(props);
    return (
      <button type="button" onClick={() => props.onSelect(choice("agdef_payroll", "Payroll desk"))}>
        Pick Payroll desk
      </button>
    );
  },
}));

afterEach(() => {
  cleanup();
  fetchRoleAgents.mockReset();
  setRoleAgentAccess.mockReset();
  pickerProps.mockReset();
});

function choice(id: string, name: string, systemKey = ""): AgentChoice {
  return {
    id,
    name,
    description: "",
    template: null,
    icon: "",
    accent: "",
    toolNames: [],
    systemKey,
    starters: [],
  };
}

function granted(id: string, name: string, overrides: Partial<RoleAgent> = {}): RoleAgent {
  return {
    id,
    name,
    icon: "",
    accent: "",
    template: null,
    accessMode: "Roles",
    enabled: true,
    triggerMode: "Chat",
    systemKey: "",
    ...overrides,
  };
}

function renderSection() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RoleAgentsSection roleId="role_billing" />
    </QueryClientProvider>,
  );
}

describe("sortRoleAgents", () => {
  it("lists the agents by name", () => {
    expect(
      sortRoleAgents([granted("b", "Settlements"), granted("a", "Billing")]).map((a) => a.name),
    ).toEqual(["Billing", "Settlements"]);
  });
});

describe("RoleAgentsSection", () => {
  it("lists the agents the role grants, marking one open to everyone and one disabled", async () => {
    fetchRoleAgents.mockResolvedValue({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [
        granted("agdef_s", "Settlements", { enabled: false }),
        granted("agdef_h", "Help desk", { accessMode: "Everyone" }),
      ],
    });

    renderSection();

    const list = await screen.findByRole("list", { name: /agents this role grants/i });
    const rows = within(list).getAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual([
      expect.stringContaining("Help desk"),
      expect.stringContaining("Settlements"),
    ]);
    expect(within(rows[0]).getByText("Open to everyone")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Disabled")).toBeInTheDocument();
    expect(fetchRoleAgents).toHaveBeenCalledWith("role_billing");
  });

  it("says that agents open to everyone are included without being added", async () => {
    fetchRoleAgents.mockResolvedValue({ roleId: "role_billing", roleName: "Billing", agents: [] });

    renderSection();

    expect(
      await screen.findByText(/this role is granted no agents limited to specific roles/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/open to everyone who can use the assistant are included for this role/i),
    ).toBeInTheDocument();
  });

  it("adds an agent by saving the whole list with it", async () => {
    fetchRoleAgents.mockResolvedValue({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [granted("agdef_s", "Settlements")],
    });
    setRoleAgentAccess.mockResolvedValue({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [granted("agdef_payroll", "Payroll desk"), granted("agdef_s", "Settlements")],
    });

    renderSection();
    await screen.findByRole("list", { name: /agents this role grants/i });

    await userEvent.click(screen.getByRole("button", { name: /add an agent/i }));
    await userEvent.click(await screen.findByRole("button", { name: "Pick Payroll desk" }));

    await waitFor(() =>
      expect(setRoleAgentAccess).toHaveBeenCalledWith("role_billing", ["agdef_s", "agdef_payroll"]),
    );
    expect(await screen.findByText("Payroll desk")).toBeInTheDocument();
  });

  it("offers every agent a role could hold, leaving out those already granted and system agents", async () => {
    fetchRoleAgents.mockResolvedValue({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [granted("agdef_s", "Settlements")],
    });

    renderSection();
    await screen.findByRole("list", { name: /agents this role grants/i });
    await userEvent.click(screen.getByRole("button", { name: /add an agent/i }));
    await screen.findByRole("button", { name: "Pick Payroll desk" });

    const props = pickerProps.mock.calls.at(-1)?.[0] as {
      source: string;
      hiddenIds: ReadonlySet<string>;
      exclude: (agent: AgentChoice) => boolean;
    };
    expect(props.source).toBe("grantable");
    expect([...props.hiddenIds]).toEqual(["agdef_s"]);
    expect(props.exclude(choice("agdef_x", "Billing exceptions", "billing_exception"))).toBe(true);
    expect(props.exclude(choice("agdef_y", "Payroll desk"))).toBe(false);
  });

  it("removes an agent by saving the list without it", async () => {
    fetchRoleAgents.mockResolvedValue({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [granted("agdef_s", "Settlements"), granted("agdef_p", "Payroll desk")],
    });
    setRoleAgentAccess.mockResolvedValue({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [granted("agdef_s", "Settlements")],
    });

    renderSection();
    await screen.findByRole("list", { name: /agents this role grants/i });

    await userEvent.click(screen.getByRole("button", { name: "Remove Payroll desk" }));

    await waitFor(() =>
      expect(setRoleAgentAccess).toHaveBeenCalledWith("role_billing", ["agdef_s"]),
    );
    await waitFor(() => expect(screen.queryByText("Payroll desk")).toBeNull());
  });

  it("reads the list again when a change is refused, rather than showing the change", async () => {
    fetchRoleAgents.mockResolvedValue({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [granted("agdef_s", "Settlements")],
    });
    setRoleAgentAccess.mockRejectedValue(
      new Error("Only a person can change who may use an agent"),
    );

    renderSection();
    await screen.findByRole("list", { name: /agents this role grants/i });

    await userEvent.click(screen.getByRole("button", { name: "Remove Settlements" }));

    await waitFor(() => expect(fetchRoleAgents).toHaveBeenCalledTimes(2));
    expect(screen.getByText("Settlements")).toBeInTheDocument();
  });

  it("offers a retry when the list cannot be read", async () => {
    fetchRoleAgents.mockRejectedValueOnce(new Error("boom"));
    fetchRoleAgents.mockResolvedValueOnce({
      roleId: "role_billing",
      roleName: "Billing",
      agents: [granted("agdef_s", "Settlements")],
    });

    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: /try again/i }));

    expect(await screen.findByText("Settlements")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add an agent/i })).toBeEnabled();
  });
});
