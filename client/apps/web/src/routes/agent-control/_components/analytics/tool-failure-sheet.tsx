import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AIAnalyticsInput, AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { aiAnalytics } from "@/lib/queries/ai-analytics";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatNumber, formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import { Ic } from "../kit/ic";
import { ReadSheet } from "../kit/read-sheet";
import { CAUSES } from "./analytics-model";
import { ProviderMarks } from "./provider-marks";

const EXAMPLES_SHOWN = 3;

type ToolFailureSheetProps = {
  open: boolean;
  input: AIAnalyticsInput;
  series: AIAnalyticsSeries | null;
  seriesKey: string | null;
  toolName: string | null;
  onOpenToolRules: () => void;
  onOpenFailedRuns: () => void;
  onClose: () => void;
};

/** Why one tool's calls failed for one series, whose fault each was, and the newest failures. */
export function ToolFailureSheet({
  open,
  input,
  series,
  seriesKey,
  toolName,
  onOpenToolRules,
  onOpenFailedRuns,
  onClose,
}: ToolFailureSheetProps) {
  const t = useT();
  const now = useNow("minute");
  const detailQuery = useQuery({
    ...aiAnalytics.toolFailures(input, seriesKey ?? "", toolName ?? ""),
    enabled: open && Boolean(seriesKey && toolName),
  });
  const detail = detailQuery.data;
  const title = toolName ? describeToolCall(toolName, null).title : "";

  return (
    <ReadSheet
      open={open}
      onClose={onClose}
      label={t("{0} failures", title)}
      head={
        <>
          <ProviderMarks
            providers={detail?.providers ?? series?.providers ?? []}
            s={30}
            limit={3}
          />
          <div className="sh-t">
            <b>
              {title}
              {series ? ` · ${series.label}` : ""}
            </b>
            <span>
              {detail
                ? t(
                    "{0} of {1} calls failed",
                    formatNumber(detail.failed),
                    formatNumber(detail.calls),
                  )
                : "…"}
            </span>
          </div>
        </>
      }
    >
      {detailQuery.isLoading || !detail ? (
        <div className="sh-p flex flex-col gap-2" aria-busy>
          <Skeleton className="h-5 w-2/3" />
          <Skeleton className="h-5 w-1/2" />
          <Skeleton className="h-5 w-3/5" />
        </div>
      ) : (
        <>
          <div className="sh-p">
            <h4 className="sh-k">{t("Why")}</h4>
            {detail.causes.length === 0 ? (
              <p className="ad-h">{t("No failures in this range.")}</p>
            ) : (
              detail.causes.map((cause) => (
                <div key={cause.cause} className="an-wr">
                  <span className="an-wl">
                    <b>
                      {t(CAUSES.find((entry) => entry.cause === cause.cause)?.label ?? cause.cause)}
                    </b>
                    <span className="mono">
                      {formatNumber(cause.calls)} ·{" "}
                      {cause.blame === "Model" ? t("the model") : t("the tool or a rule")}
                    </span>
                  </span>
                  <span className="an-stk">
                    <i className={`r-${cause.cause}`} style={{ flexGrow: cause.calls }} />
                    <i
                      className="z"
                      style={{ flexGrow: Math.max(0, detail.failed - cause.calls) }}
                    />
                  </span>
                </div>
              ))
            )}
          </div>
          {detail.recent.length > 0 && (
            <div className="sh-p">
              <h4 className="sh-k">{t("Recent failures")}</h4>
              {detail.recent.slice(0, EXAMPLES_SHOWN).map((example) => (
                <div key={`${example.unitId}-${example.at}-${example.message}`} className="an-ex">
                  <code className="mono">{example.toolName}</code>
                  <span>{example.message || t("The tool gave no reason.")}</span>
                  <em>
                    {[example.agentName, formatRelativeTime(example.at - now)]
                      .filter(Boolean)
                      .join(" · ")}
                  </em>
                </div>
              ))}
            </div>
          )}
        </>
      )}
      <div className="ad-bar sh-f">
        <button type="button" className="xa ui-focus-ring" onClick={onOpenToolRules}>
          <Ic n="shield" s={13} />
          {t("Tool rules")}
        </button>
        <button type="button" className="xa ui-focus-ring" onClick={onOpenFailedRuns}>
          <Ic n="timeline" s={13} />
          {t("Failed runs")}
        </button>
      </div>
    </ReadSheet>
  );
}
