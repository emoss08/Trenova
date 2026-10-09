import type { ActiveTurn, StartedTurn } from "@/services/assistant";
import type { AssistantStreamEvent } from "@/types/assistant";
import type { AssistantMessage, SendMessageResult } from "@/types/assistant";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { queries } from "@/lib/queries";
import { useRealtimeStore } from "@/stores/realtime-store";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useAssistantTurn } from "../use-assistant-turn";

type StartTurn = (
  threadId: string,
  content: string,
  options: unknown,
  request: { signal?: AbortSignal },
) => Promise<StartedTurn>;

type Attach = (
  turnId: string,
  onEvent: (event: AssistantStreamEvent, cursor: string) => void,
  options: { cursor?: string; signal?: AbortSignal },
) => Promise<void>;

const activeTurn = vi.hoisted(() => vi.fn<(threadId: string) => Promise<ActiveTurn | null>>());
const startTurn = vi.hoisted(() => vi.fn<StartTurn>());
const attachTurn = vi.hoisted(() => vi.fn<Attach>());
const stopTurn = vi.hoisted(() => vi.fn(async (_turnId: string) => undefined));

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      activeTurn: (threadId: string) => activeTurn(threadId),
      startTurn: (...args: Parameters<StartTurn>) => startTurn(...args),
      attachTurn: (...args: Parameters<Attach>) => attachTurn(...args),
      stopTurn: (turnId: string) => stopTurn(turnId),
    },
  },
}));

const started: StartedTurn = {
  turnId: "atrn_1",
  threadId: "athr_1",
  streamUrl: "/api/v1/assistant/turns/atrn_1/stream/",
  status: "Running",
};

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function renderTurn(client = new QueryClient({ defaultOptions: { queries: { retry: false } } })) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useAssistantTurn("athr_1", () => null), { wrapper });
}

afterEach(() => {
  cleanup();
  useRealtimeStore.getState().setConnectionState("disconnected");
  activeTurn.mockReset();
  startTurn.mockReset();
  attachTurn.mockReset();
  stopTurn.mockClear();
});

/**
 * The composer is disabled from the click, not from the moment the turn
 * appears. Between the two the view asks whether the conversation is already
 * producing a reply, and a second Enter in that gap used to send the message
 * twice.
 */
describe("useAssistantTurn sending", () => {
  it("is busy from the click and refuses a second send before the first starts", async () => {
    const lookup = deferred<ActiveTurn | null>();
    activeTurn.mockReturnValue(lookup.promise);
    startTurn.mockReturnValue(new Promise(() => undefined));

    const { result } = renderTurn();

    act(() => {
      void result.current.send("Where is S1?");
      void result.current.send("Where is S1?");
    });
    expect(result.current.isActive).toBe(true);

    await act(async () => {
      lookup.resolve(null);
    });

    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));
    expect(activeTurn).toHaveBeenCalledTimes(1);
  });

  it("accepts a send again once the start fails", async () => {
    activeTurn.mockResolvedValue(null);
    startTurn.mockRejectedValueOnce(new Error("offline"));

    const { result } = renderTurn();

    await act(async () => {
      await result.current.send("Where is S1?");
    });
    expect(result.current.isActive).toBe(false);
    expect(result.current.turn?.status).toBe("error");

    startTurn.mockReturnValue(new Promise(() => undefined));
    act(() => {
      void result.current.send("Where is S1?");
    });

    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(2));
  });
});

/**
 * Before asking, the view checks whether the conversation is already producing
 * a reply. That check is a round trip, and the person's own question used to
 * stay off screen until it returned.
 */
