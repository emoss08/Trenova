import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { usePermission } from "@/hooks/use-permission";
import { downloadReportRun, useCancelReportRun } from "@/hooks/use-reports";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { describeCron } from "@/lib/cron";
import type { ReportRunDeskCard } from "@/lib/graphql/report-desk-card";
import { assistant } from "@/lib/queries/assistant";
import { reports } from "@/lib/queries/reports";
import { apiService } from "@/services/api";
import type { AssistantArtifact } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { Button } from "@trenova/shared/components/ui/button";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, useMemo, type ReactNode } from "react";
import { Link } from "react-router";
import { artifactReaderOf, reportRunFrom, type ArtifactReader } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";
import { cardDate, cardDay, cardMoment, cardTime } from "../../record-card/record-card-values";
import { Notice } from "../artifact-body-parts";
import {
  durationText,
  hintsOf,
  isRunActive,
  parameterText,
  reportsPath,
  sheetWindow,
  sourceOfRun,
  windowText,
} from "./report-run-card";
import { kpisOf, pictureOf, SHEET_ROWS, type SheetSource } from "./report-sheet";
import { ReportKpis, ReportSheet, useSheetText } from "./report-sheet-parts";
import { ReportRunSkeleton, ReportSheetSkeleton } from "./report-run-skeleton";

const RUN_POLL_MS = 2_000;

/** A report run the agent or the person started, following the run to its result. */
export function DeskReportRunBody({
  artifact,
  versions,
}: {
  artifact: AssistantArtifact;
  versions: ReactNode;
}) {
  const t = useT();
  const run = useMemo(() => reportRunFrom(artifact), [artifact]);
  const query = useQuery({
    ...reports.deskCard(run?.runId ?? ""),
    enabled: run !== null,
    refetchInterval: (state) =>
      state.state.data && isRunActive(state.state.data.run.status) ? RUN_POLL_MS : false,
  });

  if (run === null) {
    return <Notice>{t("This run has no id to follow.")}</Notice>;
  }
  if (query.isPending) {
    return <ReportRunSkeleton />;
  }
  if (query.isError) {
    return (
      <Notice>{t("This report run could not be read. It may belong to someone else.")}</Notice>
    );
  }

  return (
    <ReportRunCardView
      artifact={artifact}
      card={query.data}
      versions={versions}
      reader={artifactReaderOf(artifact.payload.readBy)}
      ranByPerson={artifact.payload.ranByPerson === true}
    />
  );
}

export function ReportRunCardView({
  artifact,
  card,
  versions,
  reader,
  ranByPerson,
}: {
  artifact: AssistantArtifact;
  card: ReportRunDeskCard;
  versions: ReactNode;
  reader: ArtifactReader | null;
  ranByPerson: boolean;
}) {
  const t = useT();
  const { run } = card;
  const source = useMemo(() => sourceOfRun(card), [card]);
  const now = useNowSeconds();
  const status = run.status;
  const active = isRunActive(status);
  const expired =
    status === "expired" ||
    (status === "succeeded" && run.artifactExpiresAt != null && run.artifactExpiresAt < now);

  return (
    <div className="dkrr">
      <div className="dk-rr">
        {versions && <div className="dk-ax-tools dk-end">{versions}</div>}
        <div className="dk-rr-head">
          <RunHead card={card} reader={reader} ranByPerson={ranByPerson} />
          <RunStamp card={card} expired={expired} />
        </div>
        <RunNotes card={card} expired={expired} />
        {status === "succeeded" && !expired && source && (
          <Suspense fallback={<ReportSheetSkeleton />}>
            <RunResult card={card} source={source} />
          </Suspense>
        )}
        {status === "succeeded" && !expired && !source && !card.withheld && run.rowCount > 0 && (
          <div className="dk-rr-note">
            <ArtIcon name="alert" size={14} />
            <div>
              <b>{t("This run finished before results were kept with it.")}</b>
              {t("Download the file to read it, or run it again to see the result here.")}
            </div>
          </div>
        )}
        {active && <RunningSheet card={card} />}
        <RunActions artifact={artifact} card={card} expired={expired} />
      </div>
    </div>
  );
}

