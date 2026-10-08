import type { AgentMemorySuggestion } from "@/lib/graphql/agent-memories";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemorySuggestions } from "../memory-suggestions";

const api = vi.hoisted(() => ({
  fetchAgentMemorySuggestions: vi.fn(),
  approveAgentMemorySuggestion: vi.fn(),
  dismissAgentMemorySuggestion: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-memories")>()),
  ...api,
}));

function suggestion(
  overrides: Partial<AgentMemorySuggestion> & { id: string },
): AgentMemorySuggestion {
  return {
    kind: "Instruction",
    source: "Feedback",
    status: "Suggested",
    content: "Copy the AP inbox on Acme invoices.",
    version: 2,
    tainted: false,
    supersedes: null,
    evidence: {
      ratingCount: 4,
      distinctUsers: 3,
      feedbackIds: [],
      quotes: ["It forgot the AP inbox again"],
      lastRatedAt: 1_790_000_000,
      reason: "",
      signals: [],
    },
    createdAt: 1_790_000_000,
    ...overrides,
  } as AgentMemorySuggestion;
}

function renderSuggestions(canDecide = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemorySuggestions canDecide={canDecide} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.approveAgentMemorySuggestion.mockResolvedValue({});
  api.dismissAgentMemorySuggestion.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("MemorySuggestions", () => {
  it("is not drawn while nothing is suggested", async () => {
    api.fetchAgentMemorySuggestions.mockResolvedValue([]);
    const { container } = renderSuggestions();

    await vi.waitFor(() => expect(api.fetchAgentMemorySuggestions).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("approves a suggestion as written, at the version it was read at", async () => {
    api.fetchAgentMemorySuggestions.mockResolvedValue([suggestion({ id: "amem_1" })]);
    renderSuggestions();

    expect(await screen.findByText("Nova suggests")).toBeInTheDocument();
    expect(screen.getByText(/Drawn from 4 ratings by 3 people/)).toBeInTheDocument();
    expect(screen.getByText("It forgot the AP inbox again")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Approve" }));
    expect(api.approveAgentMemorySuggestion).toHaveBeenCalledWith("amem_1", {
      content: "Copy the AP inbox on Acme invoices.",
      version: 2,
    });
  });

  it("approves the edited words, trimmed, when edited first", async () => {
    api.fetchAgentMemorySuggestions.mockResolvedValue([suggestion({ id: "amem_1" })]);
    renderSuggestions();

    await userEvent.click(await screen.findByRole("button", { name: "Edit" }));
    const box = screen.getByRole("textbox", { name: "Memory" });
    await userEvent.clear(box);
    await userEvent.type(box, "  Always copy ap@acme.com on Acme invoices.  ");
    const approves = screen.getAllByRole("button", { name: "Approve" });
    await userEvent.click(approves[approves.length - 1]!);

    expect(api.approveAgentMemorySuggestion).toHaveBeenCalledWith("amem_1", {
      content: "Always copy ap@acme.com on Acme invoices.",
      version: 2,
    });
  });

  it("dismisses a suggestion", async () => {
    api.fetchAgentMemorySuggestions.mockResolvedValue([suggestion({ id: "amem_1" })]);
    renderSuggestions();

    await userEvent.click(await screen.findByRole("button", { name: "Dismiss" }));
    expect(api.dismissAgentMemorySuggestion).toHaveBeenCalledWith("amem_1", 2);
  });

  it("says a lesson was learned by an agent, and why, rather than drawn from ratings", async () => {
    api.fetchAgentMemorySuggestions.mockResolvedValue([
      suggestion({
        id: "amem_2",
        source: "Reflection",
        evidence: {
          ratingCount: null,
          distinctUsers: null,
          feedbackIds: [],
          quotes: [],
          lastRatedAt: 0,
          reason: "It took three tries to assign the move",
          signals: [],
        } as unknown as AgentMemorySuggestion["evidence"],
      }),
    ]);
    renderSuggestions();

    expect(
      await screen.findByText(/Learned by an agent looking back over its work/),
    ).toBeInTheDocument();
    expect(screen.getByText("It took three tries to assign the move")).toBeInTheDocument();
    expect(screen.queryByText(/Drawn from/)).not.toBeInTheDocument();
  });

  it("offers no decision to someone who cannot make one", async () => {
    api.fetchAgentMemorySuggestions.mockResolvedValue([suggestion({ id: "amem_1" })]);
    renderSuggestions(false);

    await screen.findByText("Nova suggests");
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
  });
});
