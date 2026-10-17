import { assistant } from "@/lib/queries/assistant";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeleteConversationCard } from "../delete-conversation-card";
import { forgetThreadSchedules } from "../thread-schedules";
import { useDeleteConversation } from "../use-delete-conversation";

/**
 * The requests scheduled in each conversation, as GET
 * /assistant/threads/:id/schedules/ lists them; a conversation named in
 * `failing` cannot be read.
 */
const server = vi.hoisted(() => ({
  byThread: new Map<string, unknown[]>(),
  failing: new Set<string>(),
}));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return {
    ...actual,
    api: {
      ...actual.api,
      get: vi.fn(async (url: string) => {
        const threadId = /^\/assistant\/threads\/([^/]+)\/schedules\//.exec(url)?.[1] ?? "";
        if (server.failing.has(threadId)) {
          throw new Error("unavailable");
        }
        const items = server.byThread.get(threadId) ?? [];
        return { items, total: items.length };
      }),
    },
  };
});

function scheduled(id: string, threadId: string, prompt: string, cronExpression: string) {
  return {
    id,
    threadId,
    userId: "usr_1",
    prompt,
    cadence: "stored",
    cronExpression,
    timezone: "America/Chicago",
    enabled: true,
    lastRunAt: null,
    nextRunAt: null,
    lastTurnId: null,
    createdAt: 1_790_000_000,
    threadTitle: "",
  };
}

