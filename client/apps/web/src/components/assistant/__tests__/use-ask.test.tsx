import type { KeptAsk, StartedAsk } from "@/services/assistant";
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
) => Promise<StartedAsk>;

type Attach = (
  askId: string,
  onEvent: (event: AssistantStreamEvent, cursor: string) => void,
  options: { cursor?: string; signal?: AbortSignal },
) => Promise<void>;

const startAsk = vi.hoisted(() => vi.fn<StartAsk>());
const attachAsk = vi.hoisted(() => vi.fn<Attach>());
const stopAsk = vi.hoisted(() => vi.fn(async (_askId: string) => undefined));
const keepAsk = vi.hoisted(() => vi.fn<(askId: string) => Promise<KeptAsk>>());
const startTurn = vi.hoisted(() => vi.fn(async (_threadId: string, _content: string) => ({})));

vi.mock("@/services/assistant", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/assistant")>()),
  assistantService: {
    startAsk: (...args: Parameters<StartAsk>) => startAsk(...args),
    attachAsk: (...args: Parameters<Attach>) => attachAsk(...args),
    stopAsk: (askId: string) => stopAsk(askId),
    keepAsk: (askId: string) => keepAsk(askId),
    startTurn: (threadId: string, content: string) => startTurn(threadId, content),
  },
}));

const started: StartedAsk = {
  askId: "pask_1",
  streamUrl: "/api/v1/assistant/ask/pask_1/stream/",
};

const deskThread = assistantThreadSchema.parse({
  id: "athr_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  userId: "usr_1",
  agentDefinitionId: "agdef_1",
  status: "Active",
  origin: "Desk",
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
  attachAsk.mockReset();
  stopAsk.mockClear();
  keepAsk.mockReset();
  startTurn.mockClear();
});

/**
 * A palette answer is read in the palette and nowhere else. So closing the
 * palette, or asking something else, takes the question back rather than
 * leaving a worker to write an answer nobody will see.
 */
