import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantHistory } from "../assistant-history";
import { AssistantSidebar } from "../assistant-sidebar";

vi.mock("../use-active-turns", () => ({
  useLiveThreadIds: () => new Set<string>(),
}));

afterEach(cleanup);

const now = Math.floor(Date.now() / 1000);

function thread(
  id: string,
  title: string,
  overrides: Partial<AssistantThread> = {},
): AssistantThread {
  return {
    id,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agdef_billing",
    preferredProviderId: "",
    origin: "Panel",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title,
    status: "Active",
    lastMessageAt: now - 3 * 60 * 60,
    version: 0,
    createdAt: now - 3 * 60 * 60,
    updatedAt: now - 3 * 60 * 60,
    ...overrides,
  };
}

const agentsById = new Map<string, AgentChoice>([
  ["agdef_billing", { id: "agdef_billing", name: "Billing exceptions" } as AgentChoice],
  ["agdef_dispatch", { id: "agdef_dispatch", name: "Dispatch desk" } as AgentChoice],
]);

const threads = [
  thread("athr_stuck", "Which loads are stuck?"),
  thread("athr_pay", "Payments due this week", {
    agentDefinitionId: "agdef_dispatch",
    attention: { pendingDecisions: 1, lastTurnFailed: false, unread: false },
  }),
];

describe("AssistantHistory", () => {
  it("lists what waits on the person first, and names each agent and how long ago", () => {
    render(
      <AssistantHistory
        threads={threads}
        agentsById={agentsById}
        activeThreadId={null}
        onOpen={vi.fn()}
        onNew={vi.fn()}
      />,
    );

    const headings = screen
      .getAllByText(/^(Waiting on you|Today)$/)
      .map((node) => node.textContent);
    expect(headings).toEqual(["Waiting on you", "Today"]);
    const waiting = screen.getByRole("button", { name: /Payments due this week/ });
    expect(within(waiting).getByText("Dispatch desk · 3h ago")).toBeInTheDocument();
    expect(within(waiting).getByTitle("Waiting on your approval")).toBeInTheDocument();
  });

  it("searches by title and by agent, and says when nothing matches", () => {
    render(
      <AssistantHistory
        threads={threads}
        agentsById={agentsById}
        activeThreadId={null}
        onOpen={vi.fn()}
        onNew={vi.fn()}
      />,
    );
    const search = screen.getByRole("textbox", { name: "Search conversations" });

    fireEvent.change(search, { target: { value: "dispatch" } });
    expect(screen.getByRole("button", { name: /Payments due this week/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Which loads are stuck/ })).toBeNull();

    fireEvent.change(search, { target: { value: "invoices" } });
    expect(screen.getByText("No conversations match “invoices”")).toBeInTheDocument();
  });

  it("opens the one chosen and starts a new one", () => {
    const onOpen = vi.fn();
    const onNew = vi.fn();
    render(
      <AssistantHistory
        threads={threads}
        agentsById={agentsById}
        activeThreadId="athr_stuck"
        onOpen={onOpen}
        onNew={onNew}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /Which loads are stuck/ }));
    fireEvent.click(screen.getByRole("button", { name: "New conversation" }));

    expect(onOpen).toHaveBeenCalledWith(threads[0]);
    expect(onNew).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: /Which loads are stuck/ })).toHaveAttribute(
      "aria-current",
      "true",
    );
  });
});

describe("AssistantSidebar", () => {
  function renderSidebar(searchSignal = 0, activeThreadId: string | null = "athr_stuck") {
    const onOpen = vi.fn();
    const view = render(
      <AssistantSidebar
        threads={threads}
        agentsById={agentsById}
        activeThreadId={activeThreadId}
        searchSignal={searchSignal}
        onOpen={onOpen}
        onNew={vi.fn()}
      />,
    );
    return { ...view, onOpen };
  }

  it("draws the Desk rail's rows, with the open one current and a warm dot on what waits", () => {
    renderSidebar();

    const open = screen.getByRole("button", { name: /Which loads are stuck/ });
    expect(open).toHaveClass("dk-sb-i", "dk-on");
    expect(open).toHaveAttribute("aria-current", "true");
    expect(open).toHaveTextContent("3h");
    const waiting = screen.getByRole("button", { name: /Payments due this week/ });
    expect(within(waiting).getByLabelText("Waiting on your approval")).toHaveClass("as-wd");
    expect(screen.getByText("Waiting on you")).toHaveClass("dk-sb-gh");
  });

  it("turns the search row into a field, and Esc clears and closes it", () => {
    renderSidebar();

    fireEvent.click(screen.getByRole("button", { name: /Search/ }));
    const field = screen.getByRole("textbox", { name: "Search conversations" });
    fireEvent.change(field, { target: { value: "payments" } });
    expect(screen.queryByRole("button", { name: /Which loads are stuck/ })).toBeNull();

    fireEvent.keyDown(field, { key: "Escape" });
    expect(screen.queryByRole("textbox", { name: "Search conversations" })).toBeNull();
    expect(screen.getByRole("button", { name: /Which loads are stuck/ })).toBeInTheDocument();
  });

  it("opens the search when ⌘K asks for it", () => {
    const { rerender } = renderSidebar(0);
    expect(screen.queryByRole("textbox", { name: "Search conversations" })).toBeNull();

    rerender(
      <AssistantSidebar
        threads={threads}
        agentsById={agentsById}
        activeThreadId="athr_stuck"
        searchSignal={1}
        onOpen={vi.fn()}
        onNew={vi.fn()}
      />,
    );

    expect(screen.getByRole("textbox", { name: "Search conversations" })).toHaveFocus();
  });

  it("says it is read-only until the person approves", () => {
    renderSidebar();

    expect(screen.getByText("Read-only until you approve")).toBeInTheDocument();
  });
});
