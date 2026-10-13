import type { CheckCallRow } from "@/lib/graphql/check-calls";
import type { MoveStatus } from "@trenova/shared/types/shipment";

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

// Newest report first; two reports for the same moment keep the order they
// arrived in, newest received first.
export function sortCheckCalls(calls: readonly CheckCallRow[]): CheckCallRow[] {
  return [...calls].sort(
    (a, b) =>
      b.reportedAt - a.reportedAt || b.receivedAt - a.receivedAt || b.id.localeCompare(a.id),
  );
}
