import type { StartedTurn } from "@/services/assistant";
import { assistantThreadSchema, type AssistantStreamEvent } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { openTurnReaderCount, releaseTurnReaders } from "../turn-readers";
import { useAsk } from "../use-ask";

type StartAsk = (
  content: string,
  options: unknown,
  request: { signal?: AbortSignal },
) => Promise<StartedTurn>;

type Attach = (
  turnId: string,
  onEvent: (event: AssistantStreamEvent, cursor: string) => void,
  options: { cursor?: string; signal?: AbortSignal },
) => Promise<void>;

const startAsk = vi.hoisted(() => vi.fn<StartAsk>());
const attachTurn = vi.hoisted(() => vi.fn<Attach>());
const stopTurn = vi.hoisted(() => vi.fn(async (_turnId: string) => undefined));
const updateThread = vi.hoisted(() => vi.fn(async (_threadId: string, _patch: unknown) => ({})));

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      startAsk: (...args: Parameters<StartAsk>) => startAsk(...args),
      attachTurn: (...args: Parameters<Attach>) => attachTurn(...args),
      stopTurn: (turnId: string) => stopTurn(turnId),
      updateThread: (threadId: string, patch: unknown) => updateThread(threadId, patch),
    },
  },
}));

const started: StartedTurn = {
  turnId: "atrn_1",
  threadId: "athr_1",
  streamUrl: "/api/v1/assistant/turns/atrn_1/stream/",
  status: "Running",
};

const askThread = assistantThreadSchema.parse({
  id: "athr_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  userId: "usr_1",
  agentDefinitionId: "agdef_1",
  status: "Active",
  origin: "Ask",
  createdAt: 1,
  updatedAt: 1,
});

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

function renderAsk() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useAsk(), { wrapper });
}

afterEach(() => {
  cleanup();
  releaseTurnReaders();
  startAsk.mockReset();
  attachTurn.mockReset();
  stopTurn.mockClear();
  updateThread.mockClear();
});

/**
 * A palette answer is read in the palette and nowhere else: the server
 * neither announces it nor lists it in the Desk. So closing the palette, or
 * asking something else, takes the question back rather than leaving a worker
 * to write an answer nobody will see. A question kept for the Desk is a Desk
 * conversation, and closing the palette leaves its turn running there.
 */
describe("useAsk reset", () => {
  it("withdraws a question still on its way, and stops the turn it becomes", async () => {
    const pending = deferred<StartedTurn>();
    startAsk.mockReturnValue(pending.promise);
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(startAsk).toHaveBeenCalledTimes(1));
    const request = startAsk.mock.calls[0][2];

    act(() => result.current.reset());

    expect(request.signal?.aborted).toBe(true);

    await act(async () => {
      pending.resolve(started);
    });

    expect(stopTurn).toHaveBeenCalledWith("atrn_1");
    expect(attachTurn).not.toHaveBeenCalled();
    expect(result.current.turn).toBeNull();
  });

  it("stops an answer being read", async () => {
    startAsk.mockResolvedValue(started);
    let readerSignal: AbortSignal | undefined;
    attachTurn.mockImplementation(
      (_turnId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          readerSignal = options.signal;
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachTurn).toHaveBeenCalledTimes(1));

    act(() => result.current.reset());

    expect(readerSignal?.aborted).toBe(true);
    expect(stopTurn).toHaveBeenCalledTimes(1);
    expect(stopTurn).toHaveBeenCalledWith("atrn_1");
    expect(result.current.turn).toBeNull();
  });

  it("stops the answer to the last question when another is asked", async () => {
    startAsk.mockResolvedValueOnce(started).mockResolvedValueOnce({
      ...started,
      turnId: "atrn_2",
      threadId: "athr_2",
    });
    attachTurn.mockImplementation(
      (_turnId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachTurn).toHaveBeenCalledTimes(1));

    act(() => {
      void result.current.ask("Where is S2?");
    });
    await waitFor(() => expect(attachTurn).toHaveBeenCalledTimes(2));

    expect(stopTurn).toHaveBeenCalledTimes(1);
    expect(stopTurn).toHaveBeenCalledWith("atrn_1");
    expect(attachTurn.mock.calls[1][0]).toBe("atrn_2");
  });

  it("stops nothing once the answer has ended", async () => {
    startAsk.mockResolvedValue(started);
    attachTurn.mockImplementation(async (_turnId, onEvent) => {
      onEvent({ event: "done", data: null }, "1-0");
    });
    const { result } = renderAsk();

    await act(async () => {
      await result.current.ask("Where is S1?");
    });

    act(() => result.current.reset());

    expect(stopTurn).not.toHaveBeenCalled();
  });

  it("leaves a kept question's turn running when the palette closes", async () => {
    startAsk.mockResolvedValue({ ...started, thread: askThread });
    let readerSignal: AbortSignal | undefined;
    attachTurn.mockImplementation(
      (_turnId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          readerSignal = options.signal;
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachTurn).toHaveBeenCalledTimes(1));

    let kept: string | null = null;
    await act(async () => {
      kept = await result.current.keep();
    });
    act(() => result.current.reset());

    expect(kept).toBe("athr_1");
    expect(updateThread).toHaveBeenCalledWith("athr_1", { keep: true });
    expect(readerSignal?.aborted).toBe(true);
    expect(stopTurn).not.toHaveBeenCalled();
  });

  it("still stops the turn when Stop is pressed", async () => {
    startAsk.mockResolvedValue(started);
    attachTurn.mockImplementation(
      (_turnId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachTurn).toHaveBeenCalledTimes(1));

    act(() => result.current.stop());

    expect(stopTurn).toHaveBeenCalledWith("atrn_1");
  });

  it("withdraws a question still on its way when Stop is pressed", async () => {
    const pending = deferred<StartedTurn>();
    startAsk.mockReturnValue(pending.promise);
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(startAsk).toHaveBeenCalledTimes(1));

    act(() => result.current.stop());

    expect(startAsk.mock.calls[0][2].signal?.aborted).toBe(true);
    await act(async () => {
      pending.resolve(started);
    });
    expect(stopTurn).toHaveBeenCalledWith("atrn_1");
  });
});

