import type { AgentChoice } from "@/lib/graphql/agent-definition";
import {
  ALL_TOOL_POLICIES,
  type AgentSafetySummary,
  type ToolPolicyFilter,
  type ToolPolicyPage,
  type ToolPolicyPageRequest,
} from "@/lib/graphql/agent-safety";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { policy, safety, tool } from "./fixtures";

const fetchAgentSafetySummary = vi.fn<() => Promise<AgentSafetySummary>>();
const fetchToolPolicyPage =
  vi.fn<(filter: ToolPolicyFilter, page: ToolPolicyPageRequest) => Promise<ToolPolicyPage>>();
const fetchAgentSafety = vi.fn();

vi.mock("@/lib/graphql/agent-safety", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/agent-safety")>();
  return {
    ...actual,
    fetchAgentSafetySummary: () => fetchAgentSafetySummary(),
    fetchToolPolicyPage: (filter: ToolPolicyFilter, page: ToolPolicyPageRequest) =>
      fetchToolPolicyPage(filter, page),
    fetchAgentSafety: (...args: unknown[]) => fetchAgentSafety(...args),
  };
});

// The picker has its own tests; here it only has to hand back an agent.
const pickerProps = vi.fn();
vi.mock("@/components/assistant/agent-picker", () => ({
  AgentPickerList: (props: {
    source: string;
    hiddenIds: ReadonlySet<string>;
    onSelect: (agent: AgentChoice) => void;
  }) => {
    pickerProps(props);
    return (
      <button type="button" onClick={() => props.onSelect(choice("agdef_1", "Customer desk"))}>
        Pick Customer desk
      </button>
    );
  },
}));

const { default: SafetyTab } = await import("../safety-tab");

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  fetchAgentSafetySummary.mockReset();
  fetchToolPolicyPage.mockReset();
  fetchAgentSafety.mockReset();
  pickerProps.mockReset();
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

function choice(id: string, name: string): AgentChoice {
  return {
    id,
    name,
    description: "",
    template: null,
    icon: "",
    accent: "",
    toolNames: [],
    systemKey: "",
    starters: [],
  };
}

function renderTab() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(<SafetyTab />, { wrapper });
}

const summary: AgentSafetySummary = {
  toolCount: 132,
  runWithoutPerson: 4,
  leaveOrganization: 9,
  openWithSensitive: 1,
  resources: ["shipment", "general", "customer"],
};

const emailCustomer = policy({
  name: "email_customer",
  title: "Email customer",
  egress: ["ExternalRecipient"],
  leavesOrganization: true,
  promotableTier: "ActWithApproval",
  needs: { resource: "customer", operation: "update" },
  rationale: "An email reaches a customer and cannot be taken back.",
  hasCondition: true,
  conditionDescription: "Only to a contact on the customer's record.",
});

const firstPage: ToolPolicyPage = {
  items: [
    policy({ name: "assign_move", title: "Assign move" }),
    emailCustomer,
    policy({
      name: "get_inbound_message",
      title: "Get inbound message",
      kind: "Query",
      effect: "Lookup",
      egress: ["None"],
      readsExternal: "Always",
      source: "InboundMessage",
    }),
  ],
  endCursor: "cursor-3",
  hasNextPage: true,
  totalCount: 30,
};

const secondPage: ToolPolicyPage = {
  items: [policy({ name: "void_invoice", title: "Void invoice", egress: ["Money"] })],
  endCursor: "cursor-4",
  hasNextPage: false,
  totalCount: null,
};

