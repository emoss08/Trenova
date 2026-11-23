import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { WorkerFormSubmission, WorkerHosViolation } from "@/lib/graphql/telematics";
import { telematics } from "@/lib/queries/telematics";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ChevronRightIcon } from "@trenova/shared/components/icons";
import { VirtualRows } from "@/components/virtual-rows";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn, toTitleCase } from "@trenova/shared/lib/utils";
import { useId, useState } from "react";
import { formatLaneTotal, HOS_LIVE_REFRESH_MS } from "./hos-model";

function PanelLoading({ label }: { label: string }) {
  return (
    <div aria-busy aria-label={label} className="flex flex-col gap-2 p-3">
      <div aria-hidden className="contents">
        <span className="block h-3 w-3/5 ui-shimmer rounded-sm" />
        <span className="block h-3 w-2/5 ui-shimmer rounded-sm" />
      </div>
    </div>
  );
}

/**
 * Hours-of-service violations in the window, in time order. The server caps the read at 500 for
 * 30 days, so it is one read, drawn a screenful at a time. A violation happening now is on the
 * band, which is live; this record is read again when the view is opened after a minute.
 */
export function ViolationsPanel({ workerId, since }: { workerId: string; since: number }) {
  const t = useT();
  const query = useQuery({
    ...telematics.workerHosViolations(workerId, since),
    staleTime: HOS_LIVE_REFRESH_MS,
  });
  const violations = query.data ?? [];

  return (
    <SectionPanel title={t("Violations")} hint={t("Last 30 days")} count={violations.length}>
      {query.isPending ? (
        <PanelLoading label={t("Loading violations")} />
      ) : query.isError ? (
        <div className="p-3">
          <ErrorState
            layout="compact"
            error={query.error}
            onRetry={() => void query.refetch()}
            title={t("The violations could not be loaded")}
          />
        </div>
      ) : violations.length === 0 ? (
        <SectionPanelQuiet>{t("None.")}</SectionPanelQuiet>
      ) : (
        <VirtualRows
          rows={violations.map((violation) => ({
            key: `${violation.violationType}-${violation.violationStartAt}-${violation.detectedAt}`,
            render: () => <ViolationRow violation={violation} />,
          }))}
          aria-label={t("Violations")}
          mask="card"
          className="max-h-72"
          contentClassName="px-1.5 py-1"
          hoverGlide
          estimateSize={53}
        />
      )}
    </SectionPanel>
  );
}

function ViolationRow({ violation }: { violation: WorkerHosViolation }) {
  return (
    <div data-glide-target className="flex items-center justify-between gap-3 rounded-md px-2 py-2">
      <div className="min-w-0">
        <p className="truncate font-medium">{toTitleCase(violation.violationType)}</p>
        {violation.description ? (
          <p className="truncate text-xs text-muted-foreground">{violation.description}</p>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="font-mono text-xs text-muted-foreground tabular-nums">
          {formatUnixDateMedium(violation.violationStartAt)}
        </span>
        <Badge variant="danger">{formatLaneTotal(violation.durationMs)}</Badge>
      </div>
    </div>
  );
}

/** Forms the driver sent from the cab in the window: inspections, receipts, anything mapped. */
export function DriverFormsPanel({
  workerId,
  startTime,
  endTime,
}: {
  workerId: string;
  startTime: number;
  endTime: number;
}) {
  const t = useT();
  const query = useQuery({
    ...telematics.workerFormSubmissions(workerId, startTime, endTime),
    staleTime: HOS_LIVE_REFRESH_MS,
  });
  const submissions = [...(query.data ?? [])].sort((a, b) => b.submittedAt - a.submittedAt);

  return (
    <SectionPanel title={t("Driver forms")} hint={t("Last 30 days")} count={submissions.length}>
      {query.isPending ? (
        <PanelLoading label={t("Loading driver forms")} />
      ) : query.isError ? (
        <div className="p-3">
          <ErrorState
            layout="compact"
            error={query.error}
            onRetry={() => void query.refetch()}
            title={t("The driver forms could not be loaded")}
          />
        </div>
      ) : submissions.length === 0 ? (
        <SectionPanelQuiet>{t("No forms sent in the last 30 days.")}</SectionPanelQuiet>
      ) : (
        <VirtualRows
          rows={submissions.map((submission) => ({
            key: submission.id,
            render: () => <FormRow submission={submission} />,
          }))}
          aria-label={t("Driver forms")}
          mask="card"
          className="max-h-72"
          contentClassName="px-1.5 py-1"
          hoverGlide
          estimateSize={53}
        />
      )}
    </SectionPanel>
  );
}

function FormRow({ submission }: { submission: WorkerFormSubmission }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const fieldsId = useId();
  const hasFields = submission.fields.length > 0;

  return (
    <div>
      <button
        type="button"
        data-glide-target
        disabled={!hasFields}
        aria-expanded={hasFields ? open : undefined}
        aria-controls={hasFields ? fieldsId : undefined}
        onClick={() => setOpen((value) => !value)}
        className="flex w-full items-center gap-2.5 rounded-md ui-focus-ring px-2 py-2 text-left disabled:cursor-default"
      >
        <ChevronRightIcon
          aria-hidden
          className={cn(
            "size-3 shrink-0 text-muted-foreground transition-transform duration-200 ease-settle",
            open && "rotate-90",
            !hasFields && "invisible",
          )}
        />
        <span className="min-w-0 flex-1">
          <span className="block truncate font-medium">{submission.templateName}</span>
          <span className="block text-xs text-muted-foreground tabular-nums">
            {formatUnixDateTimeMedium(submission.submittedAt)}
          </span>
        </span>
        <span className="shrink-0 text-xs text-muted-foreground tabular-nums">
          {t("{0, plural, one {# field} other {# fields}}", submission.fields.length)}
        </span>
      </button>
      {hasFields && open ? (
        <dl
          id={fieldsId}
          className="mx-2 mb-2 grid animate-expand-in grid-cols-1 gap-x-4 gap-y-2 rounded-md border border-border-subtle bg-sunken px-3 py-2.5 motion-reduce:animate-none sm:grid-cols-2"
        >
          {submission.fields.map((field, index) => (
            <div key={`${field.label}-${index}`} className="min-w-0">
              <dt className="text-xs text-foreground-subtle">{field.label}</dt>
              <dd className="break-words">{field.value || "—"}</dd>
            </div>
          ))}
        </dl>
      ) : null}
    </div>
  );
}
