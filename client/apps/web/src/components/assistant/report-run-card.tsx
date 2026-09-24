import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { downloadReportRun, isReportRunActive, useReportRun } from "@/hooks/use-reports";
import {
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  DownloadIcon,
  FileSpreadsheetIcon,
} from "lucide-react";
import type { ThreadReportRun } from "./report-runs";
import { WorkingDot } from "./voice/working-dot";

/**
 * A report the assistant started, following itself to the finish.
 *
 * A run is asynchronous by design — it is compiled, queued and executed away
 * from the turn that asked for it — so the conversation used to end at "I have
 * queued it" and the only way to learn the outcome was to ask again, which the
 * model answered by calling a status tool and reading the row count aloud. The
 * card replaces that whole loop: it polls the run itself, shows what state the
 * run is actually in, and offers the artifact when there is one.
 *
 * The download is a button rather than a link the model writes, because a
 * report artifact is presigned for under a minute against the person asking for
 * it. A URL pasted into a message is expired before anyone reads it and belongs
 * to whoever the model was acting for rather than to whoever opens the thread
 * later — so the link is minted at the moment of the click, by the viewer's own
 * session, through the same endpoint the Reports page uses.
 */
export function ReportRunCard({ run }: { run: ThreadReportRun }) {
  const t = useT();
  const query = useReportRun(run.runId, { poll: true });
  const record = query.data;

  const status = record?.status ?? "queued";
  const active = isReportRunActive(status);
  const failure = record?.error?.message ?? "";
  // The finish is marked with a spring only when this card watched the run
  // get there; a finished run opened from history simply is finished.
  const [firstStatus, setFirstStatus] = useState<string | null>(null);
  if (record && firstStatus === null) {
    setFirstStatus(record.status);
  }
  const landed = firstStatus !== null && isReportRunActive(firstStatus) && !active;

  return (
    <div
      data-slot="report-run"
      data-status={status}
      className="border-border bg-card flex min-w-0 items-center gap-3 rounded-lg border px-3 py-2.5"
    >
      <StatusMark status={status} working={query.isPending || active} landed={landed} />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5" aria-live="polite">
        <span className="truncate text-sm font-semibold">{reportLabel(run, t)}</span>
        <span
          key={status}
          className={cn("text-foreground-muted text-xs", landed && "animate-rise")}
        >
          {describe({ status, record, failure, t })}
        </span>
      </div>
      {record?.status === "succeeded" && (
        <Button
          type="button"
          size="sm"
          variant="outline"
          className={cn("shrink-0", landed && "animate-rise")}
          onClick={() => downloadReportRun({ id: run.runId })}
        >
          <DownloadIcon className="size-3.5" />
          {t("Download")}
        </Button>
      )}
    </div>
  );
}

/**
 * The tool names the report when it can. Failing that, the run carries the
 * canned key it was compiled from, so the key is shown as a name rather than
 * left as a slug nobody set.
 */
function reportLabel(run: ThreadReportRun, t: ReturnType<typeof useT>): string {
  if (run.reportName !== "") {
    return run.reportName;
  }
  if (run.reportKey === "") {
    return t("Report");
  }

  return run.reportKey
    .split("-")
    .map((word) => (word === "" ? word : word[0].toUpperCase() + word.slice(1)))
    .join(" ");
}

type DescribeArgs = {
  status: string;
  record: ReturnType<typeof useReportRun>["data"];
  failure: string;
  t: ReturnType<typeof useT>;
};

/**
 * What the run is doing, in the reader's terms. Every branch says something:
 * a run with no rows is a finished run that matched nothing, which is a real
 * answer and must not read as though the report is still working.
 */
function describe({ status, record, failure, t }: DescribeArgs): string {
  switch (status) {
    case "queued":
      return t("Queued — waiting to start.");
    case "running":
      return t("Running — this is not finished yet.");
    case "succeeded": {
      const rows = record?.rowCount ?? 0;
      if (rows === 0) {
        return t("Finished with no matching rows.");
      }
      const finished =
        record?.truncated === true
          ? t("Finished with {0} rows, capped — this is not the full set.", rows)
          : t("Finished with {0} rows.", rows);

      return finished;
    }
    case "failed":
      return failure === "" ? t("The report failed.") : t("Failed: {0}", failure);
    case "canceled":
      return t("Canceled before it finished.");
    case "expired":
      return t("The result expired — run it again to get a fresh copy.");
    default:
      return status;
  }
}

/**
 * The run's state as a mark in a small well: the spreadsheet with a breathing
 * dot while it is still going — the one loop the product allows, because the
 * work it describes is still running — and a tone once it lands, which
 * settles in with the confirm spring so the finish is felt, not just seen.
 */
function StatusMark({
  status,
  working,
  landed,
}: {
  status: string;
  working: boolean;
  landed: boolean;
}) {
  const glyph = "size-4";
  const settled = !working;

  return (
    <span className="bg-sunken relative flex size-8 shrink-0 items-center justify-center rounded-md">
      <span key={settled ? status : "working"} className={cn("flex", landed && "animate-confirm")}>
        {settled && status === "succeeded" ? (
          <CircleCheckIcon className={cn(glyph, "text-success")} />
        ) : settled && status === "failed" ? (
          <CircleAlertIcon className={cn(glyph, "text-danger")} />
        ) : settled && (status === "canceled" || status === "expired") ? (
          <CircleSlashIcon className={cn(glyph, "text-warning")} />
        ) : (
          <FileSpreadsheetIcon className={cn(glyph, "text-foreground-muted")} />
        )}
      </span>
      {working && <WorkingDot working className="absolute -top-0.5 -right-0.5 ring-2 ring-card" />}
    </span>
  );
}