describe("SafetyTab", () => {
  // The figures come from the server's count, and nothing about any agent is
  // read until someone picks one: an organization with two hundred agents
  // opens this tab for the price of one page of rules.
  it("counts on the server, reads one page of rules and no agent until one is picked", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchToolPolicyPage.mockResolvedValue(firstPage);

    renderTab();

    const figures = await screen.findByRole("group", { name: "AI safety figures" });
    expect(within(figures).getByText("4")).toBeTruthy();
    expect(within(figures).getByText("9")).toBeTruthy();
    expect(within(figures).getByText("1")).toBeTruthy();

    const rules = screen.getByRole("region", { name: "Tool rules" });
    expect(await within(rules).findByText("email_customer")).toBeTruthy();
    expect(within(rules).getByText("Outside recipient")).toBeTruthy();
    expect(within(rules).getByText("Always, from inbound messages")).toBeTruthy();
    expect(within(rules).getAllByText("30").length).toBeGreaterThan(0);

    expect(fetchToolPolicyPage).toHaveBeenCalledTimes(1);
    expect(fetchToolPolicyPage).toHaveBeenCalledWith(
      { ...ALL_TOOL_POLICIES, search: "" },
      { first: 25, after: null, includeTotalCount: true },
    );

    const byAgent = screen.getByRole("region", { name: "By agent" });
    expect(
      within(byAgent).getByText("Add an agent to see what it can do without a person."),
    ).toBeTruthy();
    expect(fetchAgentSafety).not.toHaveBeenCalled();
  });

  // The rationale and the condition are long; they open under the row
  // rather than widening the table.
  it("opens a rule's rationale and condition under its row", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchToolPolicyPage.mockResolvedValue(firstPage);

    renderTab();

    const rules = await screen.findByRole("region", { name: "Tool rules" });
    await within(rules).findByText("email_customer");
    expect(
      within(rules).queryByText("An email reaches a customer and cannot be taken back."),
    ).toBeNull();

    await userEvent.click(
      within(rules).getByRole("button", { name: "Show details for Email customer" }),
    );

    expect(
      within(rules).getByText("An email reaches a customer and cannot be taken back."),
    ).toBeTruthy();
    expect(within(rules).getByText("Only to a contact on the customer's record.")).toBeTruthy();
    expect(
      within(rules).getByRole("button", { name: "Hide details for Email customer" }),
    ).toBeTruthy();
  });

  // Each page is read after the cursor the page before it ended on, and the
  // count, which walks every rule, is asked for once per scope.
  it("pages forward by cursor and back from the cache", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchToolPolicyPage.mockImplementation(async (_filter, page) =>
      page.after === "cursor-3" ? secondPage : firstPage,
    );

    renderTab();

    const rules = await screen.findByRole("region", { name: "Tool rules" });
    await within(rules).findByText("email_customer");

    await userEvent.click(within(rules).getByRole("button", { name: "Go to next page" }));
    expect(await within(rules).findByText("void_invoice")).toBeTruthy();
    expect(fetchToolPolicyPage).toHaveBeenLastCalledWith(
      { ...ALL_TOOL_POLICIES, search: "" },
      { first: 25, after: "cursor-3", includeTotalCount: false },
    );
    expect(within(rules).getAllByText("30").length).toBeGreaterThan(0);

    await userEvent.click(within(rules).getByRole("button", { name: "Go to previous page" }));
    expect(await within(rules).findByText("email_customer")).toBeTruthy();
    expect(fetchToolPolicyPage).toHaveBeenCalledTimes(2);
  });

  // A search is a new scope: it goes to the server from the first page, not
  // after the cursor of the page someone happened to be on.
  it("sends a settled search to the server from the first page", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchToolPolicyPage.mockImplementation(async (filter, page) =>
      filter.search === "email"
        ? { items: [emailCustomer], endCursor: "cursor-e", hasNextPage: false, totalCount: 1 }
        : page.after === "cursor-3"
          ? secondPage
          : firstPage,
    );

    renderTab();

    const rules = await screen.findByRole("region", { name: "Tool rules" });
    await within(rules).findByText("email_customer");
    await userEvent.click(within(rules).getByRole("button", { name: "Go to next page" }));
    await within(rules).findByText("void_invoice");

    await userEvent.type(within(rules).getByRole("searchbox", { name: "Search tools" }), "email");

    await waitFor(() =>
      expect(fetchToolPolicyPage).toHaveBeenLastCalledWith(
        { ...ALL_TOOL_POLICIES, search: "email" },
        { first: 25, after: null, includeTotalCount: true },
      ),
    );
    await waitFor(() => expect(within(rules).queryByText("assign_move")).toBeNull());
    expect(within(rules).getByText("email_customer")).toBeTruthy();
  });

  it("says when no rule matches", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchToolPolicyPage.mockImplementation(async (filter) =>
      filter.search === ""
        ? firstPage
        : { items: [], endCursor: null, hasNextPage: false, totalCount: 0 },
    );

    renderTab();

    const rules = await screen.findByRole("region", { name: "Tool rules" });
    await within(rules).findByText("email_customer");
    await userEvent.type(within(rules).getByRole("searchbox", { name: "Search tools" }), "zzz");

    expect(await within(rules).findByText("No tool matches these filters.")).toBeTruthy();
  });

  it("reads an agent only once it is picked, and pages its tools", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchToolPolicyPage.mockResolvedValue(firstPage);
    const held = Array.from({ length: 30 }, (_, index) =>
      tool(`tool_${String(index).padStart(2, "0")}`, { answer: "NEEDS_APPROVAL" }),
    );
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
            { egress: ["ExternalRecipient"], leavesOrganization: true },
          ),
          ...held,
        ],
        reach: {
          accessMode: "Everyone",
          roles: [],
          warnings: [{ kind: "OpenWithSensitiveTools", tools: ["email_customer"] }],
        },
      }),
    ]);

    renderTab();

    const byAgent = await screen.findByRole("region", { name: "By agent" });
    await userEvent.click(within(byAgent).getByRole("button", { name: /add an agent/i }));
    await userEvent.click(await screen.findByRole("button", { name: "Pick Customer desk" }));

    const matrix = await within(byAgent).findByRole("region", { name: "Customer desk" });
    expect(fetchAgentSafety).toHaveBeenCalledTimes(1);
    expect(fetchAgentSafety.mock.calls[0]?.[0]).toEqual(["agdef_1"]);
    expect(pickerProps.mock.calls.at(-1)?.[0].source).toBe("grantable");

    expect(within(matrix).getByText("Everyone who can use the assistant")).toBeTruthy();
    expect(within(matrix).getByText(/leave the organization: Email customer\./)).toBeTruthy();
    expect(within(matrix).getByText("Read outside text")).toBeTruthy();
    expect(within(matrix).getAllByText("Runs on its own")).toHaveLength(2);

    const rows = () => within(matrix).getAllByRole("button", { name: /^Show details for / });
    expect(rows()).toHaveLength(25);
    expect(within(matrix).getByText("assign_move")).toBeTruthy();
    expect(within(matrix).queryByText("tool_29")).toBeNull();

    await userEvent.click(within(matrix).getByRole("button", { name: "Go to next page" }));
    expect(rows()).toHaveLength(7);
    expect(within(matrix).getByText("tool_29")).toBeTruthy();

    await userEvent.click(
      within(matrix).getByRole("button", { name: "Remove Customer desk from the comparison" }),
    );
    await waitFor(() => expect(within(byAgent).queryByRole("region")).toBeNull());
  });

  it("says so when the figures cannot be loaded", async () => {
    fetchAgentSafetySummary.mockRejectedValue(new Error("offline"));
    fetchToolPolicyPage.mockResolvedValue(firstPage);

    renderTab();

    expect(
      await screen.findByText(
        "What agents can do without a person could not be loaded. Try again shortly.",
      ),
    ).toBeTruthy();
  });

  it("offers to try again when the rules cannot be loaded", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchToolPolicyPage.mockRejectedValueOnce(new Error("offline")).mockResolvedValue(firstPage);

    renderTab();

    const rules = await screen.findByRole("region", { name: "Tool rules" });
    expect(await within(rules).findByText("Tool rules could not be loaded.")).toBeTruthy();

    await userEvent.click(within(rules).getByRole("button", { name: "Try again" }));
    expect(await within(rules).findByText("email_customer")).toBeTruthy();
  });
});