describe("useAssistantTurn showing the question", () => {
  it("shows the question while the conversation is still being checked", async () => {
    const lookup = deferred<ActiveTurn | null>();
    activeTurn.mockReturnValue(lookup.promise);
    startTurn.mockReturnValue(new Promise(() => undefined));

    const { result } = renderTurn();

    act(() => {
      void result.current.send("Where is S1?");
    });

    expect(result.current.turn?.userContent).toBe("Where is S1?");
    expect(result.current.turn?.status).toBe("guarding");

    await act(async () => {
      lookup.resolve(null);
    });
    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));
  });

  /**
   * The live-turn list is kept current by the realtime connection. While it
   * is connected and the list it last delivered names no reply on this
   * conversation, asking the server again only delays the question.
   */
  it("skips the check when the live list, kept current, says the conversation is quiet", async () => {
    startTurn.mockReturnValue(new Promise(() => undefined));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(queries.assistant.activeTurns().queryKey, {
      items: [
        {
          turnId: "atrn_9",
          threadId: "athr_other",
          threadTitle: "",
          origin: "Person",
          startedAt: 1,
        },
      ],
    });
    useRealtimeStore.getState().setConnectionState("connected");

    const { result } = renderTurn(client);

    act(() => {
      void result.current.send("Where is S1?");
    });

    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));
    expect(activeTurn).not.toHaveBeenCalled();
  });

  it.each([
    ["the realtime connection is down", "disconnected", false, "athr_other"],
    ["the list is being refreshed", "connected", true, "athr_other"],
    ["the list names this conversation", "connected", false, "athr_1"],
  ] as const)("still checks when %s", async (_case, connection, invalidated, liveThread) => {
    activeTurn.mockResolvedValue(null);
    startTurn.mockReturnValue(new Promise(() => undefined));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(queries.assistant.activeTurns().queryKey, {
      items: [
        { turnId: "atrn_9", threadId: liveThread, threadTitle: "", origin: "Person", startedAt: 1 },
      ],
    });
    if (invalidated) {
      await client.invalidateQueries({
        queryKey: queries.assistant.activeTurns().queryKey,
        refetchType: "none",
      });
    }
    useRealtimeStore.getState().setConnectionState(connection);

    const { result } = renderTurn(client);

    act(() => {
      void result.current.send("Where is S1?");
    });

    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));
    expect(activeTurn).toHaveBeenCalledTimes(1);
  });
});

function savedMessage(sequence: number): AssistantMessage {
  return {
    id: `amsg_${sequence}`,
    threadId: "athr_1",
    kind: "Message",
    sequence,
    role: sequence % 2 === 0 ? "User" : "Assistant",
    content: `m${sequence}`,
    toolCalls: null,
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt: 0,
  } as AssistantMessage;
}

/**
 * A finished turn writes its saved rows into the cached history, so the reply
 * is already in the thread when the stream ends. The streaming copy used to
 * stay up until five unrelated lists had refetched as well — the reply shown
 * twice, and the composer held, for as long as the slowest of them took.
 */
describe("useAssistantTurn handing over to the saved thread", () => {
  it("clears the streaming turn once the saved rows are in, without waiting on refetches", async () => {
    activeTurn.mockResolvedValue(null);
    startTurn.mockResolvedValue(started);
    const result: SendMessageResult = {
      thread: {} as SendMessageResult["thread"],
      messages: [savedMessage(2), savedMessage(3)],
      reply: "m3",
      refused: false,
      proposals: null,
      proposalsUnrecorded: false,
      artifacts: null,
    };
    attachTurn.mockImplementation(async (_turnId, onEvent) => {
      onEvent({ event: "done", data: result } as AssistantStreamEvent, "1");
    });

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(queries.assistant.messages("athr_1").queryKey, {
      pages: [
        { results: [savedMessage(0), savedMessage(1)], hasMore: false, total: 2, limit: 400 },
      ],
      pageParams: [undefined],
    });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    const hung = () => new Promise<never>(() => undefined);
    const { result: hook } = renderHook(
      () => {
        useQuery({ queryKey: queries.assistant.activeTurns().queryKey, queryFn: hung });
        useQuery({ queryKey: queries.assistant.threads().queryKey, queryFn: hung });
        return useAssistantTurn("athr_1", () => null);
      },
      { wrapper },
    );

    await act(async () => {
      void hook.current.send("m2");
    });

    await waitFor(() => expect(hook.current.turn).toBeNull());
    const history = client.getQueryData<{ pages: { results: AssistantMessage[] }[] }>(
      queries.assistant.messages("athr_1").queryKey,
    );
    expect(history?.pages[0]?.results.map((message) => message.sequence)).toEqual([0, 1, 2, 3]);
  });
});

