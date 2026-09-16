import type { FiscalYearStatus } from "@/types/fiscal-year";
import { Operation, type OperationType } from "@trenova/shared/types/permission";

export type FiscalYearAction = "activate" | "close" | "reopen";

export const fiscalYearActionOperation: Record<FiscalYearAction, OperationType> = {
  activate: Operation.Activate,
  close: Operation.Close,
  reopen: Operation.Reopen,
};

type FiscalYearActionInput = {
  status: FiscalYearStatus | undefined;
  isCurrent: boolean | undefined;
  can: (operation: OperationType) => boolean;
};

function isAvailable(action: FiscalYearAction, { status, isCurrent }: FiscalYearActionInput) {
  switch (action) {
    case "activate":
      return !isCurrent && (status === "Draft" || status === "Open");
    case "close":
      return status === "Open";
    case "reopen":
      return status === "Closed";
  }
}

const ACTION_ORDER: readonly FiscalYearAction[] = ["activate", "close", "reopen"];

export function getFiscalYearActions(input: FiscalYearActionInput): FiscalYearAction[] {
  return ACTION_ORDER.filter(
    (action) => isAvailable(action, input) && input.can(fiscalYearActionOperation[action]),
  );
}
