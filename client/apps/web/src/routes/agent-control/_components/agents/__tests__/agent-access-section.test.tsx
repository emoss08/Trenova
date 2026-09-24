import type { AgentAudienceSuggestion } from "@/lib/graphql/agent-access";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentAccessSection, coverageLine, rankAudience } from "../agent-access-section";
import { agentFormDefaults, type AgentFormValues } from "../agent-form-schema";

const fetchSuggestedAgentAudience = vi.fn<(agentId: string) => Promise<AgentAudienceSuggestion>>();

vi.mock("@/lib/graphql/agent-access", () => ({
  fetchSuggestedAgentAudience: (agentId: string) => fetchSuggestedAgentAudience(agentId),
  fetchRoleAgents: vi.fn(),
}));

// The role picker reads its options from the server; the section only hands
// it the field, which is what is asserted here.
vi.mock("@/components/autocomplete-fields", () => ({
  RoleAutocompleteField: ({ label }: { label: string }) => (
    <div data-testid="role-picker">{label}</div>
  ),
}));

afterEach(() => {
  cleanup();
  fetchSuggestedAgentAudience.mockReset();
});

const role = (id: string, name: string) => ({ id, name, description: "", isSystem: false });

function suggestion(overrides: Partial<AgentAudienceSuggestion> = {}): AgentAudienceSuggestion {
  return {
    agentId: "agdef_1",
    accessMode: "Roles",
    sensitiveTools: [],
    roles: [
      {
        role: role("role_ops", "Operations"),
        coverage: "None",
        missingResources: [],
        granted: false,
      },
      {
        role: role("role_billing", "Billing"),
        coverage: "Partial",
        missingResources: ["payroll", "settlement"],
        granted: false,
      },
      { role: role("role_admin", "Admin"), coverage: "Full", missingResources: [], granted: true },
    ],
    ...overrides,
  };
}

function RoleIds() {
  const ids = useWatch<AgentFormValues, "accessRoleIds">({ name: "accessRoleIds" });
  return <output data-testid="role-ids">{ids.join(",")}</output>;
}

function Harness({ values, children }: { values: Partial<AgentFormValues>; children: ReactNode }) {
  const form = useForm<AgentFormValues>({ defaultValues: { ...agentFormDefaults, ...values } });
  return (
    <FormProvider {...form}>
      {children}
      <RoleIds />
    </FormProvider>
  );
}

function renderSection({
  values = {},
  mode = "edit",
  agentId = "agdef_1",
  isSystem = false,
}: {
  values?: Partial<AgentFormValues>;
  mode?: "create" | "edit";
  agentId?: string;
  isSystem?: boolean;
} = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Harness values={values}>
        <AgentAccessSection mode={mode} agentId={agentId} isSystem={isSystem} />
      </Harness>
    </QueryClientProvider>,
  );
}

describe("rankAudience", () => {
  it("puts roles that can use all of it first, then partial, then none, by name within each", () => {
    const ranked = rankAudience([
      ...suggestion().roles,
      {
        role: role("role_acct", "Accounting"),
        coverage: "Full",
        missingResources: [],
        granted: false,
      },
    ]);

    expect(ranked.map((entry) => entry.role.name)).toEqual([
      "Accounting",
      "Admin",
      "Billing",
      "Operations",
    ]);
  });
});

describe("coverageLine", () => {
  const t = ((text: string, ...args: unknown[]) =>
    args.reduce<string>(
      (out, arg, index) => out.replace(`{${index}}`, String(arg)),
      text,
    )) as never;

  it("says what each coverage means, naming what a partial role is missing once each", () => {
    expect(coverageLine({ coverage: "Full", missingResources: [] }, t)).toBe(
      "Can use all of its tools",
    );
    expect(
      coverageLine(
        { coverage: "Partial", missingResources: ["payroll", "payroll", "settlement"] },
        t,
      ),
    ).toBe("Missing: Payroll, Settlement");
    expect(coverageLine({ coverage: "None", missingResources: [] }, t)).toBe(
      "Can't use the assistant",
    );
  });
});

