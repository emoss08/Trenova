import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AgentRunTranscriptMessage } from "@/lib/graphql/agent-activity-tables";
import type { DeskConversationRow, OpenedDeskConversation } from "@/lib/graphql/desk-transcripts";
import { ConversationSheet } from "../conversation-sheet";

const mocks = vi.hoisted(() => ({ open: vi.fn(), download: vi.fn() }));

vi.mock("@/lib/graphql/desk-transcripts", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/desk-transcripts")>()),
  openDeskConversation: mocks.open,
}));

vi.mock("@/lib/desk-transcripts", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/desk-transcripts")>()),
  downloadDeskConversation: mocks.download,
}));

afterEach(cleanup);

const ROW: DeskConversationRow = {
  id: "athr_01",
  ownerName: "Dana Reyes",
  agentName: "Billing exceptions",
  title: "Why is INV-20419 short-paid",
  origin: "Desk",
  archived: false,
  messageCount: 4,
  startedAt: 1_790_000_000,
  lastMessageAt: 1_790_000_060,
} as DeskConversationRow;

/** One message as AgentRunTranscriptFields returns it, every field present. */
function message(fields: Partial<AgentRunTranscriptMessage>): AgentRunTranscriptMessage {
  return {
    role: "Assistant",
    kind: "Message",
    content: "",
    reasoning: "",
    toolCalls: [],
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    toolVerdict: "",
    toolSummary: "",
    agentDefinitionId: "",
    delegateCallId: "",
    omitted: false,
    createdAt: 0,
    ...fields,
  };
}

const MESSAGES: AgentRunTranscriptMessage[] = [
  message({ role: "User", content: "Why was it short-paid?", createdAt: 1_790_000_000 }),
  message({
    content: "I'll pull the invoice and the remittance.",
    toolCalls: [{ id: "call_1", name: "get_invoice", arguments: { number: "INV-20419" } }],
    createdAt: 1_790_000_010,
  }),
  message({
    role: "Tool",
    toolCallId: "call_1",
    toolName: "get_invoice",
    content: '{"balance":"240.00"}',
    toolSummary: "INV-20419",
    createdAt: 1_790_000_020,
  }),
  message({ content: "The customer deducted detention.", createdAt: 1_790_000_060 }),
];

