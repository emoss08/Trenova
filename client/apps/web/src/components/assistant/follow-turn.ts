import { ApiRequestError } from "@trenova/shared/lib/api";
import { apiService } from "@/services/api";
import { AssistantStreamError, type StartedTurn } from "@/services/assistant";
import type { AssistantStreamEvent } from "@/types/assistant";

/**
 * How long to wait before reattaching to a turn whose connection dropped.
 *
 * A dropped connection is not a lost reply: the turn is running on a worker
 * and its events are waiting in its workflow. So the right response is to go
 * back and read from where we stopped, and only give up once it is clear the
 * reader — not the turn — is what is broken.
 */
const reattachDelaysMs = [250, 750, 2000, 5000, 5000];

export type RunTurnOptions = {
  /**
   * Aborting it lets go of the reply. Unless `withdrawSignal` is given, it
   * also withdraws a question the server has not yet handed back a turn for.
   */
  signal: AbortSignal;
  /**
   * Aborting it withdraws the question: before the turn is known the request
   * is cancelled and any turn it made anyway is stopped, and a turn returned
   * after it was aborted is stopped rather than followed.
   *
   * Given separately when letting go of the reader must not cost the answer —
   * a palette closed while its question is on the way has not asked for the
   * question to be taken back, and the answer arrives as a notification.
   */
  withdrawSignal?: AbortSignal;
  /** Called once a worker has the question, before any of the answer. */
  onTurnStarted?: (turn: StartedTurn) => void;
  onEvent: (event: AssistantStreamEvent) => void;
  /**
   * Finds the turn a withdrawn question made anyway: the server can record it
   * before the aborted request reaches it, and the id went down with the
   * response. Null when there is none.
   */
  findWithdrawn?: () => Promise<string | null>;
};

/**
 * Tells the server a turn is no longer wanted. The reader has already let go,
 * so a stop that fails has nobody to report to; the turn then ends on its own.
 */
export function stopTurnQuietly(turnId: string): void {
  void apiService.assistantService.stopTurn(turnId).catch(() => undefined);
}

/**
 * Asks a question and follows the answer.
 *
 * The question goes to a worker, so this function's own lifetime has nothing
 * to do with the reply's: it can drop the connection and pick it up again, and
 * the turn neither notices nor restarts. What it cannot do is lose its place,
 * which is why the cursor is the id of the last event *applied* rather than
 * the last received — a connection that dies mid-frame delivers an event this
 * side never folded in, and resuming past it would silently skip it.
 */
export async function runTurn(
  start: (signal: AbortSignal) => Promise<StartedTurn>,
  options: RunTurnOptions,
): Promise<void> {
  const withdraw = options.withdrawSignal ?? options.signal;
  let started: StartedTurn;
  try {
    started = await start(withdraw);
  } catch (error) {
    if (withdraw.aborted) {
      await stopWithdrawn(options.findWithdrawn);
      return;
    }
    if (options.signal.aborted) {
      return;
    }
    throw error;
  }

  // Stop was pressed while the question was on its way. The turn exists now
  // and would answer a question nobody is waiting on, so it is stopped rather
  // than followed.
  if (withdraw.aborted) {
    stopTurnQuietly(started.turnId);
    return;
  }
  // The reader was let go of while the question was on its way, but nobody
  // took the question back: the turn runs to its end on the server without
  // a reader, and its answer is kept.
  if (options.signal.aborted) {
    return;
  }
  options.onTurnStarted?.(started);

  await followTurn(started.turnId, options);
}

/**
 * Follows a turn already under way from its first event: one this page lost
 * when it was closed or reloaded, or one the application started, such as the
 * agent reporting what came of a decision.
 */
export async function followTurn(
  turnId: string,
  options: Pick<RunTurnOptions, "signal" | "onEvent">,
): Promise<void> {
  let cursor = "";
  let terminal = false;
  let attempt = 0;

  const apply = (event: AssistantStreamEvent, id: string) => {
    // A frame arriving means the last reattach worked. The allowance is for
    // drops in a row, not over the whole reply: a long answer that survives a
    // blip every few minutes must not run out of reconnects.
    attempt = 0;
    options.onEvent(event);
    // Only after the event has been folded in. A cursor moved on receipt
    // would skip whatever arrived in a frame this side never processed.
    if (id) {
      cursor = id;
    }
    if (event.event === "done" || event.event === "error") {
      terminal = true;
    }
  };

  for (;;) {
    try {
      await apiService.assistantService.attachTurn(turnId, apply, {
        cursor,
        signal: options.signal,
      });
    } catch (error) {
      if (options.signal.aborted || terminal) {
        return;
      }
      // A refusal to open the stream at all is the reader's problem, not a
      // dropped connection: retrying it would fail the same way.
      if (error instanceof AssistantStreamError && error.status >= 400 && error.status < 500) {
        throw error;
      }
      if (attempt >= reattachDelaysMs.length) {
        throw error;
      }
      await delay(reattachDelaysMs[attempt], options.signal);
      attempt += 1;
      continue;
    }

    if (terminal || options.signal.aborted) {
      return;
    }

    // The stream closed without an ending. The turn may still be going, so
    // this is a reattach rather than a failure — but a stream that keeps
    // closing quietly is not one to sit on for ever.
    if (attempt >= reattachDelaysMs.length) {
      return;
    }
    await delay(reattachDelaysMs[attempt], options.signal);
    attempt += 1;
  }
}

/**
 * What to tell the reader about a question that never reached a worker, or
 * whose stream could not be followed. A refusal the server wrote for them — a
 * busy conversation, a disabled agent, an empty message — is shown as written;
 * anything else is the connection's fault and says so.
 */
export function turnFailureDetail(error: unknown, fallback: string): string {
  if (error instanceof AssistantStreamError) {
    return error.message;
  }
  if (
    error instanceof ApiRequestError &&
    (error.isBusinessError() || error.isValidationError()) &&
    error.message !== ""
  ) {
    return error.message;
  }

  return fallback;
}

async function stopWithdrawn(findWithdrawn: RunTurnOptions["findWithdrawn"]): Promise<void> {
  if (!findWithdrawn) {
    return;
  }
  let turnId: string | null;
  try {
    turnId = await findWithdrawn();
  } catch {
    return;
  }
  if (turnId !== null) {
    stopTurnQuietly(turnId);
  }
}

function delay(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    const timer = setTimeout(resolve, ms);
    signal.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        resolve();
      },
      { once: true },
    );
  });
}
