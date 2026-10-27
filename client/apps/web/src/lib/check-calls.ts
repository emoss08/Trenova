import type { CheckCallRow } from "@/lib/graphql/check-calls";
import { getStartOfDay } from "@trenova/shared/lib/date";
import type { MoveStatus, Shipment } from "@trenova/shared/types/shipment";

export function checkCallPlace(call: Pick<CheckCallRow, "city" | "stateCode">): string {
  return [call.city, call.stateCode].filter((part) => !!part && part.trim() !== "").join(", ");
}

export type LoggableMove = {
  id: string;
  label: string;
};

type MoveLike = {
  id?: string | null;
  status?: MoveStatus | null;
  sequence?: number | null;
};

// A canceled move takes no check calls; every other saved move does, including
// a completed one a dispatcher is filling in after the fact.
export function loggableMoves(
  moves: readonly MoveLike[],
  label: (move: MoveLike, index: number) => string,
): LoggableMove[] {
  const result: LoggableMove[] = [];
  moves.forEach((move, index) => {
    if (!move.id || move.status === "Canceled") {
      return;
    }
    result.push({ id: move.id, label: label(move, index) });
  });
  return result;
}

// The move a new check call lands on unless the dispatcher picks another: the
// one the truck is driving, else the first that takes calls at all.
export function defaultCheckCallMoveId(
  loggable: readonly LoggableMove[],
  moves: readonly MoveLike[],
): string {
  const inTransit = loggable.find(
    (move) => moves.find((candidate) => candidate.id === move.id)?.status === "InTransit",
  );
  return inTransit?.id ?? loggable[0]?.id ?? "";
}

// Newest report first; two reports for the same moment keep the order they
// arrived in, newest received first.
export function sortCheckCalls(calls: readonly CheckCallRow[]): CheckCallRow[] {
  return [...calls].sort(
    (a, b) =>
      b.reportedAt - a.reportedAt || b.receivedAt - a.receivedAt || b.id.localeCompare(a.id),
  );
}

export type CheckCallDay = {
  /** The instant the day starts in the timezone the calls were grouped in. */
  day: number;
  calls: CheckCallRow[];
};

// Sorted calls cut into the calendar days they were reported on, in the
// person's timezone, so a call at 11pm in Denver is not filed under tomorrow.
export function groupCheckCallsByDay(
  calls: readonly CheckCallRow[],
  timezone?: string,
): CheckCallDay[] {
  const days: CheckCallDay[] = [];
  for (const call of calls) {
    const day = getStartOfDay(new Date(call.reportedAt * 1000), timezone);
    const last = days.at(-1);
    if (last?.day === day) {
      last.calls.push(call);
    } else {
      days.push({ day, calls: [call] });
    }
  }
  return days;
}

export type RelativeDay = "today" | "yesterday" | "earlier";

export function relativeDay(day: number, now: Date, timezone?: string): RelativeDay {
  const today = getStartOfDay(now, timezone);
  if (day === today) {
    return "today";
  }
  // The day before today starts wherever the second before midnight falls,
  // which is not always 24 hours earlier across a clock change.
  if (day === getStartOfDay(new Date((today - 1) * 1000), timezone)) {
    return "yesterday";
  }
  return "earlier";
}

export type StopTimesFreeze = "invoiced" | "billing-queue";

// Why the server will refuse a check call's change to a stop's actual times,
// or null when it will apply it. Mirrors trackingeventservice's
// projectionOptions: an invoiced load, then any load billing still holds.
export function stopTimesFreeze(
  shipment: Pick<Shipment, "status" | "billingTransferStatus">,
): StopTimesFreeze | null {
  if (shipment.status === "Invoiced") {
    return "invoiced";
  }
  const transfer = shipment.billingTransferStatus;
  if (transfer && transfer !== "SentBackToOps") {
    return "billing-queue";
  }
  return null;
}