function thread(id: string, title: string): AssistantThread {
  return { id, title } as AssistantThread;
}

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function withClient(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

afterEach(() => {
  server.byThread = new Map();
  server.failing = new Set();
  vi.clearAllMocks();
});

/*
Deleting a conversation asks first only when something else goes with it:
the requests scheduled in it. Any other conversation is deleted at once.
*/
describe("useDeleteConversation", () => {
  it("deletes a conversation with no schedules at once", async () => {
    const onDelete = vi.fn();
    const { result } = renderHook(() => useDeleteConversation(onDelete), {
      wrapper: withClient(newClient()),
    });

    act(() => result.current.ask(thread("athr_quiet", "Late loads")));

    await waitFor(() => expect(onDelete).toHaveBeenCalledTimes(1));
    expect(onDelete).toHaveBeenCalledWith(thread("athr_quiet", "Late loads"));
    expect(result.current.asking).toBeNull();
  });

  it("asks before deleting a conversation with schedules, and deletes once answered", async () => {
    server.byThread.set("athr_ar", [
      scheduled("csch_1", "athr_ar", "Send me invoices past 60 days", "0 8 * * 1"),
    ]);
    const onDelete = vi.fn();
    const subject = thread("athr_ar", "AR follow-ups");
    const { result } = renderHook(() => useDeleteConversation(onDelete), {
      wrapper: withClient(newClient()),
    });

    act(() => result.current.ask(subject));

    await waitFor(() => expect(result.current.asking).toBe(subject));
    expect(onDelete).not.toHaveBeenCalled();

    act(() => result.current.confirm(subject));
    expect(onDelete).toHaveBeenCalledWith(subject);
    expect(result.current.asking).toBeNull();
  });

  it("asks when the schedules could not be counted, rather than lose them unasked", async () => {
    server.failing.add("athr_ar");
    const onDelete = vi.fn();
    const subject = thread("athr_ar", "AR follow-ups");
    const { result } = renderHook(() => useDeleteConversation(onDelete), {
      wrapper: withClient(newClient()),
    });

    act(() => result.current.ask(subject));

    await waitFor(() => expect(result.current.asking).toBe(subject));
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("deletes once for a second press made while the first is counting", async () => {
    const onDelete = vi.fn();
    const subject = thread("athr_quiet", "Late loads");
    const { result } = renderHook(() => useDeleteConversation(onDelete), {
      wrapper: withClient(newClient()),
    });

    act(() => {
      result.current.ask(subject);
      result.current.ask(subject);
    });

    await waitFor(() => expect(onDelete).toHaveBeenCalledTimes(1));
    await act(async () => {});
    expect(onDelete).toHaveBeenCalledTimes(1);
  });

  it("puts the question away on cancel", async () => {
    server.byThread.set("athr_ar", [scheduled("csch_1", "athr_ar", "Unbilled", "30 7 * * 1-5")]);
    const onDelete = vi.fn();
    const subject = thread("athr_ar", "AR follow-ups");
    const { result } = renderHook(() => useDeleteConversation(onDelete), {
      wrapper: withClient(newClient()),
    });

    act(() => result.current.ask(subject));
    await waitFor(() => expect(result.current.asking).toBe(subject));
    act(() => result.current.cancel());

    expect(result.current.asking).toBeNull();
    expect(onDelete).not.toHaveBeenCalled();
  });
});

/*
The question names the conversation and lists each request that goes with
it by what it asks and how often.
*/
describe("DeleteConversationCard", () => {
  it("names the conversation and lists each schedule that goes with it", async () => {
    server.byThread.set("athr_ar", [
      scheduled("csch_1", "athr_ar", "Send me invoices past 60 days", "0 8 * * 1"),
      scheduled("csch_2", "athr_ar", "Unbilled shipments", "30 7 * * 1-5"),
    ]);
    render(
      <DeleteConversationCard
        thread={thread("athr_ar", "AR follow-ups")}
        onConfirm={vi.fn()}
        onClose={vi.fn()}
      />,
      { wrapper: withClient(newClient()) },
    );

    const card = within(screen.getByRole("alertdialog"));
    expect(card.getByText("Delete “AR follow-ups”?")).toBeVisible();
    expect(await card.findByText("These schedules go with it:")).toBeVisible();
    const rows = card.getAllByRole("listitem").map((row) => row.textContent);
    expect(rows).toEqual([
      "Send me invoices past 60 daysEvery Monday · 8:00 AM",
      "Unbilled shipmentsEvery weekday · 7:30 AM",
    ]);
  });

  it("says the schedules go with it when they could not be listed", async () => {
    server.failing.add("athr_ar");
    render(
      <DeleteConversationCard
        thread={thread("athr_ar", "")}
        onConfirm={vi.fn()}
        onClose={vi.fn()}
      />,
      { wrapper: withClient(newClient()) },
    );

    const card = within(screen.getByRole("alertdialog"));
    expect(card.getByText("Delete “Untitled conversation”?")).toBeVisible();
    expect(await card.findByText("Its scheduled requests go with it.")).toBeVisible();
    expect(card.queryByRole("list")).toBeNull();
  });

  it("deletes on Delete and closes on Cancel", async () => {
    const onConfirm = vi.fn();
    const onClose = vi.fn();
    const subject = thread("athr_ar", "AR follow-ups");
    render(<DeleteConversationCard thread={subject} onConfirm={onConfirm} onClose={onClose} />, {
      wrapper: withClient(newClient()),
    });

    const card = within(screen.getByRole("alertdialog"));
    fireEvent.click(card.getByRole("button", { name: "Delete" }));
    expect(onConfirm).toHaveBeenCalledWith(subject);

    fireEvent.click(card.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
  });

  it("cannot be confirmed twice while the delete is on its way", () => {
    render(
      <DeleteConversationCard
        thread={thread("athr_ar", "AR follow-ups")}
        pending
        onConfirm={vi.fn()}
        onClose={vi.fn()}
      />,
      { wrapper: withClient(newClient()) },
    );

    expect(
      within(screen.getByRole("alertdialog")).getByRole("button", { name: "Delete" }),
    ).toBeDisabled();
  });
});

describe("forgetThreadSchedules", () => {
  it("drops the conversation's schedule cards and reads the Scheduled page again", () => {
    const client = newClient();
    const threadKey = assistant.schedules("athr_ar").queryKey;
    const otherKey = assistant.schedules("athr_other").queryKey;
    client.setQueryData(threadKey, { items: [], total: 2 });
    client.setQueryData(otherKey, { items: [], total: 1 });
    client.setQueryData(assistant.allSchedules().queryKey, { pages: [], pageParams: [] });

    forgetThreadSchedules(client, "athr_ar");

    expect(client.getQueryData(threadKey)).toBeUndefined();
    expect(client.getQueryData(otherKey)).toEqual({ items: [], total: 1 });
    expect(client.getQueryState(assistant.allSchedules().queryKey)?.isInvalidated).toBe(true);
  });
});
