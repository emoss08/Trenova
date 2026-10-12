import type { TriageBriefing, TriageUser } from "@/lib/graphql/inbox";

type DeckCard = TriageBriefing["deck"][number];
export type DeckGroup = DeckCard["group"];
export type DeckItem = DeckCard["items"][number];
export type RiverPoint = TriageBriefing["yours"][number];

/**
 * The deck's order, whatever order the server sends: the quick wins first
 * (filing, then Nova's drafts), then what needs thought, then what needs a
 * record, and last what was held back as unsafe.
 */
export const DECK_ORDER: readonly DeckGroup[] = ["File", "Draft", "Decide", "Link", "Held"];

/** One card of the deck as drawn: the server's card less what was already acted on here. */
export type DeckEntry = {
  group: DeckGroup;
  count: number;
  estimateMinutes: number;
  items: DeckItem[];
  messageIds: string[];
};

/**
 * The cards to draw. A card's "File all" or "Draft all" takes its messages
 * out of the deck the moment it is pressed, before the server's next
 * briefing says where they went, so `acted` holds them out; a card left with
 * nothing is not drawn. The estimate shrinks with what is left.
 */
export function deckEntries(deck: readonly DeckCard[], acted: ReadonlySet<string>): DeckEntry[] {
  const entries: DeckEntry[] = [];
  for (const group of DECK_ORDER) {
    const card = deck.find((row) => row.group === group);
    if (card === undefined) {
      continue;
    }
    const messageIds = card.messageIds.filter((id) => !acted.has(id));
    if (messageIds.length === 0) {
      continue;
    }
    const total = Math.max(card.messageIds.length, card.count, 1);
    const scaled = Math.round((card.estimateMinutes * messageIds.length) / total);
    entries.push({
      group,
      count: messageIds.length,
      estimateMinutes: card.estimateMinutes > 0 ? Math.max(1, scaled) : 0,
      items: card.items.filter((row) => !acted.has(row.messageId)),
      messageIds,
    });
  }

  return entries;
}

/**
 * How far through the day's work you are: what cleared against what was
 * yours today. Nothing waiting is a full ring, however the day went.
 */
export function ringProgress(cleared: number, waiting: number): number {
  if (waiting <= 0) {
    return 1;
  }
  const done = Math.max(0, cleared);

  return done / (done + waiting);
}

const HOUR = 3600;
/** Where the river starts its marks: six in the morning, the organization's time. */
const RIVER_FIRST_HOUR = 6;
/** The least the river shows, so a message early in the day still has room. */
const RIVER_MIN_SPAN = 2 * HOUR;
/** Room left of the first mark, as a share of the day shown. */
const RIVER_LEAD = 0.13;
/** A mark this close to now is left out rather than drawn over "now". */
const RIVER_CROWD = 0.06;
const X_MIN = 2;
const X_MAX = 98;

export type RiverWindow = { start: number; end: number };

/** From a little before six in the morning (or two hours back, if earlier) to now. */
export function riverWindow(dayStart: number, now: number): RiverWindow {
  const first = Math.min(dayStart + RIVER_FIRST_HOUR * HOUR, now - RIVER_MIN_SPAN);
  const lead = (now - first) * RIVER_LEAD;

  return { start: first - lead, end: now };
}

/** Where a moment sits on the river, as a percentage across; earlier mail sits at the edge. */
export function riverX(at: number, window: RiverWindow): number {
  const span = window.end - window.start;
  if (span <= 0) {
    return X_MAX;
  }
  const share = (at - window.start) / span;

  return Math.min(X_MAX, Math.max(X_MIN, X_MIN + (X_MAX - X_MIN) * share));
}

export type RiverTick = { at: number; x: number };

/** A mark every two hours from six, inside the window, none crowding "now". */
export function riverTicks(window: RiverWindow, dayStart: number): RiverTick[] {
  const span = window.end - window.start;
  const ticks: RiverTick[] = [];
  for (
    let at = dayStart + RIVER_FIRST_HOUR * HOUR;
    at < window.end - span * RIVER_CROWD;
    at += 2 * HOUR
  ) {
    if (at >= window.start) {
      ticks.push({ at, x: riverX(at, window) });
    }
  }

  return ticks;
}

/** Which way a tooltip opens from its dot so it stays inside the river. */
export function tipAlign(x: number): "start" | "center" | "end" {
  if (x < 18) {
    return "start";
  }

  return x > 82 ? "end" : "center";
}

/** A dot in your lane: done (settled or parked), over its target, or still open. */
export function yourDotState(point: RiverPoint, now: number): "done" | "over" | "open" {
  if (point.done) {
    return "done";
  }

  return point.dueAt !== null && point.dueAt < now ? "over" : "open";
}

export type NovaLineInput = Pick<
  TriageBriefing,
  | "phase"
  | "waiting"
  | "handledByNovaToday"
  | "closedByPeopleToday"
  | "estimateMinutes"
  | "longestWaiting"
  | "carryOver"
> & { onCall: TriageUser | null };

/** What Nova says at the top of the briefing, before it is put into words. */
export type NovaLine =
  | { kind: "clear" }
  | {
      kind: "day";
      handled: number;
      waiting: number;
      longest: DeckItem | null;
      minutes: number;
    }
  | {
      kind: "endOfDay";
      handled: number;
      closed: number;
      carry: number;
      onCall: TriageUser | null;
    };

export function novaLine(briefing: NovaLineInput): NovaLine {
  if (briefing.phase === "EndOfDay") {
    return {
      kind: "endOfDay",
      handled: briefing.handledByNovaToday,
      closed: briefing.closedByPeopleToday,
      carry: briefing.carryOver.length,
      onCall: briefing.onCall,
    };
  }
  if (briefing.waiting <= 0) {
    return { kind: "clear" };
  }

  return {
    kind: "day",
    handled: briefing.handledByNovaToday,
    waiting: briefing.waiting,
    longest: briefing.longestWaiting,
    minutes: briefing.estimateMinutes,
  };
}

/** What carries over to tomorrow, soonest due first; anything without a target last. */
export function carryOverRows(items: readonly DeckItem[]): DeckItem[] {
  return [...items].sort((a, b) => {
    if (a.dueAt === b.dueAt) {
      return 0;
    }
    if (a.dueAt === null) {
      return 1;
    }
    if (b.dueAt === null) {
      return -1;
    }
    return a.dueAt - b.dueAt;
  });
}

/** How long after connecting a source with nothing waiting the first day stops saying it is reading. */
const QUIET_AFTER_SECONDS = 20;

export type FirstDayStage = "empty" | "reading" | "quiet" | "done" | "briefing";

export type FirstDayInput = {
  firstDay: boolean;
  /** When a source was connected from this page, if one was. */
  connectedAt: number | null;
  counts: { total: number; processing: number };
  now: number;
  /** The "done reading" beat has been shown. */
  finished: boolean;
};

/**
 * Where the first day stands: nothing connected yet; the backlog being read
 * after a source was connected here (with a live count by kind); nothing was
 * waiting at the address; done reading; then the ordinary briefing.
 */
export function firstDayStage({
  firstDay,
  connectedAt,
  counts,
  now,
  finished,
}: FirstDayInput): FirstDayStage {
  if (connectedAt === null) {
    if (!firstDay) {
      return "briefing";
    }
    return counts.processing > 0 ? "reading" : "empty";
  }
  if (counts.processing > 0) {
    return "reading";
  }
  if (counts.total === 0) {
    return now - connectedAt >= QUIET_AFTER_SECONDS ? "quiet" : "reading";
  }

  return finished ? "briefing" : "done";
}
