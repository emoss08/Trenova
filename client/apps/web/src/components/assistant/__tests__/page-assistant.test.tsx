import type { PageThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PageAssistant } from "../page-assistant";

const mocks = vi.hoisted(() => ({
  canCreate: true,
  getThread: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canCreate, isLoading: false }),
}));

vi.mock("@/services/api", () => ({
  apiService: { assistantService: { getThread: mocks.getThread, listArtifacts: vi.fn() } },
}));

vi.mock("../message-thread", () => ({
  MessageThread: ({ thread }: { thread: { id: string } }) => (
    <div data-testid="thread">{thread.id}</div>
  ),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.canCreate = true;
});

/* services.PageThread as the thread routes return it. */
function pageThread(overrides: Partial<PageThread["thread"]> = {}): PageThread {
  return {
    thread: {
      id: "athr_1",
      businessUnitId: "bu_1",
      organizationId: "org_1",
      userId: "usr_1",
      agentDefinitionId: "agdef_1",
      title: "Shipment import",
      status: "Active",
      lastMessageAt: 0,
      preferredProviderId: "",
      origin: "Import",
      pinned: false,
      subjectType: "Document",
      subjectId: "doc_1",
      canContinue: true,
      taintedAt: 10,
      version: 1,
      createdAt: 10,
      updatedAt: 10,
      ...overrides,
    },
    agent: {
      id: "agdef_1",
      name: "Shipment import assistant",
      description: "",
      template: "ImportAssistant",
      icon: "",
      accent: "",
      systemKey: "import_assistant",
      toolNames: [],
      starters: [],
    },
  };
}

function renderAssistant(open: () => Promise<PageThread>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PageAssistant
        conversationKey={["import", "doc_1"]}
        open={open}
        page={{ surface: "shipment_import", readDraft: () => null, onDraftEdit: vi.fn() }}
      />
    </QueryClientProvider>,
  );
}

describe("PageAssistant", () => {
  it("explains that assistant access is needed, without asking the server", () => {
    mocks.canCreate = false;
    const open = vi.fn();

    renderAssistant(open);

    expect(screen.getByText("The assistant is not available to you")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Using the assistant needs permission to start assistant conversations. An administrator can add it to one of your roles.",
      ),
    ).toBeInTheDocument();
    expect(open).not.toHaveBeenCalled();
    expect(screen.queryByTestId("thread")).toBeNull();
  });

  it("passes on the server's refusal when the person may not use the page's agent", async () => {
    const open = vi.fn().mockRejectedValue(
      new ApiRequestError(403, {
        type: "authorization-error",
        title: "Forbidden",
        status: 403,
        detail:
          "You do not have access to Shipment import assistant. An administrator can give one of your roles access to it.",
      }),
    );

    renderAssistant(open);

    expect(
      await screen.findByText(
        "You do not have access to Shipment import assistant. An administrator can give one of your roles access to it.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("thread")).toBeNull();
  });

  it("opens the conversation and marks it as having read the document", async () => {
    const opened = pageThread();
    mocks.getThread.mockResolvedValue(opened.thread);

    renderAssistant(vi.fn().mockResolvedValue(opened));

    expect(await screen.findByTestId("thread")).toHaveTextContent("athr_1");
    expect(screen.getByText("Shipment import assistant")).toBeInTheDocument();
    expect(screen.getByText("Read outside content")).toBeInTheDocument();
  });

  it("offers a new conversation once the server closed this one", async () => {
    const closed = pageThread({ status: "Archived" });
    const fresh = pageThread({ id: "athr_2", status: "Active", taintedAt: null });
    mocks.getThread.mockImplementation(async (id: string) =>
      id === "athr_1" ? closed.thread : fresh.thread,
    );
    const open = vi.fn().mockResolvedValueOnce(closed).mockResolvedValueOnce(fresh);

    renderAssistant(open);

    await userEvent.click(await screen.findByRole("button", { name: "Start a new conversation" }));

    await waitFor(() => expect(screen.getByTestId("thread")).toHaveTextContent("athr_2"));
    expect(open).toHaveBeenCalledTimes(2);
    expect(screen.queryByText("Read outside content")).toBeNull();
  });
});