function RunHead({
  card,
  reader,
  ranByPerson,
}: {
  card: ReportRunDeskCard;
  reader: ArtifactReader | null;
  ranByPerson: boolean;
}) {
  const t = useT();
  const { report, run } = card;
  const when = cardMoment(run.queuedAt ?? run.createdAt);
  const kicker = [
    report.category ? translateLabel(report.category) : "",
    report.cannedKey ? t("Built in") : report.definitionId ? t("Saved report") : "",
  ].filter((part) => part !== "");
  const schedule = card.schedule;

  return (
    <div>
      {kicker.length > 0 && (
        <div className="dk-rr-kick">
          <b>{kicker.join(" · ")}</b>
        </div>
      )}
      {report.description && <div className="dk-rr-desc">{report.description}</div>}
      {card.parameters.length > 0 && (
        <div className="dk-rr-params">
          {card.parameters.map((parameter) => (
            <span key={parameter.name} className="dk-rr-param">
              <span>{translateLabel(parameter.label || parameter.name)}</span>
              {parameterText(parameter, t, card.timezone)}
            </span>
          ))}
        </div>
      )}
      <div className="dk-rr-by">
        {schedule ? (
          <span>
            <b>{t("Scheduled")}</b>
            {" · "}
            {describeCron(schedule.cronExpression) ?? schedule.cronExpression}
            {schedule.nextRunAt ? ` · ${t("next {0}", cardMoment(schedule.nextRunAt))}` : ""}
          </span>
        ) : ranByPerson ? (
          <span>{t("You ran it · {0}", when)}</span>
        ) : reader ? (
          <>
            <DeskAgentTile agent={reader} size="xs" />
            <span>{t("{0} ran it for you · {1}", reader.name, when)}</span>
          </>
        ) : (
          <span>{t("Ran · {0}", when)}</span>
        )}
      </div>
    </div>
  );
}

function stampOf(
  status: string,
  expired: boolean,
  t: TranslateFn,
): { label: string; tone: string } {
  if (expired) return { label: t("Expired"), tone: "dk-off" };
  switch (status) {
    case "queued":
      return { label: t("Queued"), tone: "dk-run" };
    case "running":
      return { label: t("Running"), tone: "dk-run" };
    case "succeeded":
      return { label: t("Finished"), tone: "" };
    case "failed":
      return { label: t("Failed"), tone: "dk-fail" };
    case "canceled":
      return { label: t("Canceled"), tone: "dk-off" };
    default:
      return { label: translateLabel(status), tone: "dk-off" };
  }
}

/** Where the run stands, as a ticket: when it queued, how long it ran, what it returned. */
function RunStamp({ card, expired }: { card: ReportRunDeskCard; expired: boolean }) {
  const t = useT();
  const { run } = card;
  const stamp = stampOf(run.status, expired, t);
  const steps: [string, string][] = [];
  const rows =
    run.rowCount === 0 ? t("no rows") : t("{0, plural, one {# row} other {# rows}}", run.rowCount);

  if (expired) {
    steps.push([t("Ran"), cardMoment(run.completedAt ?? run.queuedAt)]);
    steps.push([t("Returned"), rows]);
    if (run.artifactExpiresAt) {
      steps.push([t("File"), t("removed {0}", cardDate(run.artifactExpiresAt))]);
    }
  } else {
    steps.push([t("Queued"), cardTime(run.queuedAt ?? run.createdAt)]);
    if (isRunActive(run.status)) {
      if (run.startedAt) steps.push([t("Started"), cardTime(run.startedAt)]);
      steps.push([t("Rows"), "—"]);
    } else {
      if (run.durationMs > 0) steps.push([t("Ran for"), durationText(run.durationMs, t)]);
      if (run.status === "succeeded") {
        steps.push([t("Returned"), rows]);
        if (run.byteSize > 0) {
          steps.push([t("File"), `${run.format.toUpperCase()} · ${formatFileSize(run.byteSize)}`]);
        }
      }
      if (run.status === "canceled" && run.completedAt) {
        steps.push([t("Canceled"), cardTime(run.completedAt)]);
      }
    }
  }

  return (
    <div className={cn("dk-rr-stamp", stamp.tone)} aria-live="polite">
      <div className="dk-rr-stamp-s">
        <i />
        {stamp.label}
      </div>
      <div className="dk-rr-steps">
        {steps.map(([label, value]) => (
          <div key={label} className={value === "—" ? "dk-wait" : undefined}>
            {label}
            <b>{value}</b>
          </div>
        ))}
      </div>
    </div>
  );
}

