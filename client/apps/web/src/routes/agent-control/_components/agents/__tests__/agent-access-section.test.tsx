import type { AgentAccessPreview, AgentAccessPreviewRequest } from "@/lib/graphql/agent-access";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { FormProvider, useForm, useFormContext, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentAccessSection, coverageLine, rankAudience } from "../agent-access-section";
import { agentFormDefaults, type AgentFormValues } from "../agent-form-schema";

const fetchAgentAccessPreview =
  vi.fn<(request: AgentAccessPreviewRequest) => Promise<AgentAccessPreview>>();

vi.mock("@/lib/graphql/agent-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-access")>()),
  fetchAgentAccessPreview: (request: AgentAccessPreviewRequest) => fetchAgentAccessPreview(request),
  fetchSuggestedAgentAudience: vi.fn(),
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
  fetchAgentAccessPreview.mockReset();
});

const role = (id: string, name: string) => ({ id, name, description: "", isSystem: false });

function suggestion(overrides: Partial<AgentAccessPreview> = {}): AgentAccessPreview {
  return {
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

/** Stands in for the tool picker: adds a tool to the form, as ticking one would. */
function AddTool({ name }: { name: string }) {
  const { getValues, setValue } = useFormContext<AgentFormValues>();
  return (
    <button
      type="button"
      onClick={() =>
        setValue("toolNames", [...getValues("toolNames"), name], { shouldDirty: true })
      }
    >
      Add {name}
    </button>
  );
}

/**
 * The server's answer to a preview, from the request as sent: sensitive tools
 * only while the form says everyone may use it, as the contract says.
 */
function previewFor(sensitive: readonly string[]) {
  return async (request: AgentAccessPreviewRequest): Promise<AgentAccessPreview> =>
    suggestion({
      accessMode: request.accessMode,
      sensitiveTools:
        request.accessMode === "Everyone"
          ? request.toolNames.filter((name) => sensitive.includes(name))
          : [],
    });
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
        <AddTool name="list_worker_pay" />
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
    expect(fetchAgentAccessPreview).not.toHaveBeenCalled();
  });

  it("offers the role picker only once it is limited to specific roles", async () => {
    fetchAgentAccessPreview.mockResolvedValue(suggestion({ accessMode: "Everyone" }));
    renderSection({ values: { accessMode: "Everyone" } });

    expect(screen.queryByTestId("role-picker")).toBeNull();

    await userEvent.click(screen.getByRole("radio", { name: /specific roles/i }));

    expect(screen.getByTestId("role-picker")).toHaveTextContent("Roles");
    expect(screen.getByText(/nobody can use this agent until one is/i)).toBeInTheDocument();
  });

  it("lists suggested roles with their coverage, and adds one to the chosen roles", async () => {
    fetchAgentAccessPreview.mockResolvedValue(suggestion());
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
    fetchAgentAccessPreview.mockResolvedValue(
      suggestion({ accessMode: "Everyone", sensitiveTools: ["list_worker_pay"] }),
    );
    renderSection({ values: { accessMode: "Everyone" } });

    expect(
      await screen.findByText(/open to everyone, with tools that reach sensitive data/i),
    ).toBeInTheDocument();
  });

  it("does not warn about sensitive tools once it is limited to roles", async () => {
    fetchAgentAccessPreview.mockResolvedValue(
      suggestion({ accessMode: "Everyone", sensitiveTools: ["list_worker_pay"] }),
    );
    renderSection({ values: { accessMode: "Roles", accessRoleIds: ["role_admin"] } });

    await screen.findByRole("list", { name: /suggested roles/i });
    expect(screen.queryByText(/tools that reach sensitive data/i)).toBeNull();
  });

  // An agent not yet saved has nothing saved to read, so the suggestions are
  // worked out from the form alone and no agent is named.
  it("suggests roles for an agent not yet saved, from the tools on the form", async () => {
    fetchAgentAccessPreview.mockResolvedValue(suggestion());
    renderSection({
      mode: "create",
      agentId: "",
      values: { accessMode: "Roles", toolNames: ["list_shipments"] },
    });

    await screen.findByRole("list", { name: /suggested roles/i });
    expect(fetchAgentAccessPreview).toHaveBeenCalledWith({
      agentId: "",
      accessMode: "Roles",
      toolNames: ["list_shipments"],
    });
  });

  // The warning followed the saved agent, so an agent saved restricted and
  // switched back to everyone on screen said nothing until it was saved.
  it("warns as soon as a restricted agent is opened to everyone, before it is saved", async () => {
    fetchAgentAccessPreview.mockImplementation(previewFor(["list_worker_pay"]));
    renderSection({
      values: {
        accessMode: "Roles",
        accessRoleIds: ["role_admin"],
        toolNames: ["list_worker_pay"],
      },
    });
    await screen.findByRole("list", { name: /suggested roles/i });
    expect(screen.queryByText(/tools that reach sensitive data/i)).toBeNull();

    await userEvent.click(
      screen.getByRole("radio", { name: /everyone who can use the assistant/i }),
    );

    expect(
      await screen.findByText(/open to everyone, with tools that reach sensitive data/i),
    ).toBeInTheDocument();
    expect(fetchAgentAccessPreview).toHaveBeenLastCalledWith({
      agentId: "agdef_1",
      accessMode: "Everyone",
      toolNames: ["list_worker_pay"],
    });
  });

  it("warns about a sensitive tool as soon as it is added on screen", async () => {
    fetchAgentAccessPreview.mockImplementation(previewFor(["list_worker_pay"]));
    renderSection({ values: { accessMode: "Everyone", toolNames: ["list_shipments"] } });
    await waitFor(() => expect(fetchAgentAccessPreview).toHaveBeenCalledTimes(1));
    expect(screen.queryByText(/tools that reach sensitive data/i)).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: /add list_worker_pay/i }));

    expect(
      await screen.findByText(/open to everyone, with tools that reach sensitive data/i),
    ).toBeInTheDocument();
    expect(fetchAgentAccessPreview).toHaveBeenLastCalledWith({
      agentId: "agdef_1",
      accessMode: "Everyone",
      toolNames: ["list_shipments", "list_worker_pay"],
    });
  });

  it("keeps roles chosen while it is open to everyone, and says so", () => {
    fetchAgentAccessPreview.mockResolvedValue(suggestion({ accessMode: "Everyone" }));
    renderSection({
      values: { accessMode: "Everyone", accessRoleIds: ["role_admin", "role_billing"] },
    });

    expect(
      screen.getByText(/2 roles stay chosen for when it is limited to specific roles again/i),
    ).toBeInTheDocument();
  });
});
