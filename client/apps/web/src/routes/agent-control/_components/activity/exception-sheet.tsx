import type { AgentExceptionRow } from "@/lib/graphql/agent-activity-tables";
import type { AgentResolutionState } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useQueryState } from "nuqs";
import { createContext, useContext } from "react";
import { RUN_OPEN_PARAM, runOpenParser } from "../../ai-control-tabs";
import { Ic } from "../kit/ic";
import { ReadSheet } from "../kit/read-sheet";
import { KV } from "../kit/values";
import { ResolutionBadge, SeverityBadge } from "./agent-badges";
import { categoryLabel } from "./agent-exception-columns";
import { AgentRunSheet } from "./agent-run-sheet";
import { AgentSubjectCell } from "./agent-subject-cell";
import { Button } from "@trenova/shared/components/ui/button";

export type ExceptionActions = {
  canResolve: boolean;
  transition: (exception: AgentExceptionRow, state: AgentResolutionState) => void;
};

export const ExceptionActionsContext = createContext<ExceptionActions | null>(null);

const SETTLED: ReadonlySet<AgentResolutionState> = new Set(["Resolved", "Dismissed"]);

/**
 * One case an agent could not settle on its own: what it tried, how far it reaches, and
 * what was noted when someone took it. The case is taken, resolved or dismissed here, and
 * the run it came from opens beside it.
 */
export function ExceptionSheet({
  exception,
  onClose,
}: {
  exception: AgentExceptionRow | null;
  onClose: () => void;
}) {
  const t = useT();
  const actions = useContext(ExceptionActionsContext);
  const [runId, setRunId] = useQueryState(RUN_OPEN_PARAM, runOpenParser);
  // The run opens over its exception, so closing the exception closes the run with it.
  const close = () => {
    void setRunId(null);
    onClose();
  };
  const title = exception ? categoryLabel(exception.category) : t("Exception");
  const settled = exception ? SETTLED.has(exception.resolutionState) : true;

  return (
    <>
      <ReadSheet
        open={exception !== null}
        onClose={close}
        label={title}
        head={
          exception && (
            <>
              <span className="src-i">
                <Ic n="warn" s={15} />
              </span>
              <div className="sh-t">
                <b>{title}</b>
                <span>{formatUnixDateTimeMedium(exception.createdAt)}</span>
              </div>
              <ResolutionBadge value={exception.resolutionState} t={t} />
            </>
          )
        }
      >
        {exception && (
          <>
            <div className="sh-p">
              <h4 className="sh-k">{t("What happened")}</h4>
              <code className="sh-code">{exception.attemptSummary || "—"}</code>
            </div>
            <KV
              items={[
                [t("Severity"), <SeverityBadge key="severity" value={exception.severity} t={t} />],
                [t("Affected"), exception.blastRadius],
                [
                  t("Subject"),
                  <AgentSubjectCell
                    key="subject"
                    subjectType={exception.subjectType}
                    subjectId={exception.subjectId}
                  />,
                ],
                exception.resolutionNotes ? [t("Notes"), exception.resolutionNotes] : null,
              ]}
            />
            <div className="ad-bar sh-f">
              <button type="button" className="xa" onClick={() => void setRunId(exception.runId)}>
                <Ic n="timeline" s={13} />
                {t("Open the run")}
              </button>
              <span className="sp" />
              {actions?.canResolve && !settled && (
                <>
                  {exception.resolutionState === "Open" && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => actions.transition(exception, "InReview")}
                    >
                      {t("Mark in review")}
                    </Button>
                  )}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => actions.transition(exception, "Dismissed")}
                  >
                    {t("Dismiss")}
                  </Button>
                  <Button
                    type="button"
                    variant="default"
                    size="sm"
                    onClick={() => actions.transition(exception, "Resolved")}
                  >
                    {t("Mark resolved")}
                  </Button>
                </>
              )}
              {actions?.canResolve && settled && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => actions.transition(exception, "Open")}
                >
                  {t("Reopen")}
                </Button>
              )}
            </div>
          </>
        )}
      </ReadSheet>
      <AgentRunSheet runId={exception ? runId : null} onClose={() => void setRunId(null)} />
    </>
  );
}

/** The exceptions table's sheet: the row's case. */
export function ExceptionPanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<AgentExceptionRow>) {
  return <ExceptionSheet exception={open ? row : null} onClose={() => onOpenChange(false)} />;
}