describe("AgentAccessSection", () => {
  it("shows a system agent as open to everyone, the other choice disabled, with the reason", () => {
    renderSection({ isSystem: true, values: { accessMode: "Everyone" } });

    expect(
      screen.getByRole("radio", { name: /everyone who can use the assistant/i }),
    ).toHaveAttribute("aria-checked", "true");
    expect(screen.getByRole("radio", { name: /specific roles/i })).toBeDisabled();
    expect(screen.getByText(/a system agent is open to everyone/i)).toBeInTheDocument();
    expect(screen.queryByTestId("role-picker")).toBeNull();
    expect(fetchSuggestedAgentAudience).not.toHaveBeenCalled();
  });

  it("offers the role picker only once it is limited to specific roles", async () => {
    fetchSuggestedAgentAudience.mockResolvedValue(suggestion({ accessMode: "Everyone" }));
    renderSection({ values: { accessMode: "Everyone" } });

    expect(screen.queryByTestId("role-picker")).toBeNull();

    await userEvent.click(screen.getByRole("radio", { name: /specific roles/i }));

    expect(screen.getByTestId("role-picker")).toHaveTextContent("Roles");
    expect(screen.getByText(/nobody can use this agent until one is/i)).toBeInTheDocument();
  });

  it("lists suggested roles with their coverage, and adds one to the chosen roles", async () => {
    fetchSuggestedAgentAudience.mockResolvedValue(suggestion());
    renderSection({ values: { accessMode: "Roles", accessRoleIds: ["role_admin"] } });

    const list = await screen.findByRole("list", { name: /suggested roles/i });
    const rows = within(list).getAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual([
      expect.stringContaining("Admin"),
      expect.stringContaining("Billing"),
      expect.stringContaining("Operations"),
    ]);
    expect(within(rows[0]).getByText("Can use all of its tools")).toBeInTheDocument();
    expect(within(rows[0]).getByText("Chosen")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Missing: Payroll, Settlement")).toBeInTheDocument();
    expect(within(rows[2]).getByText("Can't use the assistant")).toBeInTheDocument();
    expect(within(rows[2]).getByRole("button", { name: /add operations/i })).toBeDisabled();

    await userEvent.click(within(rows[1]).getByRole("button", { name: /add billing/i }));

    expect(screen.getByTestId("role-ids")).toHaveTextContent("role_admin,role_billing");
    expect(within(rows[1]).getByText("Chosen")).toBeInTheDocument();
  });

  it("warns when an agent open to everyone holds tools that reach sensitive data", async () => {
    fetchSuggestedAgentAudience.mockResolvedValue(
      suggestion({ accessMode: "Everyone", sensitiveTools: ["list_worker_pay"] }),
    );
    renderSection({ values: { accessMode: "Everyone" } });

    expect(
      await screen.findByText(/open to everyone, with tools that reach sensitive data/i),
    ).toBeInTheDocument();
  });

  it("does not warn about sensitive tools once it is limited to roles", async () => {
    fetchSuggestedAgentAudience.mockResolvedValue(
      suggestion({ accessMode: "Everyone", sensitiveTools: ["list_worker_pay"] }),
    );
    renderSection({ values: { accessMode: "Roles", accessRoleIds: ["role_admin"] } });

    await screen.findByRole("list", { name: /suggested roles/i });
    expect(screen.queryByText(/tools that reach sensitive data/i)).toBeNull();
  });

  it("asks for suggestions only once the agent is saved", async () => {
    renderSection({ mode: "create", agentId: "", values: { accessMode: "Roles" } });

    expect(screen.getByText(/suggestions appear once the agent is saved/i)).toBeInTheDocument();
    expect(fetchSuggestedAgentAudience).not.toHaveBeenCalled();
  });

  it("keeps roles chosen while it is open to everyone, and says so", () => {
    fetchSuggestedAgentAudience.mockResolvedValue(suggestion({ accessMode: "Everyone" }));
    renderSection({
      values: { accessMode: "Everyone", accessRoleIds: ["role_admin", "role_billing"] },
    });

    expect(
      screen.getByText(/2 roles stay chosen for when it is limited to specific roles again/i),
    ).toBeInTheDocument();
  });
});