function RunNotes({ card, expired }: { card: ReportRunDeskCard; expired: boolean }) {
  const t = useT();
  const { run } = card;

  if (run.status === "failed") {
    return (
      <div className="dk-rr-note dk-fail">
        <ArtIcon name="x" size={14} stroke={2} />
        <div>
          <b>{run.error?.message || t("The report failed.")}</b>
          {run.error?.detail && <span>{run.error.detail}</span>}
          {t("Nothing was written, so there is nothing to download.")}
          {run.error?.code && (
            <div>
              <span className="dk-rr-code">{run.error.code}</span>
            </div>
          )}
        </div>
      </div>
    );
  }
  if (run.status === "canceled") {
    return (
      <div className="dk-rr-note">
        <ArtIcon name="x" size={14} stroke={2} />
        <div>
          <b>{t("The run was canceled before it finished.")}</b>
          {t("Nothing was written. Run it again when you need it.")}
        </div>
      </div>
    );
  }
  if (expired) {
    return (
      <div className="dk-rr-note">
        <ArtIcon name="alert" size={14} />
        <div>
          <b>
            {run.artifactExpiresAt
              ? t("This run's file and rows were removed on {0}.", cardDate(run.artifactExpiresAt))
              : t("This run's file and rows were removed.")}
          </b>
          {t("Run it again for today's figures.")}
        </div>
      </div>
    );
  }
  if (run.status !== "succeeded") {
    return null;
  }
  if (card.withheld) {
    return (
      <div className="dk-rr-note dk-warn">
        <ArtIcon name="lock" size={14} />
        <div>
          <b>
            {card.withheld === "access"
              ? t("You can no longer read everything this report shows.")
              : t("This report has changed since it ran.")}
          </b>
          {card.withheld === "access"
            ? t("Its result is not shown here and its file cannot be downloaded.")
            : t("Its result is not shown here. Run it again for a result you can read.")}
        </div>
      </div>
    );
  }
  if (run.truncated) {
    return (
      <div className="dk-rr-note dk-warn">
        <ArtIcon name="warn" size={14} stroke={2} />
        <div>
          <b>
            {t(
              "Capped at {0, plural, one {# row} other {# rows}}. This isn't the whole set.",
              run.rowCount,
            )}
          </b>
          {t(
            "The figures add up only the rows that were kept. Narrow the report to get every row.",
          )}
        </div>
      </div>
    );
  }
  return null;
}

/** The figures, the picture and the first rows of a finished run. */
function RunResult({ card, source }: { card: ReportRunDeskCard; source: SheetSource }) {
  const t = useT();
  const text = useSheetText(source, card.timezone);
  const picture = pictureOf(source, hintsOf(card));
  const kpis = kpisOf(
    source,
    picture,
    card.previous ? { totals: card.previous.totals, rowCount: card.previous.rowCount } : null,
  );
  const window = sheetWindow(card);
  const title = window
    ? windowText(window.start, window.end, t, { timeZone: card.timezone, withYear: true })
    : cardDate(card.run.completedAt ?? card.run.createdAt);
  const size = t(
    "{0, plural, one {# row} other {# rows}} · {1, plural, one {# column} other {# columns}}",
    source.rowCount,
    source.columns.length,
  );

  if (source.rowCount === 0 || source.rows.length === 0) {
    return (
      <div className="dk-rr-sheet">
        <div className="dk-rr-sheet-h">
          <b>{title}</b>
          <span>{size}</span>
        </div>
        <div className="dk-rr-empty">
          <b>{t("Nothing matched")}</b>
          {t("The report finished and matched nothing. That is an answer, not an error.")}
        </div>
      </div>
    );
  }

  const shown = Math.min(SHEET_ROWS, source.rows.length);
  const moreRows = source.rowCount - shown;
  const moreColumns = Math.max(0, source.columns.length - 6);
  const rankedBy =
    picture.kind === "ranked" && picture.ranked ? source.columns[picture.measure]?.label : null;

  return (
    <>
      {kpis.length > 1 && (
        <ReportKpis
          source={source}
          kpis={kpis}
          text={text}
          since={card.previous ? cardDay(card.previous.completedAt) : ""}
        />
      )}
      <ReportSheet
        source={source}
        picture={picture}
        text={text}
        title={title}
        size={size}
        rows={SHEET_ROWS}
        footer={
          (moreRows > 0 || moreColumns > 0 || rankedBy) && (
            <div className="dk-rr-more">
              <span>
                {moreRows > 0 && moreColumns > 0
                  ? t(
                      "{0, plural, one {# more row} other {# more rows}} and {1, plural, one {# more column} other {# more columns}} are in the file",
                      moreRows,
                      moreColumns,
                    )
                  : moreRows > 0
                    ? t(
                        "{0, plural, one {# more row is} other {# more rows are}} in the file",
                        moreRows,
                      )
                    : moreColumns > 0
                      ? t(
                          "{0, plural, one {# more column is} other {# more columns are}} in the file",
                          moreColumns,
                        )
                      : ""}
              </span>
              {rankedBy && <span>{t("Ranked by {0}", rankedBy.toLowerCase())}</span>}
            </div>
          )
        }
      />
    </>
  );
}

