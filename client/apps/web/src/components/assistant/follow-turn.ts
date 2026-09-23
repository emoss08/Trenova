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
  signal: AbortSignal;
  /** Called once a worker has the question, before any of the answer. */
  onTurnStarted?: (turn: StartedTurn) => void;
  onEvent: (event: AssistantStreamEvent) => void;
};

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
  start: () => Promise<StartedTurn>,
  options: RunTurnOptions,
): Promise<void> {
  const started = await start();
  options.onTurnStarted?.(started);

  await followTurn(started.turnId, options);
}

async function followTurn(turnId: string, options: RunTurnOptions): Promise<void> {
  let cursor = "";
  let terminal = false;
  let attempt = 0;

  const apply = (event: AssistantStreamEvent, id: string) => {
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
