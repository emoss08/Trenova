import type { StartedTurn } from "@/services/assistant";
import type { AssistantStreamEvent } from "@/types/assistant";
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

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      startAsk: (...args: Parameters<StartAsk>) => startAsk(...args),
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
});

/**
 * Closing the palette resets the ask. It used to abort the one signal the
 * question and its reader shared, so a palette closed while the question was
 * on its way withdrew it, and one closed just after stopped the turn the
 * server had already begun. Either way the question was lost. Now only Stop
 * takes a question back; a reset lets go of the reader and the answer is
 * written to the end, to arrive as a notification.
 */
describe("useAsk reset", () => {
  it("leaves a question on its way asked, and never stops the turn it becomes", async () => {
    const pending = deferred<StartedTurn>();
    startAsk.mockReturnValue(pending.promise);
    const { result } = renderAsk();

    act(() => {
      void result.current.ask("Where is S1?");
    });
    await waitFor(() => expect(startAsk).toHaveBeenCalledTimes(1));
    const request = startAsk.mock.calls[0][2];

    act(() => result.current.reset());

    expect(request.signal?.aborted).toBe(false);

    await act(async () => {
      pending.resolve(started);
    });

    expect(stopTurn).not.toHaveBeenCalled();
    expect(attachTurn).not.toHaveBeenCalled();
    expect(result.current.turn).toBeNull();
  });

  it("lets go of an answer being read without stopping it", async () => {
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