/**
 * Stop pressed before the worker has the question. Nothing used to be
 * cancelled: the request finished, the turn ran on, and it answered a question
 * the person had taken back.
 */
describe("useAssistantTurn stopping before the turn starts", () => {
  it("aborts the start request and stops the turn whose id arrives after Stop", async () => {
    activeTurn.mockResolvedValue(null);
    const request = deferred<StartedTurn>();
    startTurn.mockReturnValue(request.promise);

    const { result } = renderTurn();

    act(() => {
      void result.current.send("Where is S1?");
    });
    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));
    const signal = startTurn.mock.calls[0]?.[3].signal;
    expect(signal?.aborted).toBe(false);

    act(() => {
      result.current.stop();
    });
    expect(signal?.aborted).toBe(true);

    await act(async () => {
      request.resolve(started);
    });

    await waitFor(() => expect(stopTurn).toHaveBeenCalledWith("atrn_1"));
    expect(attachTurn).not.toHaveBeenCalled();
    expect(result.current.isActive).toBe(false);
    expect(result.current.turn?.status).toBe("error");
  });

  it("stops the turn the server made when the aborted request lost its id", async () => {
    activeTurn.mockResolvedValueOnce(null).mockResolvedValue({
      id: "atrn_orphan",
      threadId: "athr_1",
      status: "Running",
      origin: "Person",
      input: "Where is S1?",
    });
    startTurn.mockImplementation(
      (_threadId, _content, _options, { signal }) =>
        new Promise((_resolve, reject) => {
          signal?.addEventListener("abort", () =>
            reject(new DOMException("The operation was aborted.", "AbortError")),
          );
        }),
    );

    const { result } = renderTurn();

    act(() => {
      void result.current.send("Where is S1?");
    });
    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));

    act(() => {
      result.current.stop();
    });

    await waitFor(() => expect(stopTurn).toHaveBeenCalledWith("atrn_orphan"));
    expect(attachTurn).not.toHaveBeenCalled();
  });

  it("leaves a turn it did not ask alone", async () => {
    activeTurn.mockResolvedValueOnce(null).mockResolvedValue({
      id: "atrn_other",
      threadId: "athr_1",
      status: "Running",
      origin: "DecisionFollowUp",
      input: "",
    });
    startTurn.mockImplementation(
      (_threadId, _content, _options, { signal }) =>
        new Promise((_resolve, reject) => {
          signal?.addEventListener("abort", () =>
            reject(new DOMException("The operation was aborted.", "AbortError")),
          );
        }),
    );

    const { result } = renderTurn();

    act(() => {
      void result.current.send("Where is S1?");
    });
    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));

    act(() => {
      result.current.stop();
    });

    await waitFor(() => expect(activeTurn).toHaveBeenCalledTimes(2));
    expect(stopTurn).not.toHaveBeenCalled();
  });

  it("never sends a question stopped while the conversation was being checked", async () => {
    const lookup = deferred<ActiveTurn | null>();
    activeTurn.mockReturnValue(lookup.promise);

    const { result } = renderTurn();

    act(() => {
      void result.current.send("Where is S1?");
    });
    act(() => {
      result.current.stop();
    });
    await act(async () => {
      lookup.resolve(null);
    });

    expect(startTurn).not.toHaveBeenCalled();
    expect(result.current.isActive).toBe(false);
    expect(result.current.turn).toMatchObject({
      userContent: "Where is S1?",
      status: "error",
    });
  });
});

/**
 * When a reply is saved, the server starts the next message the person queued
 * and names it on the ending stream (next_turn, before done). The view follows
 * it straight on, showing the queued words as the new question with the
 * records they named, rather than going quiet until the live list catches up.
 */
