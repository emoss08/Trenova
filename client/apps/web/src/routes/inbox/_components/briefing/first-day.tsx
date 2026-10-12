import type { InboundMessageCounts } from "@/lib/graphql/inbox";
import { CheckIcon, PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  CLASSIFICATION_DOT,
  INBOUND_CLASSIFICATIONS,
  classificationLabel,
} from "../classification";
import type { FirstDayStage } from "./briefing-model";
import { useCountUp } from "./use-briefing-motion";

type Stage = Exclude<FirstDayStage, "briefing">;

const STEPS = ["source", "backlog", "briefing"] as const;

/** Which of the three steps a stage has reached: done ones before it, the current one, the rest. */
function stepIndex(stage: Stage): number {
  switch (stage) {
    case "empty":
      return 0;
    case "reading":
    case "quiet":
      return 1;
    case "done":
      return 3;
  }
}

/**
 * The briefing before anything has arrived: connect a source, watch Nova
 * read what was already sitting at the address (counted by kind as each
 * message is read), then the first briefing.
 */
export function FirstDay({
  stage,
  counts,
  canConnect,
  onConnect,
}: {
  stage: Stage;
  counts: InboundMessageCounts | undefined;
  /** Whether the person may add a source; without it the button says who can. */
  canConnect: boolean;
  onConnect: () => void;
}) {
  const t = useT();
  const byKind = new Map(
    (counts?.byClassification ?? []).map((row) => [row.classification, row.total] as const),
  );
  const read = [...byKind.values()].reduce((sum, count) => sum + count, 0);
  const shown = useCountUp(read, stage === "reading" || stage === "done", 600);
  const current = stepIndex(stage);

  const heading =
    stage === "empty"
      ? t("Nothing has come in yet")
      : stage === "reading"
        ? t("Reading what was already waiting…")
        : stage === "quiet"
          ? t("Connected — nothing was waiting")
          : t("Done reading");
  const lead =
    stage === "empty"
      ? t(
          "Point a shared address at Triage — dispatch@, pod@, ap@. Nova reads what's already sitting there, so your first briefing isn't empty.",
        )
      : stage === "reading"
        ? t("Sorting, matching to shipments, and setting aside anything that needs a person.")
        : stage === "quiet"
          ? t(
              "Nothing was sitting at the address. New mail shows up here, already read, as it arrives.",
            )
          : t("Your first briefing is ready.");

  return (
    <section className="grid min-h-full place-items-center" aria-live="polite">
      <div className="animate-rise flex w-full max-w-130 flex-col items-center gap-3.5 px-7 py-12 text-center">
        <div className="relative mb-1.5 grid size-24 place-items-center">
          <span
            aria-hidden
            data-reading={stage === "reading"}
            className="tb-orb-ring absolute inset-0 rounded-full"
          />
          <span className="relative font-mono text-4xl font-semibold tabular-nums">
            {stage === "done" ? (
              <CheckIcon
                aria-label={t("Done")}
                className="animate-confirm size-7"
                strokeWidth={2.4}
              />
            ) : stage === "empty" ? (
              0
            ) : (
              shown
            )}
          </span>
        </div>
        <h2 className="text-2xl font-semibold">{heading}</h2>
        <p className="text-foreground-subtle max-w-[44ch] text-base leading-relaxed">{lead}</p>
        {stage === "empty" &&
          (canConnect ? (
            <Button size="lg" onClick={onConnect}>
              <PlusIcon className="size-3.5" />
              {t("Add a source")}
            </Button>
          ) : (
            <p className="text-foreground-muted text-sm">
              {t("Ask an administrator to add a source; adding one needs mailbox access.")}
            </p>
          ))}
        {stage !== "empty" && (
          <ul
            aria-label={t("Read so far, by kind")}
            className="mt-1 flex flex-wrap justify-center gap-1.5"
          >
            {INBOUND_CLASSIFICATIONS.map((kind) => {
              const count = byKind.get(kind) ?? 0;
              return (
                <li
                  key={kind}
                  className={cn(
                    "bg-surface-hover inline-flex h-7 items-center gap-2 rounded-full px-2.5 text-sm transition-opacity duration-300",
                    count === 0 && "opacity-35",
                  )}
                >
                  <span aria-hidden className={cn("size-2 rounded-xs", CLASSIFICATION_DOT[kind])} />
                  <span>{classificationLabel(t, kind)}</span>
                  <b className="font-mono text-xs font-semibold tabular-nums">{count}</b>
                </li>
              );
            })}
          </ul>
        )}
        <ol className="mt-4.5 flex flex-wrap justify-center gap-4.5">
          {STEPS.map((step, index) => {
            const done = index < current;
            const on = index === current;
            return (
              <li
                key={step}
                aria-current={on ? "step" : undefined}
                className={cn(
                  "flex items-center gap-1.75 text-sm",
                  done ? "text-success" : on ? "text-foreground" : "text-foreground-subtle",
                )}
              >
                <i
                  aria-hidden
                  className={cn(
                    "flex size-4.5 items-center justify-center rounded-full border font-mono text-2xs not-italic",
                    done
                      ? "bg-success border-success text-foreground-on-solid"
                      : on
                        ? "bg-ink border-ink text-ink-foreground"
                        : "border-border-strong",
                  )}
                >
                  {done ? <CheckIcon className="size-2.5" strokeWidth={3} /> : index + 1}
                </i>
                {step === "source"
                  ? t("Add a source")
                  : step === "backlog"
                    ? t("Nova reads the backlog")
                    : t("Your first briefing")}
              </li>
            );
          })}
        </ol>
      </div>
    </section>
  );
}
