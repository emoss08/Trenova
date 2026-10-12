import type { InboundMailbox } from "@/lib/graphql/inbox";

/**
 * Connecting a source, as steps: which kind of source, the address to watch
 * and the source's name, where to forward it, then waiting for a message to
 * arrive. The overlay draws the step this says and does the writes; this is
 * the order and what each step needs before the next.
 */
export type ConnectStep = "source" | "address" | "forward" | "test" | "done";

/** The numbered steps, as the flow's header lists them. */
export const NUMBERED_STEPS = ["address", "forward", "test"] as const;

export type ConnectWait = { via: "person" | "test"; since: number };

export type ConnectState = {
  step: ConnectStep;
  /** What the source is called in Triage's rail. */
  name: string;
  /** The shared address people already send to: dispatch@, pod@, ap@. */
  watch: string;
  /** Where Trenova receives the forwarded mail: the source's own address. */
  receiveAt: string;
  /** Typed by hand, so renaming the source leaves it alone. */
  receiveAtEdited: boolean;
  /** The domain the organization's other sources receive on, when there are any. */
  domain: string | null;
  mailbox: InboundMailbox | null;
  /** Where the provider posts this source's mail, when the server handed it over. */
  webhookUrl: string | null;
  saving: boolean;
  error: string | null;
  waiting: ConnectWait | null;
};

export type ConnectEvent =
  | { type: "continue" }
  | { type: "back" }
  | { type: "edit"; field: "name" | "watch" | "receiveAt"; value: string }
  | { type: "saving" }
  | { type: "created"; mailbox: InboundMailbox; webhookUrl: string | null }
  | { type: "failed"; message: string }
  | { type: "waitFor"; via: ConnectWait["via"]; since: number }
  | { type: "stopWaiting" }
  | { type: "arrived" };

const ADDRESS = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
/** A provider's clock may run a little behind ours. */
const CLOCK_SLACK_SECONDS = 120;

export function isAddress(value: string): boolean {
  return ADDRESS.test(value.trim());
}

/** The domain most of the organization's sources receive on. */
export function receivingDomain(
  mailboxes: readonly Pick<InboundMailbox, "address">[],
): string | null {
  const counts = new Map<string, number>();
  for (const { address } of mailboxes) {
    const at = address.lastIndexOf("@");
    if (at <= 0) {
      continue;
    }
    const domain = address
      .slice(at + 1)
      .trim()
      .toLowerCase();
    if (domain !== "") {
      counts.set(domain, (counts.get(domain) ?? 0) + 1);
    }
  }
  let best: string | null = null;
  let most = 0;
  for (const [domain, count] of counts) {
    if (count > most) {
      best = domain;
      most = count;
    }
  }

  return best;
}

/** "dispatch@dev.trenova.app" for a source named Dispatch; blank without a name or a domain. */
export function receivingAddress(name: string, domain: string | null): string {
  const local = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  if (local === "" || domain === null) {
    return "";
  }

  return `${local}@${domain}`;
}

export function initialConnect(
  mailbox: InboundMailbox | null,
  domain: string | null,
): ConnectState {
  return {
    step: mailbox === null ? "source" : "forward",
    name: mailbox?.name ?? "",
    watch: "",
    receiveAt: mailbox?.address ?? "",
    receiveAtEdited: false,
    domain,
    mailbox,
    webhookUrl: null,
    saving: false,
    error: null,
    waiting: null,
  };
}

/** Whether the step's own button may be pressed. */
export function canContinue(state: ConnectState): boolean {
  switch (state.step) {
    case "source":
      return true;
    case "address":
      return (
        !state.saving &&
        state.name.trim() !== "" &&
        isAddress(state.watch) &&
        isAddress(state.receiveAt) &&
        state.watch.trim().toLowerCase() !== state.receiveAt.trim().toLowerCase()
      );
    case "forward":
      return state.mailbox !== null;
    case "test":
    case "done":
      return false;
  }
}

/**
 * Back goes to the step before, except once the source exists: it cannot be
 * un-created by going back, so the forward step is where it starts.
 */
export function canGoBack(state: ConnectState): boolean {
  switch (state.step) {
    case "address":
      return !state.saving;
    case "test":
      return true;
    case "source":
    case "forward":
    case "done":
      return false;
  }
}

export function connectReducer(state: ConnectState, event: ConnectEvent): ConnectState {
  switch (event.type) {
    case "continue":
      if (!canContinue(state)) {
        return state;
      }
      if (state.step === "source") {
        return { ...state, step: "address", error: null };
      }
      if (state.step === "forward") {
        return { ...state, step: "test", error: null };
      }
      return state;
    case "back":
      if (!canGoBack(state)) {
        return state;
      }
      if (state.step === "address") {
        return { ...state, step: "source", error: null };
      }
      return { ...state, step: "forward", waiting: null, error: null };
    case "edit": {
      const value = event.value;
      if (event.field === "receiveAt") {
        return { ...state, receiveAt: value, receiveAtEdited: true, error: null };
      }
      if (event.field === "watch") {
        return { ...state, watch: value, error: null };
      }
      return {
        ...state,
        name: value,
        receiveAt: state.receiveAtEdited ? state.receiveAt : receivingAddress(value, state.domain),
        error: null,
      };
    }
    case "saving":
      return state.step === "address" ? { ...state, saving: true, error: null } : state;
    case "failed":
      return { ...state, saving: false, error: event.message };
    case "created":
      return {
        ...state,
        step: "forward",
        mailbox: event.mailbox,
        receiveAt: event.mailbox.address,
        webhookUrl: event.webhookUrl,
        saving: false,
        error: null,
      };
    case "waitFor":
      return state.step === "test"
        ? { ...state, waiting: { via: event.via, since: event.since }, error: null }
        : state;
    case "stopWaiting":
      return { ...state, waiting: null };
    case "arrived":
      return state.step === "test" ? { ...state, step: "done", waiting: null, error: null } : state;
  }
}

/**
 * Whether the mail being waited for has arrived: anything at all since the
 * person said they sent it, or Trenova's own test, which is recorded only
 * when the message carrying its token comes back.
 */
export function testArrived(
  mailbox: Pick<InboundMailbox, "lastReceivedAt" | "testSentAt" | "testReceivedAt">,
  waiting: ConnectWait,
): boolean {
  const since = waiting.since - CLOCK_SLACK_SECONDS;
  if (waiting.via === "test") {
    return (
      mailbox.testReceivedAt !== null &&
      mailbox.testSentAt !== null &&
      mailbox.testReceivedAt >= mailbox.testSentAt &&
      mailbox.testReceivedAt >= since
    );
  }

  return (
    (mailbox.lastReceivedAt !== null && mailbox.lastReceivedAt >= since) ||
    (mailbox.testReceivedAt !== null && mailbox.testReceivedAt >= since)
  );
}
