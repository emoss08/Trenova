import { previewWarnings } from "@/lib/formula-preview";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import {
  AlertTriangleIcon,
  Beaker02Icon,
  CheckCircleIcon,
  Truck01Icon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import type { ReactNode } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import {
  formatMoney,
  formatStudioValue,
  numericAmount,
  previewAmounts,
  roundingSentence,
} from "./model/format";
import { readinessGoal } from "./model/readiness";
import { useStudio, type StudioSection } from "./studio-context";

type MarkerState = "ok" | "warn" | "bad" | "todo";

function Marker({ state, current }: { state: MarkerState; current: boolean }) {
  return (
    <span
      aria-hidden
      className={cn(
        "bg-canvas relative z-10 flex size-2.75 shrink-0 items-center justify-center rounded-full border",
        state === "bad" ? "border-danger" : "border-border-strong",
        current && "border-foreground ring-foreground/15 ring-2",
      )}
    >
      {state !== "todo" && (
        <span
          className={cn(
            "size-1.25 rounded-full",
            state === "ok" && "bg-success",
            state === "warn" && "bg-warning",
            state === "bad" && "bg-danger fs-breathe-once",
          )}
        />
      )}
    </span>
  );
}

function Step({
  state,
  section,
  title,
  meta,
  tag,
  children,
}: {
  state: MarkerState;
  section: StudioSection;
  title: string;
  meta?: ReactNode;
  tag?: ReactNode;
  children?: ReactNode;
}) {
  const { activeSection, jumpTo } = useStudio();
  const current =
    activeSection === section || (section === "expression" && activeSection === "details");
  return (
    <li className="before:bg-border relative flex flex-col gap-1.5 pb-4 before:absolute before:top-4 before:bottom-0 before:left-1.25 before:w-px last:pb-3 last:before:hidden">
      <button
        type="button"
        onClick={() => jumpTo(section)}
        aria-current={current ? "location" : undefined}
        className="ui-focus-ring hover:bg-surface-hover -mx-1.5 flex items-center gap-2 rounded-md px-1.5 py-0.5 text-left"
      >
        <Marker state={state} current={current} />
        <span className={cn("text-sm", current ? "font-semibold" : "font-medium")}>{title}</span>
        {tag}
        <span className="flex-1" />
        {meta && <span className="text-foreground-subtle font-mono text-2xs">{meta}</span>}
      </button>
      {children && <div className="flex flex-col gap-1.5 pl-5">{children}</div>}
    </li>
  );
}

function Note({ tone, children }: { tone?: "warn" | "bad"; children: ReactNode }) {
  return (
    <p
      className={cn(
        "text-xs leading-snug",
        tone === "warn" ? "text-warning" : tone === "bad" ? "text-danger" : "text-foreground-muted",
      )}
    >
      {children}
    </p>
  );
}

/**
 * A live column that follows the preview's load through the template, top to
 * bottom: what the formula reads, what it comes to, what the charge policy
 * does to it, how the breakdown splits it, and what is billed. Every amount
 * is the preview's. Below it, the scenarios and what review is waiting on.
 */
export function RateTrace({ className }: { className?: string }) {
  const t = useT();
  const studio = useStudio();
  const { preview, analysis, scenarios, readiness, reconciliation, mode, template, setTab } =
    studio;
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const [
    variableDefinitions,
    breakdownDefinitions,
    minCharge,
    maxCharge,
    roundingMode,
    roundingPrecision,
  ] = useWatch({
    control,
    name: [
      "variableDefinitions",
      "breakdownDefinitions",
      "minCharge",
      "maxCharge",
      "roundingMode",
      "roundingPrecision",
    ],
  });

  const result = preview.result;
  const amounts = previewAmounts(result);
  const ok = amounts !== null;
  const empty = analysis.empty;
  const runtimeError = result && !result.valid;

  const reads = ok
    ? (result?.receipt?.variables ?? []).map((variable) => ({
        name: variable.name,
        value: variable.value,
      }))
    : analysis.reads.map((name) => ({ name, value: preview.valueOf(name) }));
  const unused = (variableDefinitions ?? []).filter(
    (variable) => variable.name && !analysis.pending && !analysis.inputReads.has(variable.name),
  ).length;
  const statementCount = empty ? 0 : analysis.statements.length;
  const edited = Object.keys(preview.overrides).length;

  const usingShipment = preview.source === "shipment" && preview.shipmentId !== "";
  const proNumber = usingShipment ? preview.valueOf("proNumber") : null;
  const origin = usingShipment ? preview.valueOf("origin.city") : null;
  const destination = usingShipment ? preview.valueOf("destination.city") : null;
  const loadLabel = usingShipment
    ? [
        proNumber ? formatStudioValue(proNumber) : null,
        origin && destination
          ? `${formatStudioValue(origin)} → ${formatStudioValue(destination)}`
          : null,
      ]
        .filter(Boolean)
        .join(" · ") || t("Picked shipment")
    : t(
        "Sample load · {0} mi · {1, plural, one {# stop} other {# stops}}",
        formatStudioValue(preview.valueOf("totalDistance")),
        numericAmount(preview.valueOf("totalStops")) ?? 0,
      );

  const formulaState: MarkerState = empty
    ? "todo"
    : analysis.errorCount > 0 || runtimeError
      ? "bad"
      : analysis.warningCount > 0 || previewWarnings(result).length > 0
        ? "warn"
        : "ok";

  const min = numericAmount(minCharge);
  const max = numericAmount(maxCharge);
  const lines = ok
    ? (result?.breakdown ?? []).filter((line) => !line.error && line.amount > 0)
    : [];
  const lineSum = lines.reduce((sum, line) => sum + line.amount, 0);
  const breakdownCount = (breakdownDefinitions ?? []).length;

  const total = scenarios.total;
  const passed = scenarios.passed;
  const checks = readiness;
  const open = checks.filter((check) => check.status !== "pass");
  const liveState = ok ? "live" : empty ? "waiting" : "stopped";

  return (
    <ScrollArea
      maskVariant="background"
      className={cn("bg-canvas", className)}
      viewportClassName="flex flex-col"
      aria-label={t("Rate trace")}
    >
      <nav aria-label={t("Rate trace")} className="flex min-h-full flex-col">
        <div className="flex flex-col gap-2 px-4 pt-4 pb-3">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold">{t("Rate trace")}</span>
            <span
              className={cn(
                "flex items-center gap-1 text-2xs",
                liveState === "live" ? "text-success" : "text-foreground-subtle",
              )}
            >
              <span
                key={preview.lastRunAt ?? 0}
                className={cn(
                  "size-1.5 rounded-full",
                  liveState === "live" ? "bg-success fs-breathe-once" : "bg-foreground-subtle",
                )}
              />
              {liveState === "live"
                ? t("live")
                : liveState === "waiting"
                  ? t("waiting")
                  : t("stopped")}
            </span>
          </div>
          <button
            type="button"
            onClick={() => setTab("preview")}
            title={t("Change the load in Preview")}
            className="ui-focus-ring hover:bg-surface-hover text-foreground-muted -mx-1.5 flex min-w-0 items-center gap-1.5 rounded-md px-1.5 py-1 text-left text-xs"
          >
            <Truck01Icon className="size-3.5 shrink-0" />
            <span className="min-w-0 flex-1 truncate">{loadLabel}</span>
            {edited > 0 && (
              <span className="bg-warning-subtle text-warning-subtle-foreground shrink-0 rounded-badge px-1.5 text-2xs">
                {t("{0} edited", edited)}
              </span>
            )}
          </button>
        </div>

        <ol className="flex flex-col px-4">
          <Step
            state={unused > 0 ? "warn" : reads.length > 0 ? "ok" : "todo"}
            section="inputs"
            title={t("Reads")}
            meta={
              reads.length > 0
                ? t("{0, plural, one {# value} other {# values}}", reads.length)
                : t("nothing yet")
            }
          >
            {reads.length > 0 && (
              <div className="flex flex-col gap-0.5">
                {reads.slice(0, 4).map((read) => (
                  <div key={read.name} className="flex items-center justify-between gap-2 text-xs">
                    <code className="text-foreground-muted truncate font-mono">{read.name}</code>
                    <span className="shrink-0 font-mono tabular-nums">
                      {formatStudioValue(read.value)}
                    </span>
                  </div>
                ))}
                {reads.length > 4 && (
                  <button
                    type="button"
                    onClick={() => setTab("preview")}
                    className="ui-focus-ring text-foreground-subtle hover:text-foreground self-start rounded-sm text-2xs"
                  >
                    {t("+{0} more in Preview", reads.length - 4)}
                  </button>
                )}
              </div>
            )}
            {unused > 0 && (
              <Note tone="warn">
                {t(
                  "{0, plural, one {# input defined but never read} other {# inputs defined but never read}}",
                  unused,
                )}
              </Note>
            )}
          </Step>

          <Step
            state={formulaState}
            section="expression"
            title={t("Formula")}
            meta={
              empty ? t("empty") : t("{0, plural, one {# line} other {# lines}}", statementCount)
            }
          >
            {ok && amounts ? (
              <div key={amounts.raw} className="fs-settle font-mono text-lg tabular-nums">
                {formatMoney(amounts.raw)}
              </div>
            ) : empty ? (
              <Note>{t("Write or draft a formula to start the trace.")}</Note>
            ) : (
              <div className="text-foreground-subtle font-mono text-lg tabular-nums">—</div>
            )}
          </Step>

          <Step
            state={!ok ? "todo" : amounts?.bound ? "warn" : "ok"}
            section="policy"
            title={t("Charge policy")}
            tag={
              ok && amounts?.bound ? (
                <span className="bg-warning-subtle text-warning-subtle-foreground rounded-badge px-1.5 text-2xs">
                  {amounts.bound === "min" ? t("minimum") : t("maximum")}
                </span>
              ) : undefined
            }
          >
            <Note>
              {ok && amounts?.bound && amounts.limit !== null ? (
                amounts.bound === "min" ? (
                  <>
                    {t("Raised to the")}{" "}
                    <b className="text-foreground font-medium">{formatMoney(amounts.limit)}</b>{" "}
                    {t("minimum")}
                  </>
                ) : (
                  <>
                    {t("Lowered to the")}{" "}
                    <b className="text-foreground font-medium">{formatMoney(amounts.limit)}</b>{" "}
                    {t("maximum")}
                  </>
                )
              ) : min !== null || max !== null ? (
                t(
                  "Inside {0} – {1}",
                  min !== null ? formatMoney(min) : t("no minimum"),
                  max !== null ? formatMoney(max) : t("no maximum"),
                )
              ) : (
                t("No minimum or maximum")
              )}
              <br />
              {roundingSentence(roundingMode, roundingPrecision ?? 2, t)}
            </Note>
          </Step>

          {breakdownCount > 0 && (
            <Step
              state={
                !ok
                  ? "todo"
                  : reconciliation && reconciliation.balanced && !reconciliation.clampMismatch
                    ? "ok"
                    : "warn"
              }
              section="breakdown"
              title={t("Breakdown")}
              meta={t("{0, plural, one {# line} other {# lines}}", breakdownCount)}
            >
              {lines.length > 0 && (
                <div className="flex flex-col gap-0.5">
                  {lines.slice(0, 4).map((line) => (
                    <div
                      key={line.name}
                      className="flex items-center justify-between gap-2 text-xs"
                      title={formatMoney(line.amount)}
                    >
                      <span className="text-foreground-muted min-w-0 flex-1 truncate">
                        {line.label || line.name}
                      </span>
                      <span className="font-mono tabular-nums">
                        {lineSum > 0 ? Math.round((line.amount / lineSum) * 100) : 0}%
                      </span>
                    </div>
                  ))}
                </div>
              )}
              {ok && reconciliation && !reconciliation.balanced && (
                <Note tone="warn">{t("Lines don't add up to the formula")}</Note>
              )}
            </Step>
          )}
        </ol>

        <div className="px-4 pb-4">
          <button
            type="button"
            onClick={() => setTab("preview")}
            className={cn(
              "ui-focus-ring bg-card hover:border-border-strong flex w-full flex-col gap-0.5 rounded-lg border px-3 py-2.5 text-left transition-colors",
              !ok && "opacity-70",
            )}
          >
            <span className="text-foreground-muted text-xs">{t("Billed")}</span>
            <span
              key={amounts?.billed ?? "none"}
              className="fs-settle font-mono text-3xl tabular-nums"
            >
              {ok && amounts ? formatMoney(amounts.billed) : "—"}
            </span>
            <span className="text-foreground-subtle text-2xs">
              {!ok
                ? t("nothing to bill yet")
                : usingShipment
                  ? t("on {0}", proNumber ? formatStudioValue(proNumber) : t("the picked shipment"))
                  : t("on the sample load")}
            </span>
          </button>
        </div>

        <span className="flex-1" />

        <div className="px-4 pb-3">
          <button
            type="button"
            onClick={() => setTab("scenarios")}
            className={cn(
              "ui-focus-ring flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-xs",
              total > 0 && passed < total
                ? "bg-danger-subtle text-danger-subtle-foreground"
                : "bg-sunken text-foreground-muted",
            )}
          >
            <Beaker02Icon className="size-3.5 shrink-0" />
            <span className="min-w-0 flex-1 truncate">
              {total === 0
                ? t("No scenarios yet")
                : passed === total
                  ? t("Scenarios hold")
                  : t(
                      "{0, plural, one {# scenario broke} other {# scenarios broke}}",
                      total - passed,
                    )}
            </span>
            {total > 0 && (
              <span className="font-mono tabular-nums">
                {passed}/{total}
              </span>
            )}
          </button>
        </div>

        <div className="flex flex-col gap-2 border-t px-4 py-3">
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-semibold">
              {readinessGoal(mode, template?.status, t)}
            </span>
            <span className="text-foreground-subtle font-mono text-2xs tabular-nums">
              {checks.length - open.length}/{checks.length}
            </span>
          </div>
          {open.length === 0 ? (
            <p className="text-success flex items-center gap-1.5 text-xs">
              <CheckCircleIcon className="size-3.5" />
              {t("Every check passes")}
            </p>
          ) : (
            <ul className="flex flex-col gap-1">
              {open.map((check) => (
                <li key={check.key} className="flex items-start gap-1.5 text-xs">
                  {check.status === "warn" ? (
                    <AlertTriangleIcon className="text-warning mt-px size-3.5 shrink-0" />
                  ) : (
                    <XCircleIcon className="text-danger mt-px size-3.5 shrink-0" />
                  )}
                  <span className="flex min-w-0 flex-col">
                    <span className="font-medium">{check.label}</span>
                    {check.detail && <span className="text-foreground-subtle">{check.detail}</span>}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </nav>
    </ScrollArea>
  );
}
