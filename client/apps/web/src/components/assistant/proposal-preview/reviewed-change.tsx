import { PlanPreview } from "@/components/assistant/proposal-preview/plan-preview";
import {
  PreviewLoadState,
  ProposalPreview,
} from "@/components/assistant/proposal-preview/proposal-preview";
import {
  usePlanPreview,
  useProposalPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import type { AssistantProposal } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useEffect, useState } from "react";

/**
 * Where the preview of a change the server approves only once it has been
 * shown stands: still loading, failed to load, out of date (the server refuses
 * an approval of it), or shown and current, with the digest it carries.
 */
export type ReviewState = "loading" | "failed" | "stale" | "shown";

export type ReviewReport = { state: ReviewState; digest: string };

type Report = (id: string, report: ReviewReport) => void;

function reportOf(query: {
  data: { digest: string; stale: boolean } | undefined;
  isError: boolean;
}): ReviewReport {
  if (query.data !== undefined) {
    return query.data.stale
      ? { state: "stale", digest: "" }
      : { state: "shown", digest: query.data.digest };
  }

  return { state: query.isError ? "failed" : "loading", digest: "" };
}

function useReportedState(id: string, report: ReviewReport, onReport: Report) {
  const { state, digest } = report;
  useEffect(() => {
    onReport(id, { state, digest });
  }, [digest, id, onReport, state]);
}

/** The preview of one change the server approves only once it has been shown. */
export function ReviewedChange({
  proposal,
  onReport,
}: {
  proposal: AssistantProposal;
  onReport: Report;
}) {
  const t = useT();
  const query = useProposalPreview({ scope: "mine", id: proposal.id });
  useReportedState(proposal.id, reportOf(query), onReport);
  const title = t(presentProposal(proposal).title);

  return (
    <section className="flex min-w-0 flex-col gap-2" aria-label={t("Preview of {0}", title)}>
      <b className="text-sm font-medium">{title}</b>
      <PreviewLoadState query={query} changed={false} density="compact" reviewRequired>
        {(preview) => <ProposalPreview preview={preview} density="compact" />}
      </PreviewLoadState>
    </section>
  );
}

/** The preview of a plan with a step the server approves only once it has been shown. */
export function ReviewedPlan({ planId, onReport }: { planId: string; onReport: Report }) {
  const query = usePlanPreview({ scope: "mine", id: planId });
  useReportedState(planId, reportOf(query), onReport);

  return (
    <PreviewLoadState query={query} changed={false} density="compact" reviewRequired>
      {(plan) => <PlanPreview plan={plan} density="compact" />}
    </PreviewLoadState>
  );
}

/**
 * What each reviewed preview reported, and the digests of the ones shown and
 * current: those are the only digests an approval may carry.
 */
export function useReviewReports() {
  const [reports, setReports] = useState<ReadonlyMap<string, ReviewReport>>(new Map());
  const onReport = useCallback((id: string, report: ReviewReport) => {
    setReports((previous) => {
      const current = previous.get(id);
      if (current?.state === report.state && current.digest === report.digest) {
        return previous;
      }

      return new Map(previous).set(id, report);
    });
  }, []);

  const shown = new Map<string, string>();
  for (const [id, report] of reports) {
    if (report.state === "shown") {
      shown.set(id, report.digest);
    }
  }

  return { reports, shown, onReport };
}

export type ReviewTally = { loading: number; failed: number; stale: number };

/** How many of the changes that must be reviewed are not yet shown, by why. */
export function reviewTally(
  ids: readonly string[],
  reports: ReadonlyMap<string, ReviewReport>,
): ReviewTally {
  const tally: ReviewTally = { loading: 0, failed: 0, stale: 0 };
  for (const id of ids) {
    const state = reports.get(id)?.state ?? "loading";
    if (state !== "shown") {
      tally[state] += 1;
    }
  }

  return tally;
}

/**
 * What stops a reviewed approval from going through, or nothing when nothing
 * does. A plan is one preview, and is spoken of as the plan.
 */
export function ReviewBlocker({ tally, plan = false }: { tally: ReviewTally; plan?: boolean }) {
  const t = useT();
  if (plan) {
    const message =
      tally.stale > 0
        ? t("The plan's records changed since it was drafted, so it can only be set aside")
        : tally.failed > 0
          ? t("The plan's preview could not be loaded; try it again above")
          : tally.loading > 0
            ? t("Waiting for the plan's preview to load")
            : null;

    return message === null ? null : (
      <span className="text-muted-foreground text-sm" role="status">
        {message}
      </span>
    );
  }

  const messages: string[] = [];
  if (tally.stale > 0) {
    messages.push(
      t(
        "{0, plural, one {# change has changed since it was drafted and can only be set aside} other {# changes have changed since they were drafted and can only be set aside}}",
        tally.stale,
      ),
    );
  }
  if (tally.failed > 0) {
    messages.push(
      t(
        "{0, plural, one {# preview could not be loaded; try it again above} other {# previews could not be loaded; try them again above}}",
        tally.failed,
      ),
    );
  }
  if (tally.loading > 0) {
    messages.push(
      t(
        "{0, plural, one {Waiting for # preview to load} other {Waiting for # previews to load}}",
        tally.loading,
      ),
    );
  }
  if (messages.length === 0) {
    return null;
  }

  return (
    <span className="text-muted-foreground text-sm" role="status">
      {messages.join(" · ")}
    </span>
  );
}