function opened(
  messages: AgentRunTranscriptMessage[],
  omitted: { count: number; at: number } = { count: 0, at: 0 },
  row: DeskConversationRow = ROW,
): OpenedDeskConversation {
  return {
    conversation: row,
    transcript: { messages, omittedMessages: omitted.count, omittedAt: omitted.at },
  };
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function transportError(status: number): GraphQLRequestError {
  return new GraphQLRequestError({
    kind: "transport",
    message: `HTTP ${status}`,
    status,
    graphQLErrors: [],
  });
}

describe("ConversationSheet", () => {
  beforeEach(() => {
    mocks.open.mockReset();
    mocks.download.mockReset();
    mocks.open.mockResolvedValue(opened(MESSAGES));
  });

  it("reads another person's conversation with nothing to compose, decide or change", async () => {
    render(<ConversationSheet row={ROW} canExport={false} onClose={() => {}} />, { wrapper });

    expect(await screen.findByText("The customer deducted detention.")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Why was it short-paid?" })).toBeTruthy();
    expect(
      screen.getByText(
        "Opening Dana Reyes's conversation was recorded in the audit log. Nothing here changes it for them.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("textbox")).toBeNull();
    expect(
      screen.queryByRole("button", {
        name: /approve|reject|decline|send|rename|pin|snooze|delete|undo|ask again|try again|start a chapter|ask another agent|rephrase|continue/i,
      }),
    ).toBeNull();
    expect(screen.queryByRole("button", { name: "Download" })).toBeNull();
    expect(mocks.open).toHaveBeenCalledTimes(1);
    expect(mocks.open.mock.calls[0]?.[0]).toBe("athr_01");
  });

  it("folds the agent's notes and tool calls behind one working line and shows its answer", async () => {
    render(<ConversationSheet row={ROW} canExport={false} onClose={() => {}} />, { wrapper });

    expect(await screen.findByText("The customer deducted detention.")).toBeTruthy();
    expect(screen.queryByText("I'll pull the invoice and the remittance.")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Worked through 1 step" }));

    expect(await screen.findByText("I'll pull the invoice and the remittance.")).toBeTruthy();
  });

  it("says who and when in one line under the title", async () => {
    render(<ConversationSheet row={ROW} canExport={false} onClose={() => {}} />, { wrapper });
    await screen.findByText("The customer deducted detention.");

    expect(screen.getByText("Why is INV-20419 short-paid")).toBeTruthy();
    expect(screen.getByText("Dana Reyes with Billing exceptions")).toBeTruthy();
    expect(screen.getByText("Desk")).toBeTruthy();
    expect(screen.getByText("4 messages")).toBeTruthy();
    expect(screen.getByText(`Started ${formatUnixDateTimeMedium(1_790_000_000)}`)).toBeTruthy();
    expect(
      screen.getByText(`Last message ${formatUnixDateTimeMedium(1_790_000_060)}`),
    ).toBeTruthy();
  });

  it("counts a single message in the singular and leaves out a last message it does not have", async () => {
    const single = { ...ROW, messageCount: 1, lastMessageAt: null } as DeskConversationRow;
    mocks.open.mockResolvedValue(opened(MESSAGES.slice(0, 1), undefined, single));
    render(<ConversationSheet row={single} canExport={false} onClose={() => {}} />, { wrapper });

    expect(await screen.findByRole("heading", { name: "Why was it short-paid?" })).toBeTruthy();
    expect(screen.getByText("1 message")).toBeTruthy();
    expect(screen.queryByText(/^Last message/)).toBeNull();
  });

  it("offers Download beside the title to a holder of Export and downloads that conversation", async () => {
    mocks.download.mockResolvedValue(undefined);
    render(<ConversationSheet row={ROW} canExport onClose={() => {}} />, { wrapper });

    const download = await screen.findByRole("button", { name: "Download" });
    download.click();

    await waitFor(() => expect(mocks.download).toHaveBeenCalledWith("athr_01"));
  });

  it("draws the conversation's own layout as busy while it opens", () => {
    mocks.open.mockReturnValue(new Promise(() => {}));
    render(<ConversationSheet row={ROW} canExport onClose={() => {}} />, { wrapper });

    const busy = screen.getByLabelText("Opening the conversation");
    expect(busy.getAttribute("aria-busy")).toBe("true");
    expect(screen.queryByRole("button", { name: "Download" })).toBeNull();
  });

  it("says where messages were left out of the middle and where a turn was too large to show", async () => {
    const clipped = MESSAGES.map((saved, index) =>
      index === 2 ? { ...saved, omitted: true } : saved,
    );
    mocks.open.mockResolvedValue(opened(clipped, { count: 3, at: 4 }));
    render(<ConversationSheet row={ROW} canExport={false} onClose={() => {}} />, { wrapper });

    expect(
      await screen.findByText(
        "3 messages from the middle of this conversation are left out here; download the transcript to read all of them",
      ),
    ).toBeTruthy();
    expect(
      screen.getByText(
        "Part of this turn is too large to show here; download the transcript to read it whole.",
      ),
    ).toBeTruthy();
  });

  it("says when nothing was said", async () => {
    mocks.open.mockResolvedValue(opened([]));
    render(<ConversationSheet row={ROW} canExport={false} onClose={() => {}} />, { wrapper });

    expect(await screen.findByText("Nothing was said in this conversation.")).toBeTruthy();
  });

  it("says when a conversation could not be opened and lets the reader try again", async () => {
    mocks.open.mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce(opened(MESSAGES));
    render(<ConversationSheet row={ROW} canExport onClose={() => {}} />, { wrapper });

    expect(
      await screen.findByText("This conversation could not be opened. Try again shortly."),
    ).toBeTruthy();
    screen.getByRole("button", { name: "Try again" }).click();

    expect(await screen.findByText("The customer deducted detention.")).toBeTruthy();
  });

  it("says the reader may no longer read it, with nothing to try again", async () => {
    mocks.open.mockRejectedValue(transportError(403));
    render(<ConversationSheet row={ROW} canExport onClose={() => {}} />, { wrapper });

    expect(
      await screen.findByText(
        "You no longer have permission to read other people's Desk conversations.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Try again" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Download" })).toBeNull();
  });

  it("says a conversation that is gone no longer exists", async () => {
    mocks.open.mockRejectedValue(transportError(404));
    render(<ConversationSheet row={ROW} canExport onClose={() => {}} />, { wrapper });

    expect(await screen.findByText("This conversation no longer exists.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Try again" })).toBeNull();
  });
});
