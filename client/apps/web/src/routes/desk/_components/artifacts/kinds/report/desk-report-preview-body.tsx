import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import { artifactCsvUrl } from "@/services/assistant";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { downloadFromUrl } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { Suspense, useMemo, type ReactNode } from "react";
import { Link } from "react-router";
import { artifactReaderOf, reportPreviewFrom } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";
import { cardMoment } from "../../record-card/record-card-values";
import { reportsPath, sourceOfPreview } from "./report-run-card";
import { kpisOf, pictureOf, type SheetSource } from "./report-sheet";
import { ReportKpis, ReportSheet, useSheetText } from "./report-sheet-parts";
import { ReportSheetSkeleton } from "./report-run-skeleton";

const NO_HINTS = { chart: null, sort: [] };

/**
 * A report the agent previewed: the same sheet a run draws, of the rows the
 * preview read, with the full report one step away.
 */
export function DeskReportPreviewBody({
  artifact,
  versions,
}: {
  artifact: AssistantArtifact;
  versions: ReactNode;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const preview = useMemo(() => reportPreviewFrom(artifact), [artifact]);
  const source = useMemo(() => sourceOfPreview(preview), [preview]);
  const reader = artifactReaderOf(artifact.payload.readBy);
  const definitionId =
    typeof artifact.payload.definitionId === "string" ? artifact.payload.definitionId : "";
  const reportKey =
    typeof artifact.payload.reportKey === "string" ? artifact.payload.reportKey : "";
  const path = reportsPath({ definitionId, cannedKey: reportKey });
  const { allowed: mayRun } = usePermission(Resource.Report, Operation.Export);
  const runnable = mayRun && path !== "" && typeof artifact.payload.readWith === "object";
  const runFull = useApiMutation({
    mutationFn: () => apiService.assistantService.runArtifactReport(artifact.threadId, artifact.id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: assistant.artifacts(artifact.threadId).queryKey,
      });
    },
    resourceName: "Report",
  });
  const when = cardMoment(artifact.updatedAt);
  const dataset = preview.dataset.replace(/[_.-]+/gu, " ").trim();

  return (
    <div className="dkrr">
      <div className="dk-rr">
        {versions && <div className="dk-ax-tools dk-end">{versions}</div>}
        <div className="dk-rr-head">
          <div>
            <div className="dk-rr-kick">
              <b>
                {dataset
                  ? t(
                      "Preview · {0}",
                      translateLabel(dataset.charAt(0).toUpperCase() + dataset.slice(1)),
                    )
                  : t("Preview")}
              </b>
            </div>
            <div className="dk-rr-by">
              {reader ? (
                <>
                  <DeskAgentTile agent={reader} size="xs" />
                  <span>{t("{0} previewed it · {1}", reader.name, when)}</span>
                </>
              ) : (
                <span>{t("Previewed · {0}", when)}</span>
              )}
            </div>
          </div>
        </div>
        {source.rows.length === 0 ? (
          <div className="dk-rr-sheet">
            <div className="dk-rr-empty">
              <b>{t("Nothing matched")}</b>
              {t("The preview ran and matched nothing. The filters may be too narrow.")}
            </div>
          </div>
        ) : (
          <Suspense fallback={<ReportSheetSkeleton />}>
            <PreviewResult source={source} title={artifact.title} />
          </Suspense>
        )}
        <div className="dk-rr-acts">
          <Button
            variant="quiet"
            onClick={() => downloadFromUrl(artifactCsvUrl(artifact.threadId, artifact.id))}
          >
            <ArtIcon name="dl" size={13} stroke={2} />
            {t("Download CSV")}
          </Button>
          {runnable && (
            <Button disabled={runFull.isPending} onClick={() => runFull.mutate(undefined)}>
              <ArtIcon name="retry" size={12} stroke={2} />
              {runFull.isPending ? t("Starting…") : t("Run the full report")}
            </Button>
          )}
          {path && (
            <Link className="dk-ax-btn dk-ghost" to={path}>
              <ArtIcon name="ext" size={12} stroke={1.9} />
              {t("Open in Reports")}
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}

function PreviewResult({ source, title }: { source: SheetSource; title: string }) {
  const t = useT();
  const text = useSheetText(source);
  const picture = pictureOf(source, NO_HINTS);
  const kpis = kpisOf(source, picture, null);
  const size = t(
    "{0, plural, one {# row} other {# rows}} · {1, plural, one {# column} other {# columns}}",
    source.rowCount,
    source.columns.length,
  );

  return (
    <>
      {kpis.length > 1 && <ReportKpis source={source} kpis={kpis} text={text} since="" />}
      <ReportSheet
        source={source}
        picture={picture}
        text={text}
        title={title}
        size={size}
        rows={source.rows.length}
        footer={
          (source.more || source.truncated) && (
            <div className="dk-rr-more">
              <span>
                {source.truncated
                  ? t("The preview stopped at its row cap; run the full report for every row.")
                  : t(
                      "The preview shows the first {0} of {1} rows; run the full report for every row.",
                      source.rows.length,
                      source.rowCount,
                    )}
              </span>
            </div>
          )
        }
      />
    </>
  );
}
