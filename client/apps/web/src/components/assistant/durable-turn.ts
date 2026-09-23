import { apiService } from "@/services/api";
import { AssistantStreamError } from "@/services/assistant";
import type { AssistantPageContext, AssistantStreamEvent } from "@/types/assistant";

/**
 * How long to wait before reattaching to a turn whose connection dropped.
 *
 * A dropped connection no longer means a lost reply: the turn is running on a
 * worker and its events are waiting in its stream. So the right response is to
 * go back and read from where we stopped, and only give up once it is clear
 * the reader — not the turn — is what is broken.
 */
const reattachDelaysMs = [250, 750, 2000, 5000, 5000];

/**
 * Whether this server answers on a worker, asked once per session.
 *
 * It is a property of the deployment, not of the question, so asking per turn
 * would add a round trip to every reply to learn something that cannot have
 * changed. A failed check reads as "no": the path that needs no extra
 * infrastructure is the one that always works.
 */
let capability: Promise<boolean> | null = null;

export function durableTurnsAvailable(): Promise<boolean> {
  capability ??= apiService.assistantService
    .capabilities()
    .then((result) => result.durableTurns)
    .catch(() => false);

  return capability;
}

/** Forgets the cached answer. For tests, and for a session that reconnects. */
export function resetDurableTurnsCapability(): void {
  capability = null;
}

export type DurableTurnOptions = {
  threadId: string;
  content: string;
  context: AssistantPageContext | null;
  providerId: string;
  attachmentDocumentIds: string[];
  mentions: unknown[];
  followUpProposalId?: string;
  signal: AbortSignal;
  /** Called once the worker has the question, before any of the answer. */
  onTurnStarted: (turnId: string) => void;
  onEvent: (event: AssistantStreamEvent) => void;
};

/**
 * Asks a question durably and follows the answer.
 *
 * The question goes to a worker, so this function's own lifetime has nothing
 * to do with the reply's: it can drop the connection and pick it up again, and
 * the turn neither notices nor restarts. What it cannot do is lose its place,
 * which is why the cursor is the id of the last event *applied* rather than
 * the last received — a connection that dies mid-frame delivers an event this
 * side never folded in, and resuming past it would silently skip it.
 */
export async function runDurableTurn(options: DurableTurnOptions): Promise<void> {
  const started = await apiService.assistantService.startTurn(options.threadId, options.content, {
    context: options.context,
    providerId: options.providerId,
    attachmentDocumentIds: options.attachmentDocumentIds,
    mentions: options.mentions as never,
    followUpProposalId: options.followUpProposalId,
  });
  options.onTurnStarted(started.turnId);

  await followTurn(started.turnId, "", options);
}

/**
 * Reattaches to a turn already in progress — a reopened tab, or a reply that
 * was still being written when the page was last closed.
 */
export async function followExistingTurn(
  turnId: string,
  options: Pick<DurableTurnOptions, "signal" | "onEvent">,
): Promise<void> {
  await followTurn(turnId, "", { ...options } as DurableTurnOptions);
}

async function followTurn(
  turnId: string,
  from: string,
  options: Pick<DurableTurnOptions, "signal" | "onEvent">,
): Promise<void> {
  let cursor = from;
  let terminal = false;
  let attempt = 0;

  const apply = (event: AssistantStreamEvent, id: string) => {
    options.onEvent(event);
    // Only after the event has been folded in. A cursor moved on receipt
    // would skip whatever arrived in a frame this side never processed.
    if (id) {
      cursor = id;
    }
    if (event.event === "done" || event.event === "refused" || event.event === "error") {
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
