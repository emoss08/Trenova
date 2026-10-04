import type { AgentReflection } from "@/lib/graphql/agent-reflections";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AgentReflections } from "../agent-reflections";

const api = vi.hoisted(() => ({ fetchRecentAgentReflections: vi.fn() }));

vi.mock("@/lib/graphql/agent-reflections", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-reflections")>()),
  ...api,
}));

function reflection(overrides: Partial<AgentReflection> & { id: string }): AgentReflection {
  return {
    agentDefinitionId: "agdef_1",
    subjectType: "Thread",
    threadId: "athr_1",
    runId: null,
    userId: "usr_1",
    fromSequence: 1,
    throughSequence: 9,
    status: "Completed",
    skipReason: null,
    signals: [],
    changes: [],
    notes: "",
    tainted: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    errorMessage: "",
    finishedAt: 1_790_000_100,
    createdAt: 1_790_000_000,
    ...overrides,
  } as AgentReflection;
}

function renderPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <AgentReflections />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.fetchRecentAgentReflections.mockReset();
});

describe("AgentReflections", () => {
  it("is not drawn until an agent has looked back over something", async () => {
    api.fetchRecentAgentReflections.mockResolvedValue([]);

    const { container } = renderPanel();

    await vi.waitFor(() => expect(api.fetchRecentAgentReflections).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("shows what made each look back happen and what it kept, offered or refused", async () => {
    api.fetchRecentAgentReflections.mockResolvedValue([
      reflection({
        id: "arfl_1",
        signals: [{ kind: "ToolRecovered", count: 2, detail: "assign_move" }],
        changes: [
          {
            action: "Saved",
            memoryId: "amem_1",
            supersedesId: null,
            kind: "Procedure",
            scope: "Agent",
            content: "Read the move before assigning it.",
            reason: "",
          },
          {
            action: "Suggested",
            memoryId: "amem_2",
            supersedesId: null,
            kind: "Instruction",
            scope: "Organization",
            content: "Copy billing on rate confirmations.",
            reason: "",
          },
          {
            action: "Refused",
            memoryId: null,
            supersedesId: null,
            kind: "Fact",
            scope: "User",
            content: "Acme has a dock in Reno.",
            reason: "It names a record the work did not touch",
          },
        ],
        notes: "One lesson kept, one offered.",
      }),
    ]);

    renderPanel();

    expect(await screen.findByText("What agents learned")).toBeInTheDocument();
    const why = screen.getByRole("list", { name: "Why it looked" });
    expect(within(why).getByText(/A tool worked after failing · assign_move/)).toBeInTheDocument();
    const lessons = screen.getByRole("list", { name: "Lessons" });
    expect(within(lessons).getByText("Kept")).toBeInTheDocument();
    expect(within(lessons).getByText("Offered")).toBeInTheDocument();
    expect(within(lessons).getByText("Not kept")).toBeInTheDocument();
    expect(
      within(lessons).getByText("It names a record the work did not touch"),
    ).toBeInTheDocument();
    expect(screen.getByText("One lesson kept, one offered.")).toBeInTheDocument();
  });

  it("says why a look back could not finish, and when a run read outside content", async () => {
    api.fetchRecentAgentReflections.mockResolvedValue([
      reflection({
        id: "arfl_2",
        subjectType: "Run",
        threadId: null,
        runId: "arun_1",
        status: "Failed",
        tainted: true,
        errorMessage: "the provider timed out",
      }),
    ]);

    renderPanel();

    expect(
      await screen.findByText("The look back could not finish: the provider timed out"),
    ).toBeInTheDocument();
    expect(screen.getByText(/A background run/)).toBeInTheDocument();
    expect(screen.getByText(/read content written outside the organization/)).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Lessons" })).not.toBeInTheDocument();
  });
});