/** The sheet while the run is going: the shape of a result, filling in when it finishes. */
function RunningSheet({ card }: { card: ReportRunDeskCard }) {
  const t = useT();
  const window = sheetWindow(card);

  return (
    <div className="dk-rr-sheet" aria-busy="true" aria-label={t("Running the report")}>
      <div className="dk-rr-sheet-h">
        <b>
          {window
            ? windowText(window.start, window.end, t, { timeZone: card.timezone, withYear: true })
            : t("Running the report")}
        </b>
        <span>{t("Reading…")}</span>
      </div>
      <div className="dk-rr-pic">
        <div className="dk-rr-bars">
          {[96, 82, 70, 58, 47, 39].map((width) => (
            <div key={width} className="dk-rr-bar">
              <span className="ui-shimmer dk-rr-shim-line" />
              <span className="dk-rr-tr">
                <i className="dk-rr-shim-bar" style={{ width: `${width}%` }} />
              </span>
              <span className="ui-shimmer dk-rr-shim-line" />
              <span />
            </div>
          ))}
        </div>
      </div>
      <div className="dk-rr-more">
        <span>
          {t("The page fills in when the run finishes; you can leave the conversation meanwhile.")}
        </span>
      </div>
    </div>
  );
}

function RunActions({
  artifact,
  card,
  expired,
}: {
  artifact: AssistantArtifact;
  card: ReportRunDeskCard;
  expired: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { run } = card;
  const { allowed: mayRun } = usePermission(Resource.Report, Operation.Export);
  const cancel = useCancelReportRun();
  const rerun = useApiMutation({
    mutationFn: () => apiService.assistantService.runArtifactReport(artifact.threadId, artifact.id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: assistant.artifacts(artifact.threadId).queryKey,
      });
    },
    resourceName: "Report",
  });
  const path = reportsPath(card.report);
  const finished = run.status === "succeeded" && !expired;
  const downloadable = finished && run.rowCount > 0 && card.withheld !== "access";
  const runAgainFirst = !finished && !isRunActive(run.status);

  const runAgain = mayRun && !isRunActive(run.status) && (
    <Button
      variant={runAgainFirst ? "default" : "quiet"}
      disabled={rerun.isPending}
      onClick={() => rerun.mutate(undefined)}
    >
      <ArtIcon name="retry" size={12} stroke={2} />
      {rerun.isPending ? t("Starting…") : t("Run again")}
    </Button>
  );

  return (
    <div className="dk-rr-acts">
      {downloadable && (
        <Button onClick={() => downloadReportRun({ id: run.id })}>
          <ArtIcon name="dl" size={13} stroke={2} />
          {t("Download {0}", run.format.toUpperCase())}
        </Button>
      )}
      {runAgain}
      {isRunActive(run.status) && run.requestedById && (
        <Button variant="quiet" disabled={cancel.isPending} onClick={() => cancel.mutate(run.id)}>
          <ArtIcon name="x" size={12} stroke={2} />
          {t("Cancel run")}
        </Button>
      )}
      {path && (
        <Link className="dk-ax-btn dk-ghost" to={path}>
          <ArtIcon name="ext" size={12} stroke={1.9} />
          {t("Open in Reports")}
        </Link>
      )}
      <span className="dk-rr-sp" />
      {finished && run.artifactExpiresAt && (
        <small>{t("The file is kept until {0}", cardDate(run.artifactExpiresAt))}</small>
      )}
    </div>
  );
}
