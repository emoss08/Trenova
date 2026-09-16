import type { FiscalPeriod } from "@/types/fiscal-period";
import type { FiscalYearStatus } from "@/types/fiscal-year";
import { Operation, type OperationType } from "@trenova/shared/types/permission";

export type FiscalPeriodAction = "activate" | "lock" | "unlock" | "close" | "reopen";

export type FiscalPeriodActionBlocker =
  | { kind: "fiscalYearClosed" }
  | { kind: "fiscalYearPermanentlyClosed" }
  | { kind: "earlierPeriodNotOpened"; periodNumber: number }
  | { kind: "earlierPeriodOpen"; periodNumber: number }
  | { kind: "laterPeriodClosed"; periodNumber: number };

export type FiscalPeriodActionOption = {
  action: FiscalPeriodAction;
  blocker: FiscalPeriodActionBlocker | null;
};

type PeriodRef = Pick<FiscalPeriod, "id" | "periodNumber" | "status">;

type FiscalPeriodActionInput = {
  period: PeriodRef;
  periods: readonly PeriodRef[];
  fiscalYearStatus: FiscalYearStatus | undefined;
  can: (operation: OperationType) => boolean;
};

export const fiscalPeriodActionOperation: Record<FiscalPeriodAction, OperationType> = {
  activate: Operation.Activate,
  lock: Operation.Lock,
  unlock: Operation.Unlock,
  close: Operation.Close,
  reopen: Operation.Reopen,
};

const actionsByStatus: Record<FiscalPeriod["status"], readonly FiscalPeriodAction[]> = {
  Inactive: ["activate"],
  Open: ["lock", "close"],
  Locked: ["unlock", "close"],
  Closed: ["reopen"],
  PermanentlyClosed: [],
};

function firstPeriod(
  periods: readonly PeriodRef[],
  predicate: (candidate: PeriodRef) => boolean,
): PeriodRef | undefined {
  let match: PeriodRef | undefined;
  for (const candidate of periods) {
    if (predicate(candidate) && (!match || candidate.periodNumber < match.periodNumber)) {
      match = candidate;
    }
  }
  return match;
}

function blockerFor(
  action: FiscalPeriodAction,
  { period, periods, fiscalYearStatus }: FiscalPeriodActionInput,
): FiscalPeriodActionBlocker | null {
  const loosens = action === "activate" || action === "unlock" || action === "reopen";
  if (loosens && fiscalYearStatus === "PermanentlyClosed") {
    return { kind: "fiscalYearPermanentlyClosed" };
  }
  if (loosens && fiscalYearStatus === "Closed") {
    return { kind: "fiscalYearClosed" };
  }

  const others = periods.filter((candidate) => candidate.id !== period.id);

  if (action === "activate") {
    const earlier = firstPeriod(
      others,
      (c) => c.periodNumber < period.periodNumber && c.status === "Inactive",
    );
    return earlier ? { kind: "earlierPeriodNotOpened", periodNumber: earlier.periodNumber } : null;
  }

  if (action === "close") {
    const earlier = firstPeriod(
      others,
      (c) =>
        c.periodNumber < period.periodNumber && (c.status === "Open" || c.status === "Inactive"),
    );
    if (!earlier) return null;
    return earlier.status === "Inactive"
      ? { kind: "earlierPeriodNotOpened", periodNumber: earlier.periodNumber }
      : { kind: "earlierPeriodOpen", periodNumber: earlier.periodNumber };
  }

  if (action === "reopen") {
    const later = firstPeriod(
      others,
      (c) =>
        c.periodNumber > period.periodNumber && (c.status === "Closed" || c.status === "Locked"),
    );
    return later ? { kind: "laterPeriodClosed", periodNumber: later.periodNumber } : null;
  }

  return null;
}

export function getFiscalPeriodActions(input: FiscalPeriodActionInput): FiscalPeriodActionOption[] {
  return actionsByStatus[input.period.status]
    .filter((action) => input.can(fiscalPeriodActionOperation[action]))
    .map((action) => ({ action, blocker: blockerFor(action, input) }));
}
