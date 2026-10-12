import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { toast } from "sonner";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToolSheet } from "../tool-sheet";
import { definition, policy } from "./fixtures";

const api = vi.hoisted(() => ({
  fetchToolHolderAnswers: vi.fn(),
  fetchToolRuleImpact: vi.fn(),
  saveAgentToolRule: vi.fn(),
  fetchToolRules: vi.fn(),
}));
const permissions = vi.hoisted(() => ({ denied: new Set<string>() }));

vi.mock("@/lib/graphql/agent-safety", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-safety")>()),
  fetchToolHolderAnswers: (...args: unknown[]) => api.fetchToolHolderAnswers(...args),
  fetchToolRuleImpact: (...args: unknown[]) => api.fetchToolRuleImpact(...args),
  saveAgentToolRule: (...args: unknown[]) => api.saveAgentToolRule(...args),
  fetchToolRules: (...args: unknown[]) => api.fetchToolRules(...args),
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: !permissions.denied.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

const holders = [definition("agdef_1", "Customer desk"), definition("agdef_2", "Dispatch desk")];

const releaseHold = policy({
  name: "release_billing_hold",
  title: "Release billing hold",
  maxTier: "AutoExecute",
  promotableTier: "AutoExecute",
  declaredMaxTier: "AutoExecute",
  readsExternal: "Marked",
  declaredReadsExternal: "Marked",
  ruleVersion: 3,
});

/** What the server answers for a rule: the dispatch desk moves once the tool is held lower. */
function impactFor(_name: string, input: { maxTier?: string }) {
  const held = input.maxTier !== "AutoExecute";
  return Promise.resolve([
    {
      agentId: "agdef_1",
      agentName: "Customer desk",
      before: "PROPOSE_ONLY",
      after: "PROPOSE_ONLY",
    },
    {
      agentId: "agdef_2",
      agentName: "Dispatch desk",
      before: "RUNS_ON_ITS_OWN",
      after: held ? "NEEDS_APPROVAL" : "RUNS_ON_ITS_OWN",
    },
  ]);
}

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  permissions.denied.clear();
  api.fetchToolHolderAnswers.mockResolvedValue([]);
  api.fetchToolRuleImpact.mockImplementation(impactFor);
  api.fetchToolRules.mockResolvedValue(new Map());
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.clearAllMocks();
});

function renderSheet(rule = releaseHold) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter hasMemory>{children}</NuqsTestingAdapter>
    </QueryClientProvider>
  );
  return render(<ToolSheet policy={rule} holders={holders} onClose={() => {}} />, { wrapper });
}

async function openEditor() {
  await userEvent.click(await screen.findByRole("button", { name: "Change tool rule" }));
  return screen.findByRole("complementary", { name: "Release billing hold" });
}

describe("Tool rule editor", () => {
  it("is offered only to people who may change agent control", async () => {
    permissions.denied.add("agent_control:4");
    renderSheet();

    expect(await screen.findByText("Held by 2 agents")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Change tool rule" })).not.toBeInTheDocument();
  });

  it("closes the choices looser than the tool declares", async () => {
    renderSheet({ ...releaseHold, maxTier: "ActWithApproval", declaredMaxTier: "ActWithApproval" });
    const editor = await openEditor();

    const freedom = within(editor).getByRole("radiogroup", { name: "Most freedom any agent gets" });
    expect(within(freedom).getByRole("radio", { name: "Automatic" })).toBeDisabled();
    expect(within(freedom).getByRole("radio", { name: "Ask first" })).toBeEnabled();
    const outside = within(editor).getByRole("radiogroup", {
      name: "Treat what it returns as outside text",
    });
    expect(within(outside).getByRole("radio", { name: "Never" })).toBeDisabled();
    expect(within(outside).getByRole("radio", { name: "Always" })).toBeEnabled();
  });

  it("says who moves before saving, and holds the save for a reason", async () => {
    renderSheet();
    const editor = await openEditor();
    expect(await within(editor).findByText("0 of 2 change")).toBeInTheDocument();

    const freedom = within(editor).getByRole("radiogroup", { name: "Most freedom any agent gets" });
    await userEvent.click(within(freedom).getByRole("radio", { name: "Ask first" }));

    expect(await within(editor).findByText("1 of 2 change")).toBeInTheDocument();
    const moved = within(editor).getByText("Dispatch desk").closest(".aff-r");
    expect(moved).toHaveClass("ch");
    expect(within(moved as HTMLElement).getByText("Runs on its own").tagName).toBe("S");
    expect(within(moved as HTMLElement).getByText("Needs approval")).toBeInTheDocument();
    expect(within(editor).getByText("Add a reason for the audit trail")).toBeInTheDocument();
    expect(within(editor).getByRole("button", { name: /Save changes/ })).toBeDisabled();
  });

  it("saves the rule at the version it started from, with the reason, and says who moved", async () => {
    api.saveAgentToolRule.mockResolvedValue({
      tool: { ...releaseHold, maxTier: "ActWithApproval", ruleVersion: 4 },
      affected: [
        {
          agentId: "agdef_2",
          agentName: "Dispatch desk",
          before: "RUNS_ON_ITS_OWN",
          after: "NEEDS_APPROVAL",
        },
      ],
    });
    renderSheet();
    const editor = await openEditor();

    const freedom = within(editor).getByRole("radiogroup", { name: "Most freedom any agent gets" });
    await userEvent.click(within(freedom).getByRole("radio", { name: "Ask first" }));
    await userEvent.type(
      within(editor).getByRole("textbox", { name: "Reason" }),
      "Two wrong releases",
    );
    await userEvent.click(within(editor).getByRole("button", { name: /Save changes/ }));

    await waitFor(() =>
      expect(api.saveAgentToolRule).toHaveBeenCalledWith("release_billing_hold", 3, {
        maxTier: "ActWithApproval",
        readsExternal: "Marked",
        reason: "Two wrong releases",
      }),
    );
    expect(toast.success).toHaveBeenCalledWith("Release billing hold rule saved · 1 agent changed");
  });

  it("takes outside text without a reason", async () => {
    api.saveAgentToolRule.mockResolvedValue({
      tool: { ...releaseHold, readsExternal: "Always", ruleVersion: 4 },
      affected: [],
    });
    renderSheet();
    const editor = await openEditor();

    const outside = within(editor).getByRole("radiogroup", {
      name: "Treat what it returns as outside text",
    });
    await userEvent.click(within(outside).getByRole("radio", { name: "Always" }));
    await userEvent.click(within(editor).getByRole("button", { name: /Save changes/ }));

    await waitFor(() =>
      expect(api.saveAgentToolRule).toHaveBeenCalledWith("release_billing_hold", 3, {
        maxTier: "AutoExecute",
        readsExternal: "Always",
        reason: null,
      }),
    );
    expect(toast.success).toHaveBeenCalledWith("Release billing hold rule saved");
  });

  it("shows the organization's own rule and why it was set", async () => {
    renderSheet({
      ...releaseHold,
      maxTier: "Propose",
      rule: {
        maxTier: "Propose",
        readsExternal: null,
        reason: "Two wrong releases last week",
        updatedAt: 1_760_000_000,
        updatedBy: { id: "usr_1", name: "Ada Park" },
      },
    });

    expect(await screen.findByText("Your organization's rule")).toBeInTheDocument();
    expect(screen.getByText("Two wrong releases last week")).toBeInTheDocument();
    expect(screen.getByText(/Ada Park/)).toBeInTheDocument();
  });
});
