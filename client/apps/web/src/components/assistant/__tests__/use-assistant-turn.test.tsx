import type { ActiveTurn, StartedTurn } from "@/services/assistant";
import type { AssistantStreamEvent } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
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

function renderTurn() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useAssistantTurn("athr_1", () => null), { wrapper });
}

afterEach(() => {
  cleanup();
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
