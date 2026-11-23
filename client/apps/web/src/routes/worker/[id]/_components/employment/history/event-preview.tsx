import {
  previewWorkerEmploymentEvent,
  WORKER_EMPLOYMENT_PREVIEW_KEY,
  type EmploymentEffect,
} from "@/lib/graphql/worker-employment";
import { useQuery } from "@tanstack/react-query";
import type { RecordWorkerEmploymentEventInput } from "@trenova/graphql/generated/graphql";
import { AlertTriangleIcon, CheckIcon, ClockRewindIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useSettledValue } from "@trenova/shared/hooks/use-settled-value";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { describeEmploymentEffect } from "./employment-effects";
import { pairValues, ValueChips } from "./employment-values";

/** How long the inputs must rest before the preview is read again. */
const SETTLE_MS = 300;

type EventPreviewProps = {
  /** The event as it would be recorded, or null while the form cannot yet say what it is. */
  input: RecordWorkerEmploymentEventInput | null;
  /** Why there is nothing to read yet: no kind chosen, or the kind's own fields are empty. */
  waiting: string;
  workerName: string;
};

/**
 * "What this will do": the server runs the same plan the write runs and writes nothing, and
 * this lists each effect in words. It reads again once the kind, the date or a field that
 * changes the outcome has rested, and a newer read cancels the one in flight.
 */
export function EventPreview({ input, waiting, workerName }: EventPreviewProps) {
  const t = useT();
  const inputKey = input === null ? null : JSON.stringify(input);
  const settledKey = useSettledValue(inputKey, SETTLE_MS);
  const preview = useQuery({
    queryKey: [WORKER_EMPLOYMENT_PREVIEW_KEY, settledKey],
    queryFn: ({ signal }) =>
      previewWorkerEmploymentEvent(
        JSON.parse(settledKey ?? "null") as RecordWorkerEmploymentEventInput,
        { signal },
      ),
    enabled: settledKey !== null,
    retry: false,
    staleTime: 30_000,
  });
  const terminating = input?.kind === "Terminated";

  return (
    <section aria-label={t("What this will do")} className="flex flex-col gap-2 border-t pt-4">
      <h4 className="text-xs font-medium text-foreground-subtle">{t("What this will do")}</h4>
      {terminating && input ? (
        <Alert variant="destructive" size="sm">
          <AlertTriangleIcon />
          <AlertTitle>{t("This ends employment.")}</AlertTitle>
          <AlertDescription>
            {t(
              "{0} leaves on {1}. You can record a rehire later.",
              workerName,
              formatUnixDateMedium(input.effectiveAt),
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      {input === null ? (
        <p className="text-xs text-muted-foreground">{waiting}</p>
      ) : preview.isError && settledKey === inputKey ? (
        <ErrorState
          layout="compact"
          error={preview.error}
          title={t("What this will change could not be loaded")}
          description={t("Saving still works.")}
          onRetry={() => void preview.refetch()}
        />
      ) : preview.data && settledKey === inputKey ? (
        <PreviewList
          key={settledKey}
          effects={preview.data.effects}
          pairs={pairValues(preview.data)}
        />
      ) : (
        <div
          aria-busy
          aria-label={t("Reading what this will change")}
          className="flex flex-col gap-2 py-1"
        >
          <span aria-hidden className="block h-2.5 w-40 ui-shimmer rounded-sm" />
          <span aria-hidden className="block h-2.5 w-56 ui-shimmer rounded-sm" />
        </div>
      )}
    </section>
  );
}

/** An effect's identity in a plan: its kind and what it acts on; a plan never repeats both. */
function effectKey(effect: EmploymentEffect): string {
  return [effect.kind, effect.name ?? "", effect.ptoType ?? ""].join(":");
}

function PreviewList({
  effects,
  pairs,
}: {
  effects: readonly EmploymentEffect[];
  pairs: ReturnType<typeof pairValues>;
}) {
  const t = useT();
  return (
    <div className="flex animate-rise flex-col gap-2 motion-reduce:animate-none">
      <ValueChips pairs={pairs} />
      <ul className="flex flex-col gap-1.5">
        {effects.map((effect) => {
          const line = describeEmploymentEffect(effect, t, "will");
          const Icon = line.icon;
          return (
            <li
              key={effectKey(effect)}
              className={cn(
                "flex items-start gap-2 text-sm",
                line.danger && "text-danger-foreground",
              )}
            >
              <Icon aria-hidden className="mt-0.5 size-3.5 shrink-0" />
              <span className="flex min-w-0 flex-col">
                <span>{line.text}</span>
                {line.sub ? (
                  <span className="text-xs text-muted-foreground">{line.sub}</span>
                ) : null}
              </span>
            </li>
          );
        })}
        <li className="flex items-start gap-2 text-sm">
          {effects.length === 0 ? (
            <CheckIcon aria-hidden className="mt-0.5 size-3.5 shrink-0" />
          ) : (
            <ClockRewindIcon aria-hidden className="mt-0.5 size-3.5 shrink-0" />
          )}
          <span>
            {effects.length === 0
              ? t("Writes the event to the employment history. Nothing else changes.")
              : t("Writes the event to the employment history.")}
          </span>
        </li>
      </ul>
    </div>
  );
}
