import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { policy, safety, tool } from "./fixtures";

const fetchAgentToolPolicies = vi.fn();
const fetchAgentSafety = vi.fn();

vi.mock("@/lib/graphql/agent-safety", () => ({
  fetchAgentToolPolicies: (...args: unknown[]) => fetchAgentToolPolicies(...args),
  fetchAgentSafety: (...args: unknown[]) => fetchAgentSafety(...args),
}));

const { default: SafetyTab } = await import("../safety-tab");

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  fetchAgentToolPolicies.mockReset();
  fetchAgentSafety.mockReset();
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

function renderTab() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(<SafetyTab />, { wrapper });
}

const policies = [
  policy({ name: "assign_move", title: "Assign move" }),
  policy({
    name: "email_customer",
    title: "Email customer",
    egress: ["ExternalRecipient"],
    leavesOrganization: true,
    promotableTier: "ActWithApproval",
    needs: { resource: "customer", operation: "update" },
    rationale: "An email reaches a customer and cannot be taken back.",
  }),
  policy({
    name: "get_inbound_message",
    title: "Get inbound message",
    kind: "Query",
    effect: "Lookup",
    egress: ["None"],
    readsExternal: "Always",
    source: "InboundMessage",
  }),
];

describe("SafetyTab", () => {
  // Every agent is read, not a page of them: the figure for open agents
  // holding sensitive tools is only true over all of them.
  it("reads every agent and says what runs without a person", async () => {
    fetchAgentToolPolicies.mockResolvedValue(policies);
    fetchAgentSafety.mockResolvedValue([
      safety("agdef_1", "Customer desk", {
        tools: [
          tool("assign_move", { answer: "RUNS_ON_ITS_OWN" }),
          tool(
            "email_customer",
            { answer: "NEEDS_APPROVAL", tier: "ActWithApproval", heldBy: ["egress_class"] },
            {
              answer: "NEEDS_APPROVAL",
              tier: "ActWithApproval",
              heldBy: ["egress_class", "tainted"],
            },
          ),
        ],
        reach: {
          accessMode: "Everyone",
          roles: [],
          warnings: [{ kind: "OpenWithSensitiveTools", tools: ["email_customer"] }],
        },
      }),
    ]);

    renderTab();

    const figures = await screen.findByRole("group", { name: "AI safety figures" });
    expect(fetchAgentSafety).toHaveBeenCalledWith(undefined, expect.anything());
    expect(within(figures).getByText("Tools that run without a person")).toBeTruthy();
    expect(within(figures).getAllByText("1")).toHaveLength(3);

    const rules = screen.getByRole("region", { name: "Tool rules" });
    expect(within(rules).getByText("email_customer")).toBeTruthy();
    expect(within(rules).getByText("Outside recipient")).toBeTruthy();
    expect(within(rules).getByText("Always, from inbound messages")).toBeTruthy();
    expect(
      within(rules).getByText("An email reaches a customer and cannot be taken back."),
    ).toBeTruthy();

    const byAgent = screen.getByRole("region", { name: "By agent" });
    expect(within(byAgent).getByText("Customer desk")).toBeTruthy();
    expect(within(byAgent).getByText("Everyone who can use the assistant")).toBeTruthy();
    expect(within(byAgent).getByText(/leave the organization: Email customer\./)).toBeTruthy();
    expect(within(byAgent).getAllByText("Needs approval")).toHaveLength(2);
    expect(within(byAgent).getByText("Read outside text")).toBeTruthy();
    expect(within(byAgent).getAllByText("Runs on its own")).toHaveLength(2);
  });

  it("says so when the rules cannot be loaded", async () => {
    fetchAgentToolPolicies.mockRejectedValue(new Error("offline"));
    fetchAgentSafety.mockResolvedValue([]);

    renderTab();

    expect(
      await screen.findByText(
        "What agents can do without a person could not be loaded. Try again shortly.",
      ),
    ).toBeTruthy();
  });

  it("says when there are no agents yet", async () => {
    fetchAgentToolPolicies.mockResolvedValue(policies);
    fetchAgentSafety.mockResolvedValue([]);

    renderTab();

    expect(await screen.findByText("No agents yet.")).toBeTruthy();
  });
});
