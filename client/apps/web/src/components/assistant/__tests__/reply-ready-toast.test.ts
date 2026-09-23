import { queries } from "@/lib/queries";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReplyReady, ViewingState } from "../reply-ready";
import {
  announceReplyReady,
  cachedConversationTitle,
  type AnnounceReplyReadyOptions,
} from "../reply-ready-toast";

type ToastOptions = {
  id?: string;
  description?: string;
  action?: { label: string; onClick: () => void };
};

const toastMock = vi.hoisted(() => {
  const neutral = vi.fn((_title: string, _options?: ToastOptions) => "toast");
  return Object.assign(neutral, {
    error: vi.fn((_title: string, _options?: ToastOptions) => "toast"),
    info: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
  });
});

vi.mock("sonner", () => ({ toast: toastMock }));

const markRead = vi.hoisted(() => vi.fn(async (_ids: string[]) => undefined));
const getThread = vi.hoisted(() => vi.fn<(id: string) => Promise<AssistantThread>>());

vi.mock("@/services/api", () => ({
  apiService: {
    notificationService: { markRead: (ids: string[]) => markRead(ids) },
    assistantService: { getThread: (id: string) => getThread(id) },
  },
}));

const t = (message: string | null | undefined) => message ?? "";

/*
The notice's data carries threadId, turnId, status and link; its own title and
message are generic, because notifications travel on a tenant-wide channel.
The conversation's name comes from the conversation.
*/
const reply: ReplyReady = {
  threadId: "athr_1",
  turnId: "atrn_1",
  status: "Completed",
  link: "/desk/t/athr_1",
};

function thread(overrides: Partial<AssistantThread> = {}): AssistantThread {
  return { id: "athr_1", title: "Where is SEED-SHP-001?", ...overrides } as AssistantThread;
}

const elsewhere: ViewingState = {
  pathname: "/shipment-management/shipments",
  panelOpen: false,
  panelThreadId: null,
};

let queryClient: QueryClient;

beforeEach(() => {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
});

afterEach(() => {
  vi.clearAllMocks();
  getThread.mockReset();
  queryClient.clear();
});

async function announce(overrides: Partial<AnnounceReplyReadyOptions> = {}) {
  const navigate = vi.fn();
  const outcome = await announceReplyReady({
    reply,
    notificationId: "ntf_1",
    queryClient,
    navigate,
    t,
    viewing: () => elsewhere,
    ...overrides,
  });

  return { outcome, navigate };
}

