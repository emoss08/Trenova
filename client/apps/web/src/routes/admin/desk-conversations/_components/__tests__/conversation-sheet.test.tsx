import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
  messageCount: 2,
  startedAt: 1_790_000_000,
  lastMessageAt: 1_790_000_060,
} as DeskConversationRow;

const OPENED: OpenedDeskConversation = {
  conversation: ROW,
  transcript: {
    omittedMessages: 0,
    omittedAt: 0,
    messages: [
      {
        role: "User",
        kind: "Message",
        content: "Why was it short-paid?",
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
        createdAt: 1_790_000_000,
      },
      {
        role: "Assistant",
        kind: "Message",
        content: "The customer deducted detention.",
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
        createdAt: 1_790_000_060,
      },
    ],
  },
};

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("ConversationSheet", () => {
  beforeEach(() => {
    mocks.open.mockReset();
    mocks.download.mockReset();
    mocks.open.mockResolvedValue(OPENED);
  });

  it("reads another person's conversation with nothing to compose, decide or change", async () => {
    render(<ConversationSheet row={ROW} canExport={false} onClose={() => {}} />, { wrapper });

    expect(await screen.findByText("The customer deducted detention.")).toBeTruthy();
    expect(screen.getByText("Why was it short-paid?")).toBeTruthy();
    expect(screen.getByText("Dana Reyes's conversation with Billing exceptions")).toBeTruthy();
    expect(
      screen.getByText(
        "You are reading Dana Reyes's conversation as they saw it. Opening it was recorded in the audit log; nothing here changes it for them.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("textbox")).toBeNull();
    expect(
      screen.queryByRole("button", { name: /approve|decline|send|rename|pin|snooze|delete/i }),
    ).toBeNull();
    expect(screen.queryByRole("button", { name: "Download" })).toBeNull();
    expect(mocks.open).toHaveBeenCalledTimes(1);
    expect(mocks.open.mock.calls[0]?.[0]).toBe("athr_01");
  });

  it("offers Download to a holder of Export and downloads that conversation", async () => {
    mocks.download.mockResolvedValue(undefined);
    render(<ConversationSheet row={ROW} canExport onClose={() => {}} />, { wrapper });

    const download = await screen.findByRole("button", { name: "Download" });
    download.click();

    await waitFor(() => expect(mocks.download).toHaveBeenCalledWith("athr_01"));
  });

  it("says when a conversation could not be opened and lets the reader try again", async () => {
    mocks.open.mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce(OPENED);
    render(<ConversationSheet row={ROW} canExport onClose={() => {}} />, { wrapper });

    expect(
      await screen.findByText("This conversation could not be opened. Try again shortly."),
    ).toBeTruthy();
    screen.getByRole("button", { name: "Try again" }).click();

    expect(await screen.findByText("The customer deducted detention.")).toBeTruthy();
  });
});