describe("useAssistantTurn following the queue", () => {
  function done(sequence: number): AssistantStreamEvent {
    return {
      event: "done",
      data: {
        thread: {} as SendMessageResult["thread"],
        messages: [savedMessage(sequence), savedMessage(sequence + 1)],
        reply: `m${sequence + 1}`,
        refused: false,
        proposals: null,
        proposalsUnrecorded: false,
        artifacts: null,
      },
    } as AssistantStreamEvent;
  }

  it("follows the reply the queue started once the first is saved", async () => {
    activeTurn.mockResolvedValue(null);
    startTurn.mockResolvedValue(started);
    const second = deferred<undefined>();
    const seen: string[] = [];
    attachTurn.mockImplementation(async (turnId, onEvent) => {
      seen.push(turnId);
      if (turnId === "atrn_1") {
        onEvent(
          {
            event: "next_turn",
            data: {
              turnId: "atrn_2",
              threadId: "athr_1",
              queuedId: "aqm_1",
              input: "Then bill it.",
            },
          },
          "1",
        );
        onEvent(done(0), "2");
        return;
      }
      await second.promise;
    });

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(queries.assistant.queue("athr_1").queryKey, {
      items: [
        {
          id: "aqm_1",
          threadId: "athr_1",
          content: "Then bill it.",
          request: {
            mentions: [{ type: "shipment", id: "shp_1", label: "S1" }],
            attachmentDocumentIds: [],
          },
          position: 1,
          steer: false,
          version: 0,
          createdAt: 0,
        },
      ],
    });
    const { result } = renderTurn(client);

    await act(async () => {
      void result.current.send("m0");
    });

    await waitFor(() => expect(seen).toEqual(["atrn_1", "atrn_2"]));
    expect(result.current.isActive).toBe(true);
    expect(result.current.turn?.userContent).toBe("Then bill it.");
    expect(result.current.turn?.mentions).toEqual([{ type: "shipment", id: "shp_1", label: "S1" }]);

    await act(async () => {
      second.resolve(undefined);
    });
  });

  it("follows a reply the queue started on request, unless one is already being followed", async () => {
    const hung = deferred<undefined>();
    attachTurn.mockImplementation(async () => hung.promise);

    const { result } = renderTurn();

    await act(async () => {
      void result.current.followStarted("atrn_9", "Send the invoice.");
    });
    await waitFor(() => expect(attachTurn).toHaveBeenCalledTimes(1));
    expect(attachTurn.mock.calls[0]?.[0]).toBe("atrn_9");
    expect(result.current.turn?.userContent).toBe("Send the invoice.");

    await act(async () => {
      void result.current.followStarted("atrn_10", "Something else.");
    });
    expect(attachTurn).toHaveBeenCalledTimes(1);

    await act(async () => {
      hung.resolve(undefined);
    });
  });
});

/**
 * The agent is told where the conversation is being had, so "where am I?"
 * on the Desk is answered with the Desk rather than with the page the person
 * came from.
 */
describe("useAssistantTurn surface", () => {
  function renderOn(surface?: "Desk" | "Assistant") {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    return renderHook(() => useAssistantTurn("athr_1", () => null, undefined, surface), {
      wrapper,
    });
  }

  it.each(["Desk", "Assistant"] as const)(
    "sends the %s surface with the question",
    async (surface) => {
      activeTurn.mockResolvedValue(null);
      startTurn.mockReturnValue(new Promise(() => undefined));

      const { result } = renderOn(surface);
      act(() => {
        void result.current.send("What does this do?");
      });

      await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));
      expect(startTurn.mock.calls[0]?.[2]).toMatchObject({ surface });
    },
  );

  it("names no surface when the view gave none", async () => {
    activeTurn.mockResolvedValue(null);
    startTurn.mockReturnValue(new Promise(() => undefined));

    const { result } = renderOn();
    act(() => {
      void result.current.send("What does this do?");
    });

    await waitFor(() => expect(startTurn).toHaveBeenCalledTimes(1));
    expect(startTurn.mock.calls[0]?.[2]).toMatchObject({ surface: undefined });
  });
});
