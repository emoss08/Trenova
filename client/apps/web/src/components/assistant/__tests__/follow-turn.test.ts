import type { StartedTurn } from "@/services/assistant";
import type { AssistantStreamEvent } from "@/types/assistant";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { followTurn, runTurn } from "../follow-turn";

type Attach = (
  turnId: string,
  onEvent: (event: AssistantStreamEvent, cursor: string) => void,
  options: { cursor?: string; signal?: AbortSignal },
) => Promise<void>;

const attachTurn = vi.hoisted(() => vi.fn<Attach>());
const stopTurn = vi.hoisted(() => vi.fn(async (_turnId: string) => undefined));

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
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

const done: AssistantStreamEvent = {
  event: "done",
  data: {
    thread: { id: "athr_1" } as never,
    messages: [],
    reply: "S1 is in Dallas.",
    refused: false,
    proposals: null,
    proposalsUnrecorded: false,
  },
};

function dropped(): Error {
  return new TypeError("network error");
}

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  attachTurn.mockReset();
  stopTurn.mockClear();
});

/**
 * The reconnect allowance is for drops in a row. A long reply that loses its
 * connection every so often, and is reattached each time, used to run out of
 * reconnects on the sixth blip and give up on a turn that was still going.
 */
describe("followTurn reconnects", () => {
  it("keeps reattaching as long as each reattach delivers a frame", async () => {
    const drops = 12;
    let call = 0;
    attachTurn.mockImplementation(async (_turnId, onEvent) => {
      call += 1;
      if (call > drops) {
        onEvent(done, `e${call}`);
        return;
      }
      onEvent({ event: "delta", data: { text: `part ${call} ` } }, `e${call}`);
      throw dropped();
    });

    const onEvent = vi.fn();
    const following = followTurn("atrn_1", { signal: new AbortController().signal, onEvent });
    await vi.runAllTimersAsync();
    await expect(following).resolves.toBeUndefined();

    expect(attachTurn).toHaveBeenCalledTimes(drops + 1);
    expect(onEvent).toHaveBeenLastCalledWith(done);
    // Each reattach resumes from the last frame applied.
    expect(attachTurn.mock.calls[drops]?.[2]).toMatchObject({ cursor: `e${drops}` });
  });

  it("gives up once reattaching fails every time in a row", async () => {
    attachTurn.mockRejectedValue(dropped());

    const following = followTurn("atrn_1", {
      signal: new AbortController().signal,
      onEvent: vi.fn(),
    });
    const outcome = expect(following).rejects.toThrow("network error");
    await vi.runAllTimersAsync();
    await outcome;

    expect(attachTurn).toHaveBeenCalledTimes(6);
  });

  it("starts counting again after a reattach that delivered a frame", async () => {
    let call = 0;
    attachTurn.mockImplementation(async (_turnId, onEvent) => {
      call += 1;
      // Four failures, a frame, then failures only: the allowance restarts at
      // the frame, so the give-up comes five failed reattaches after it.
      if (call === 5) {
        onEvent({ event: "delta", data: { text: "still here" } }, "e5");
      }
      throw dropped();
    });

    const following = followTurn("atrn_1", {
      signal: new AbortController().signal,
      onEvent: vi.fn(),
    });
    const outcome = expect(following).rejects.toThrow("network error");
    await vi.runAllTimersAsync();
    await outcome;

    expect(attachTurn).toHaveBeenCalledTimes(10);
  });
});

/**
 * Stop pressed while the question is on its way. The request is aborted, and
 * a turn that exists anyway — its id arrived after Stop, or the server made it
 * before the abort reached it — is stopped rather than followed.
 */
describe("runTurn withdrawn before it started", () => {
  it("hands the abort signal to the start request", async () => {
    attachTurn.mockImplementation(async (_turnId, onEvent) => onEvent(done, "e1"));
    const controller = new AbortController();
    const start = vi.fn(async (_signal: AbortSignal) => started);

    await runTurn(start, { signal: controller.signal, onEvent: vi.fn() });

    expect(start).toHaveBeenCalledWith(controller.signal);
  });

  it("stops a turn whose id arrives after Stop and never attaches to it", async () => {
    const controller = new AbortController();
    const onTurnStarted = vi.fn();
    const start = vi.fn(async () => {
      controller.abort();
      return started;
    });

    await runTurn(start, { signal: controller.signal, onTurnStarted, onEvent: vi.fn() });

    expect(stopTurn).toHaveBeenCalledWith("atrn_1");
    expect(onTurnStarted).not.toHaveBeenCalled();
    expect(attachTurn).not.toHaveBeenCalled();
  });

  it("stops the turn the server made anyway when the aborted request lost its id", async () => {
    const controller = new AbortController();
    const start = vi.fn(async () => {
      controller.abort();
      throw new DOMException("The operation was aborted.", "AbortError");
    });
    const findWithdrawn = vi.fn(async () => "atrn_orphan");

    await expect(
      runTurn(start, { signal: controller.signal, onEvent: vi.fn(), findWithdrawn }),
    ).resolves.toBeUndefined();

    expect(findWithdrawn).toHaveBeenCalledTimes(1);
    expect(stopTurn).toHaveBeenCalledWith("atrn_orphan");
    expect(attachTurn).not.toHaveBeenCalled();
  });

  it("stops nothing when the aborted request made no turn", async () => {
    const controller = new AbortController();
    const start = vi.fn(async () => {
      controller.abort();
      throw new DOMException("The operation was aborted.", "AbortError");
    });

    await runTurn(start, {
      signal: controller.signal,
      onEvent: vi.fn(),
      findWithdrawn: async () => null,
    });

    expect(stopTurn).not.toHaveBeenCalled();
  });

  it("still reports a start that failed on its own", async () => {
    const start = vi.fn(async () => {
      throw new Error("already working on a reply");
    });

    await expect(
      runTurn(start, { signal: new AbortController().signal, onEvent: vi.fn() }),
    ).rejects.toThrow("already working on a reply");
    expect(stopTurn).not.toHaveBeenCalled();
  });
});