describe("announceReplyReady", () => {
  it("names the conversation from the list and offers a way to open it", async () => {
    queryClient.setQueryData(queries.assistant.threads().queryKey, {
      items: [thread()],
      total: 1,
    });

    const { outcome, navigate } = await announce();

    expect(outcome).toBe("shown");
    expect(getThread).not.toHaveBeenCalled();
    const [title, options] = toastMock.mock.calls[0];
    expect(title).toBe("Your reply is ready");
    expect(options?.description).toBe("Where is SEED-SHP-001?");
    expect(options?.action?.label).toBe("Open");

    options?.action?.onClick();
    expect(navigate).toHaveBeenCalledWith("/desk/t/athr_1");
    expect(markRead).toHaveBeenCalledWith(["ntf_1"]);
  });

  // A palette question nobody kept is not in the list; the live replies were
  // the one place holding its title, and they are read before being refreshed.
  it("names an unlisted conversation from the reply that was writing in it", async () => {
    queryClient.setQueryData(queries.assistant.activeTurns().queryKey, {
      items: [
        {
          turnId: "atrn_1",
          threadId: "athr_1",
          threadTitle: "What is overdue?",
          origin: "Person",
          startedAt: 1,
        },
      ],
    });

    await announce();

    expect(getThread).not.toHaveBeenCalled();
    expect(toastMock.mock.calls[0][1]?.description).toBe("What is overdue?");
  });

  it("reads the conversation when nothing in this tab holds it", async () => {
    getThread.mockResolvedValue(thread({ title: "Asked in another tab" }));

    await announce();

    expect(getThread).toHaveBeenCalledWith("athr_1");
    expect(toastMock.mock.calls[0][1]?.description).toBe("Asked in another tab");
  });

  it("still announces the reply when the conversation cannot be read", async () => {
    getThread.mockRejectedValue(new Error("offline"));

    const { outcome } = await announce();

    expect(outcome).toBe("shown");
    expect(toastMock.mock.calls[0][1]?.description).toBeUndefined();
  });

  it("calls an untitled conversation untitled", async () => {
    getThread.mockResolvedValue(thread({ title: "" }));

    await announce();

    expect(toastMock.mock.calls[0][1]?.description).toBe("Untitled conversation");
  });

  it("keeps a refused reply neutral", async () => {
    getThread.mockResolvedValue(thread());

    await announce({ reply: { ...reply, status: "Refused" } });

    expect(toastMock).toHaveBeenCalledTimes(1);
    expect(toastMock.mock.calls[0][0]).toBe("The assistant declined to answer");
    expect(toastMock.error).not.toHaveBeenCalled();
  });

  it("gives a failed reply the danger tone", async () => {
    getThread.mockResolvedValue(thread());

    await announce({ reply: { ...reply, status: "Failed" } });

    expect(toastMock).not.toHaveBeenCalled();
    expect(toastMock.error).toHaveBeenCalledTimes(1);
    expect(toastMock.error.mock.calls[0][0]).toBe("Your reply could not finish");
  });

  it("shows one notice per reply however often it is delivered", async () => {
    getThread.mockResolvedValue(thread());

    await announce();
    await announce();

    const ids = toastMock.mock.calls.map(([, options]) => options?.id);
    expect(new Set(ids).size).toBe(1);
  });

  it("stays quiet about the conversation the person is looking at, and marks it read", async () => {
    getThread.mockResolvedValue(thread());

    const { outcome } = await announce({
      viewing: () => ({ pathname: "/desk/t/athr_1", panelOpen: false, panelThreadId: null }),
    });

    expect(outcome).toBe("suppressed");
    expect(toastMock).not.toHaveBeenCalled();
    expect(toastMock.error).not.toHaveBeenCalled();
    expect(markRead).toHaveBeenCalledWith(["ntf_1"]);
  });

  it("stays quiet when the corner panel is open on the conversation", async () => {
    getThread.mockResolvedValue(thread());

    const { outcome } = await announce({
      viewing: () => ({ pathname: "/", panelOpen: true, panelThreadId: "athr_1" }),
    });

    expect(outcome).toBe("suppressed");
  });

  it("still speaks when the person is looking at a different conversation", async () => {
    getThread.mockResolvedValue(thread());

    const { outcome } = await announce({
      viewing: () => ({ pathname: "/desk/t/athr_2", panelOpen: false, panelThreadId: null }),
    });

    expect(outcome).toBe("shown");
  });

  it("clears the writing markers and refreshes the conversation either way", async () => {
    getThread.mockResolvedValue(thread());
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    await announce({
      viewing: () => ({ pathname: "/desk/t/athr_1", panelOpen: false, panelThreadId: null }),
    });

    const keys = invalidate.mock.calls.map(([filters]) => JSON.stringify(filters?.queryKey));
    expect(keys).toContain(JSON.stringify(queries.assistant.activeTurns().queryKey));
    expect(keys).toContain(JSON.stringify(queries.assistant.messages("athr_1").queryKey));
  });

  it("opens without marking anything when the notice has no id", async () => {
    getThread.mockResolvedValue(thread());

    const { navigate } = await announce({ notificationId: null });
    toastMock.mock.calls[0][1]?.action?.onClick();

    expect(navigate).toHaveBeenCalledWith("/desk/t/athr_1");
    expect(markRead).not.toHaveBeenCalled();
  });
});

describe("cachedConversationTitle", () => {
  it("is null when nothing in the tab knows the conversation", () => {
    expect(cachedConversationTitle(queryClient, "athr_1")).toBeNull();
  });

  it("prefers the list, then the conversation read on its own", () => {
    queryClient.setQueryData(queries.assistant.thread("athr_1").queryKey, thread({ title: "B" }));
    expect(cachedConversationTitle(queryClient, "athr_1")).toBe("B");

    queryClient.setQueryData(queries.assistant.threads().queryKey, {
      items: [thread({ title: "A" })],
      total: 1,
    });
    expect(cachedConversationTitle(queryClient, "athr_1")).toBe("A");
  });
});
