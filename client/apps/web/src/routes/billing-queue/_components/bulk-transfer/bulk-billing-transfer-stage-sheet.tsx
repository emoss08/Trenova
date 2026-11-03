import { CheckIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";
import type { ReadinessStep } from "./bulk-billing-transfer-report";

export type SheetFieldState = "idle" | "active" | "pass" | "fail";

export type SheetMarks = Partial<Record<ReadinessStep, SheetFieldState>>;

/** The bill's drawn size, in pixels. It is only ever scaled down, so it stays sharp at rest. */
export const SHEET_WIDTH = 80;
export const SHEET_HEIGHT = 104;

/**
 * A freight bill as the transfer stage draws it: a header, then one section for each step of
 * the readiness check, in the order the desk reads them. A section's `data-state` marks it
 * active, passed or failed, so the stage engine can drive a cloned bill by attribute alone.
 * Nothing on it is text: the run reports counts, not the shipments behind them, so the bill
 * stands for a shipment without pretending to be a particular one.
 */
export function TransferStageSheet({
  marks,
  stamped = false,
  className,
  style,
}: {
  marks?: SheetMarks;
  stamped?: boolean;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <div
      aria-hidden
      data-sheet=""
      data-stamped={stamped ? "true" : "false"}
      className={cn(
        "group/sheet bg-raised border-border-strong relative flex h-26 w-20 flex-col gap-[7px] overflow-hidden rounded-[5px] border px-2 pt-2.5 pb-2",
        "[clip-path:polygon(0_0,calc(100%-14px)_0,100%_14px,100%_100%,0_100%)]",
        className,
      )}
      style={style}
    >
      <span className="bg-border-strong absolute top-0 right-0 size-3.5 [clip-path:polygon(0_0,100%_100%,0_100%)]" />
      <span className="flex h-[11px] items-center gap-1 pr-2.5">
        <span className="bg-foreground-subtle h-[3px] w-8 rounded-full" />
        <span className="bg-foreground-subtle ml-auto h-[3px] w-4 rounded-full opacity-0 transition-opacity duration-300 group-data-[stamped=true]/sheet:opacity-100" />
      </span>
      <SheetField step="status" state={marks?.status}>
        <span className="bg-border h-[3px] w-[38%] rounded-full" />
        <span className="bg-border h-1.5 w-3.5 rounded-full" />
      </SheetField>
      <SheetField step="documents" state={marks?.documents}>
        <span className="border-border-strong h-[11px] w-[9px] rounded-[1.5px] border" />
        <span className="border-border-strong h-[11px] w-[9px] rounded-[1.5px] border" />
        <span className="border-border-strong h-[11px] w-[9px] rounded-[1.5px] border" />
      </SheetField>
      <SheetField step="rate" state={marks?.rate}>
        <span className="bg-border h-[3px] w-[22%] rounded-full" />
        <span className="bg-border-strong h-[3px] w-[30%] rounded-full" />
      </SheetField>
      <SheetField step="billed" state={marks?.billed}>
        <span className="bg-border h-[3px] w-[52%] rounded-full" />
      </SheetField>
      <span
        data-stamp=""
        className="bg-success text-foreground-on-solid absolute right-[7px] bottom-[7px] grid size-4 scale-50 -rotate-12 place-items-center rounded-full opacity-0 transition-[opacity,transform] duration-300 ease-spring group-data-[stamped=true]/sheet:scale-100 group-data-[stamped=true]/sheet:rotate-0 group-data-[stamped=true]/sheet:opacity-100"
      >
        <CheckIcon className="size-2.5" strokeWidth={3} />
      </span>
    </div>
  );
}

function SheetField({
  step,
  state = "idle",
  children,
}: {
  step: ReadinessStep;
  state?: SheetFieldState;
  children: ReactNode;
}) {
  return (
    <span
      data-field={step}
      data-state={state}
      className="group/field data-[state=active]:bg-surface-hover relative flex h-[13px] items-center gap-1 overflow-hidden rounded-[3px] px-0.5 transition-colors duration-200"
    >
      {children}
      <span
        data-sweep=""
        className="via-foreground/15 pointer-events-none absolute inset-y-0 left-0 w-3/5 bg-linear-to-r from-transparent to-transparent opacity-0"
      />
      <span className="ml-auto grid size-[9px] shrink-0 scale-50 place-items-center rounded-full opacity-0 transition-[opacity,transform] duration-300 ease-spring group-data-[state=fail]/field:bg-warning group-data-[state=fail]/field:scale-100 group-data-[state=fail]/field:opacity-100 group-data-[state=pass]/field:bg-success group-data-[state=pass]/field:scale-100 group-data-[state=pass]/field:opacity-100">
        <CheckIcon
          className="text-foreground-on-solid size-1.5 group-data-[state=fail]/field:hidden"
          strokeWidth={3.5}
        />
        <span className="bg-warning-on-solid hidden h-[1.5px] w-[5px] rounded-full group-data-[state=fail]/field:block" />
      </span>
      <span className="bg-warning absolute inset-x-0.5 bottom-0 hidden h-[1.5px] rounded-full group-data-[state=fail]/field:block" />
    </span>
  );
}

/** Which sections a bill passed and where it stopped, read from its failure reason's step. */
export function sheetMarksFor(failedAt: ReadinessStep | null, steps: readonly ReadinessStep[]) {
  const marks: SheetMarks = {};
  for (const step of steps) {
    if (step === failedAt) {
      marks[step] = "fail";
      break;
    }
    marks[step] = "pass";
  }
  return marks;
}