describe("useAsk reset", () => {
  it("withdraws a question still on its way, and stops the answer it becomes", async () => {
    const pending = deferred<StartedAsk>();
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

    expect(stopAsk).toHaveBeenCalledWith("pask_1");
    expect(attachAsk).not.toHaveBeenCalled();
    expect(result.current.turn).toBeNull();
  });

  it("stops an answer being read", async () => {
    startAsk.mockResolvedValue(started);
    let readerSignal: AbortSignal | undefined;
    attachAsk.mockImplementation(
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
    await waitFor(() => expect(attachAsk).toHaveBeenCalledTimes(1));

    act(() => result.current.reset());

    expect(readerSignal?.aborted).toBe(true);
    expect(stopAsk).toHaveBeenCalledTimes(1);
    expect(stopAsk).toHaveBeenCalledWith("pask_1");
    expect(result.current.turn).toBeNull();
  });

  it("stops the answer to the last question when another is asked", async () => {
    startAsk.mockResolvedValueOnce(started).mockResolvedValueOnce({
      askId: "pask_2",
      streamUrl: "/api/v1/assistant/ask/pask_2/stream/",
    });
    attachAsk.mockImplementation(
      (_turnId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachAsk).toHaveBeenCalledTimes(1));

    act(() => {
      void result.current.ask("Where is S2?");
    });
    await waitFor(() => expect(attachAsk).toHaveBeenCalledTimes(2));

    expect(stopAsk).toHaveBeenCalledTimes(1);
    expect(stopAsk).toHaveBeenCalledWith("pask_1");
    expect(attachAsk.mock.calls[1][0]).toBe("pask_2");
  });

  it("stops nothing once the answer has ended", async () => {
    startAsk.mockResolvedValue(started);
    attachAsk.mockImplementation(async (_turnId, onEvent) => {
      onEvent({ event: "done", data: null }, "1-0");
    });
    const { result } = renderAsk();

    await act(async () => {
      await result.current.ask("Where is S1?");
    });

    act(() => result.current.reset());

    expect(stopAsk).not.toHaveBeenCalled();
  });

  it("carries an answered question into a new Desk conversation", async () => {
    startAsk.mockResolvedValue(started);
    attachAsk.mockImplementation(async (_askId, onEvent) => {
      onEvent({ event: "done", data: null }, "3");
    });
    keepAsk.mockResolvedValue({ thread: deskThread, unanswered: "" });
    const { result } = renderAsk();

    await act(async () => {
      await result.current.ask("Where is S1?");
    });
    let kept: string | null = null;
    await act(async () => {
      kept = await result.current.keep();
    });

    expect(keepAsk).toHaveBeenCalledWith("pask_1");
    expect(kept).toBe("athr_1");
    expect(startTurn).not.toHaveBeenCalled();
    expect(result.current.keepable).toBe(true);
  });

  it("carries the question once however many times it is asked to", async () => {
    startAsk.mockResolvedValue(started);
    attachAsk.mockImplementation(async (_askId, onEvent) => {
      onEvent({ event: "done", data: null }, "3");
    });
    const pending = deferred<KeptAsk>();
    keepAsk.mockReturnValue(pending.promise);
    const { result } = renderAsk();

    await act(async () => {
      await result.current.ask("Where is S1?");
    });
    let first: Promise<string | null> = Promise.resolve(null);
    let second: Promise<string | null> = Promise.resolve(null);
    act(() => {
      first = result.current.keep();
      second = result.current.keep();
    });
    await act(async () => {
      pending.resolve({ thread: deskThread, unanswered: "" });
    });

    expect(keepAsk).toHaveBeenCalledTimes(1);
    await expect(first).resolves.toBe("athr_1");
    await expect(second).resolves.toBe("athr_1");
  });

  it("asks a question that went unanswered again in the Desk", async () => {
    startAsk.mockResolvedValue(started);
    attachAsk.mockImplementation(async (_askId, onEvent) => {
      onEvent({ event: "error", data: { message: "Nova couldn't answer that just now." } }, "2");
    });
    keepAsk.mockResolvedValue({ thread: deskThread, unanswered: "Where is S1?" });
    const { result } = renderAsk();

    await act(async () => {
      await result.current.ask("Where is S1?");
    });
    expect(result.current.keepable).toBe(true);
    await act(async () => {
      await result.current.keep();
    });

    expect(startTurn).toHaveBeenCalledWith("athr_1", "Where is S1?");
  });

  it("offers nothing to carry when the question never reached Nova", async () => {
    startAsk.mockRejectedValue(new Error("allowance used"));
    const { result } = renderAsk();

    await act(async () => {
      await result.current.ask("Where is S1?");
    });

    expect(result.current.keepable).toBe(false);
  });

  it("carries nothing while the answer is still being written", async () => {
    startAsk.mockResolvedValue(started);
    attachAsk.mockImplementation(
      (_askId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachAsk).toHaveBeenCalledTimes(1));

    let kept: string | null = null;
    await act(async () => {
      kept = await result.current.keep();
    });

    expect(kept).toBeNull();
    expect(keepAsk).not.toHaveBeenCalled();
  });

  it("still stops the answer when Stop is pressed", async () => {
    startAsk.mockResolvedValue(started);
    attachAsk.mockImplementation(
      (_turnId, _onEvent, options) =>
        new Promise<void>((resolve) => {
          options.signal?.addEventListener("abort", () => resolve(), { once: true });
        }),
    );
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(attachAsk).toHaveBeenCalledTimes(1));

    act(() => result.current.stop());

    expect(stopAsk).toHaveBeenCalledWith("pask_1");
  });

  it("withdraws a question still on its way when Stop is pressed", async () => {
    const pending = deferred<StartedAsk>();
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
    expect(stopAsk).toHaveBeenCalledWith("pask_1");
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
    attachAsk.mockImplementation(
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
    await waitFor(() => expect(attachAsk).toHaveBeenCalledTimes(1));
    expect(openTurnReaderCount()).toBe(1);

    act(() => {
      releaseTurnReaders();
    });

    expect(readerSignal?.aborted).toBe(true);
    expect(stopAsk).not.toHaveBeenCalled();
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
    attachAsk.mockResolvedValue(undefined);
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
    attachAsk.mockResolvedValue(undefined);
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
    expect(stopAsk).not.toHaveBeenCalled();
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
    expect(stopAsk).not.toHaveBeenCalled();
  });
});