/**
 * Signing out lets go of every reader before the session ends, so none of
 * them reattaches against a session that no longer exists.
 */
describe("useAsk and sign-out", () => {
  it("registers its reader so sign-out can release it", async () => {
    startAsk.mockResolvedValue(started);
    let readerSignal: AbortSignal | undefined;
    attachTurn.mockImplementation(
      (_turnId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          readerSignal = options.signal;
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachTurn).toHaveBeenCalledTimes(1));
    expect(openTurnReaderCount()).toBe(1);

    act(() => {
      releaseTurnReaders();
    });

    expect(readerSignal?.aborted).toBe(true);
    expect(stopTurn).not.toHaveBeenCalled();
  });
});

/**
 * A question from the palette carries the records its search found that it
 * names. Finding them is part of asking: a search that fails still asks the
 * question, and Stop or closing the palette withdraws it.
 */
describe("useAsk found mentions", () => {
  const shipment = { type: "shipment", id: "shp_1", label: "SEED-PAY-002" };

  it("asks with the records found while the question was on its way", async () => {
    startAsk.mockResolvedValue(started);
    attachTurn.mockResolvedValue(undefined);
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Tell me about SEED-PAY-002?", {
        findMentions: async () => [shipment],
      });
    });
    await waitFor(() => expect(startAsk).toHaveBeenCalledTimes(1));

    expect(startAsk.mock.calls[0][1]).toEqual({ context: undefined, mentions: [shipment] });
    expect(result.current.turn?.mentions).toEqual([shipment]);
  });

  it("asks without them when finding them fails", async () => {
    startAsk.mockResolvedValue(started);
    attachTurn.mockResolvedValue(undefined);
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Tell me about SEED-PAY-002?", {
        findMentions: async () => {
          throw new Error("search is down");
        },
      });
    });
    await waitFor(() => expect(startAsk).toHaveBeenCalledTimes(1));

    expect(startAsk.mock.calls[0][1]).toEqual({ context: undefined, mentions: [] });
  });

  it("never asks a question the palette closed on while its records were being found", async () => {
    const found = deferred<(typeof shipment)[]>();
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Tell me about SEED-PAY-002?", {
        findMentions: () => found.promise,
      });
    });
    act(() => result.current.reset());
    await act(async () => {
      found.resolve([shipment]);
    });

    expect(startAsk).not.toHaveBeenCalled();
    expect(stopTurn).not.toHaveBeenCalled();
  });

  it("never asks a question stopped while its records were being found", async () => {
    const found = deferred<(typeof shipment)[]>();
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Tell me about SEED-PAY-002?", {
        findMentions: () => found.promise,
      });
    });
    act(() => result.current.stop());
    await act(async () => {
      found.resolve([shipment]);
    });

    expect(startAsk).not.toHaveBeenCalled();
    expect(stopTurn).not.toHaveBeenCalled();
  });
});
