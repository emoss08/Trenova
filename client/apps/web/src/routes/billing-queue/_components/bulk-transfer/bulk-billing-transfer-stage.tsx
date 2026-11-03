import type {
  BillingTransferFailureCode,
  BillingTransferRun,
} from "@/lib/graphql/billing-transfer";
import { CheckIcon } from "@trenova/shared/components/icons";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotionConfig } from "motion/react";
import { useEffect, useRef, useState, type ReactNode, type Ref } from "react";
import {
  BILLING_TRANSFER_FAILURE_REASONS,
  READINESS_STEPS,
  type ReadinessStep,
} from "./bulk-billing-transfer-report";
import { isRunTerminal } from "./bulk-billing-transfer-run";
import {
  heldTotal,
  IDLE_STEPS,
  STACK_SCALE,
  stageCountsOf,
  TransferStageEngine,
  type StageCounts,
  type StageSteps,
} from "./bulk-billing-transfer-stage-engine";
import { sheetMarksFor, TransferStageSheet } from "./bulk-billing-transfer-stage-sheet";

const STACK_LAYERS = 6;

const COUNT = {
  b: (chunk: ReactNode) => <b className="text-foreground font-medium tabular-nums">{chunk}</b>,
};

/** Whether this browser can run the stage's moves; where it cannot, the stage holds still. */
const CAN_ANIMATE =
  typeof Element !== "undefined" && typeof Element.prototype.animate === "function";
const STEP_KEYS: readonly ReadinessStep[] = READINESS_STEPS.map((step) => step.key);

/**
 * A run while it works, drawn as paper: freight bills leave the delivered stack, are read
 * on the desk beside the readiness checklist, and land on the billing queue or the
 * needs-attention stack. It moves only as the run's own answers arrive and stops when the
 * run does. `onSettled` fires once a finished run's last bill has landed, which is the
 * dialog's cue to hand over to the results.
 *
 * The stage starts from the run's counts when it mounts, so reopening the dialog on a run
 * that is halfway through does not replay what already happened.
 */
export function BulkBillingTransferStage({
  run,
  onSettled,
}: {
  run: BillingTransferRun;
  onSettled: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const reduceMotion = (useReducedMotionConfig() ?? false) || !CAN_ANIMATE;
  const target = stageCountsOf(run);
  const terminal = isRunTerminal(run);
  const settle = useLatestCallback(onSettled);

  const [landed, setLanded] = useState<StageCounts>(target);
  const latestTarget = useRef(target);
  const [steps, setSteps] = useState<StageSteps>(IDLE_STEPS);
  const engineRef = useRef<TransferStageEngine | null>(null);

  const stageRef = useRef<HTMLDivElement>(null);
  const templateRef = useRef<HTMLDivElement>(null);
  const sourceRef = useRef<HTMLDivElement>(null);
  const deskRef = useRef<HTMLDivElement>(null);
  const billingRef = useRef<HTMLDivElement>(null);
  const attentionRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    latestTarget.current = target;
  });

  useEffect(() => {
    if (reduceMotion) return;

    const engine = new TransferStageEngine(
      {
        anchors: () => {
          const stage = stageRef.current;
          const template = templateRef.current;
          const source = sourceRef.current;
          const desk = deskRef.current;
          const billing = billingRef.current;
          const attention = attentionRef.current;
          if (!stage || !template || !source || !desk || !billing || !attention) return null;
          return { stage, template, source, desk, billing, attention };
        },
        onSteps: setSteps,
        onLanded: setLanded,
        onSettled: settle,
      },
      latestTarget.current,
    );
    engineRef.current = engine;

    return () => {
      engine.dispose();
      engineRef.current = null;
    };
  }, [reduceMotion, settle]);

  useEffect(() => {
    engineRef.current?.update(target, terminal);
  }, [target, terminal]);

  useEffect(() => {
    if (reduceMotion && terminal) settle();
  }, [reduceMotion, terminal, settle]);

  const shown = reduceMotion ? target : landed;
  const held = heldTotal(shown);
  const remaining = Math.max(0, run.totalCount - shown.transferred - held);
  const sized = run.totalCount > 0;

  return (
    <div
      ref={stageRef}
      className="bg-sunken relative h-75 overflow-hidden rounded-lg border [--side:min(40%,310px)] max-sm:[--side:38%]"
      style={{
        backgroundImage: "radial-gradient(circle at 1px 1px, var(--border) 1px, transparent 0)",
        backgroundSize: "16px 16px",
      }}
    >
      <span className="text-foreground-subtle absolute top-3.5 left-1/2 -translate-x-1/2 text-2xs">
        {t("Readiness check")}
      </span>

      <StagePile
        ref={sourceRef}
        className="top-[169px] left-[calc(50%-var(--side))]"
        count={remaining}
        total={run.totalCount}
        kind="source"
      >
        {sized
          ? rt("Delivered <b>{0}</b> left", COUNT, formatNumber(remaining))
          : t("Finding shipments")}
      </StagePile>

      <div className="absolute top-1/2 left-1/2 flex translate-x-[-40px] -translate-y-1/2 items-center gap-6">
        <div
          ref={deskRef}
          className="border-border-strong h-26 w-20 shrink-0 rounded-[5px] border border-dashed"
        />
        <ol aria-label={t("Readiness check")} className="flex w-32 flex-col gap-0.5 max-sm:w-auto">
          {READINESS_STEPS.map((step) => (
            <ReadinessStepRow key={step.key} label={t(step.label)} state={steps[step.key]} />
          ))}
        </ol>
      </div>

      <StagePile
        ref={billingRef}
        className="top-[92px] left-[calc(50%+var(--side))]"
        count={shown.transferred}
        total={run.totalCount}
        kind="billing"
        dot="bg-success"
      >
        {rt("Billing queue <b>{0}</b>", COUNT, formatNumber(shown.transferred))}
      </StagePile>
      <StagePile
        ref={attentionRef}
        className="top-[208px] left-[calc(50%+var(--side))]"
        count={held}
        total={run.totalCount}
        kind="attention"
        failedAt={shown.lastHeld}
        dot="bg-warning"
      >
        {rt("Needs attention <b>{0}</b>", COUNT, formatNumber(held))}
      </StagePile>

      <div ref={templateRef} hidden>
        <TransferStageSheet />
      </div>
    </div>
  );
}

type PileKind = "source" | "billing" | "attention";

function StagePile({
  ref,
  className,
  count,
  total,
  kind,
  failedAt = null,
  dot,
  children,
}: {
  ref: Ref<HTMLDivElement>;
  className: string;
  count: number;
  total: number;
  kind: PileKind;
  failedAt?: BillingTransferFailureCode | null;
  dot?: string;
  children: ReactNode;
}) {
  const layers = stackLayers(count, total, kind);
  const marks =
    kind === "source"
      ? undefined
      : sheetMarksFor(failedAt ? BILLING_TRANSFER_FAILURE_REASONS[failedAt].step : null, STEP_KEYS);

  return (
    <div
      className={cn(
        "absolute flex -translate-x-1/2 -translate-y-1/2 flex-col items-center gap-2.5 px-2.5 py-1.5",
        className,
      )}
    >
      <div ref={ref} className="relative h-[58px] w-11">
        {layers === 0 ? (
          <span className="border-border-strong absolute inset-0 rounded-[3px] border border-dashed" />
        ) : (
          Array.from({ length: layers }, (_, index) => {
            const depth = layers - 1 - index;
            const top = index === layers - 1;
            return (
              <TransferStageSheet
                key={index}
                className="absolute top-0 left-0 origin-top-left"
                style={{
                  transform: `translate(${depth * 1.5}px, ${depth * 2.5}px) rotate(${
                    (index % 2 ? 1 : -1) * Math.min(depth, 3) * 0.6
                  }deg) scale(${STACK_SCALE})`,
                }}
                marks={top ? marks : undefined}
                stamped={top && kind === "billing"}
              />
            );
          })
        )}
      </div>
      <span className="text-muted-foreground flex items-center gap-1.5 text-2xs whitespace-nowrap">
        {dot ? <span aria-hidden className={cn("size-1.5 rounded-full", dot)} /> : null}
        <span>{children}</span>
      </span>
    </div>
  );
}

/**
 * How thick a stack is drawn: up to six sheets. The delivered stack thins from full; the two
 * outcome stacks thicken faster than their share, so a few landed bills already read as a pile.
 */
function stackLayers(count: number, total: number, kind: PileKind): number {
  if (count <= 0) return 0;
  const share = count / Math.max(total, 1);
  const weight = kind === "source" ? 1 : 1.6;
  return Math.max(1, Math.min(STACK_LAYERS, Math.ceil(share * STACK_LAYERS * weight)));
}

function ReadinessStepRow({ label, state }: { label: string; state: StageSteps[ReadinessStep] }) {
  return (
    <li
      data-state={state}
      className={cn(
        "grid grid-cols-[12px_minmax(0,1fr)] items-center gap-2 rounded-md px-1.5 py-1 text-xs transition-colors duration-300",
        state === "idle" && "text-foreground-subtle",
        (state === "active" || state === "pass") && "text-foreground",
        state === "fail" && "text-warning font-medium",
      )}
    >
      <span
        aria-hidden
        className={cn(
          "grid size-3 place-items-center rounded-full border transition-[background-color,border-color,box-shadow] duration-300",
          state === "idle" && "border-foreground-subtle/50",
          state === "active" && "border-foreground shadow-[0_0_0_3px_var(--muted)]",
          state === "pass" && "bg-success border-success",
          state === "fail" && "bg-warning border-warning",
        )}
      >
        {state === "pass" ? (
          <CheckIcon className="text-foreground-on-solid size-2 animate-confirm" strokeWidth={3} />
        ) : state === "fail" ? (
          <span className="bg-warning-on-solid h-0.5 w-1.5 rounded-full" />
        ) : null}
      </span>
      <span className="truncate max-sm:sr-only">{label}</span>
    </li>
  );
}
