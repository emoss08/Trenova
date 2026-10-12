import type { AgentProposalRow } from "@/lib/graphql/agent-activity-tables";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useQueryState } from "nuqs";
import { createContext, useContext } from "react";
import { RUN_OPEN_PARAM, runOpenParser } from "../../ai-control-tabs";
import { Ic } from "../kit/ic";
import { ReadSheet } from "../kit/read-sheet";
import { KV } from "../kit/values";
import { ProposalStatusBadge, TierBadge } from "./agent-badges";
import { AgentRunSheet } from "./agent-run-sheet";
import { Button } from "@trenova/shared/components/ui/button";

export type ProposalActions = {
  canDecide: boolean;
  approve: (proposal: AgentProposalRow) => void;
  modify: (proposal: AgentProposalRow) => void;
  reject: (proposal: AgentProposalRow) => void;
};

export const ProposalActionsContext = createContext<ProposalActions | null>(null);

/**
 * One proposed change, read: what the agent wants to do and why, how sure it is, and at
 * what autonomy; a warning when the run read outside text first. A waiting proposal is
 * approved, approved with changes, or rejected here, through the same dialogs as the
 * row's menu, and the run that proposed it opens beside it.
 */
export function ProposalSheet({
  proposal,
  onClose,
}: {
  proposal: AgentProposalRow | null;
  onClose: () => void;
}) {
  const t = useT();
  const actions = useContext(ProposalActionsContext);
  const [runId, setRunId] = useQueryState(RUN_OPEN_PARAM, runOpenParser);
  // The run opens over its proposal, so closing the proposal closes the run with it.
  const close = () => {
    void setRunId(null);
    onClose();
  };
  const pending = proposal?.status === "Pending";
  const title = proposal ? proposal.rationale || proposal.toolName : t("Proposal");

  return (
    <>
      <ReadSheet
        open={proposal !== null}
        onClose={close}
        label={title}
        head={
          proposal && (
            <>
              <span className="src-i">
                <Ic n="tool" s={15} />
              </span>
              <div className="sh-t">
                <b>{title}</b>
                <span className="mono">{proposal.toolName}</span>
              </div>
              <ProposalStatusBadge value={proposal.status} t={t} />
            </>
          )
        }
      >
        {proposal && (
          <>
            <KV
              items={[
                [t("Autonomy"), <TierBadge key="tier" value={proposal.autonomyTier} t={t} />],
                [t("Confidence"), `${Math.round(Number(proposal.confidence) * 100)}%`],
                [
                  t("Run"),
                  <button
                    key="run"
                    type="button"
                    className="lnk mono"
                    onClick={() => void setRunId(proposal.runId)}
                  >
                    {proposal.runId}
                  </button>,
                ],
                [t("Proposed"), formatUnixDateTimeMedium(proposal.createdAt)],
              ]}
            />
            {proposal.tainted && (
              <div className="sh-p">
                <div className="ah-w nm">
                  <Ic n="warn" s={13} />
                  <span>
                    {t(
                      "The run read an email or document written outside the organization before proposing this. Check the details before approving.",
                    )}
                  </span>
                </div>
              </div>
            )}
            <div className="ad-bar sh-f">
              {pending && actions?.canDecide ? (
                <>
                  <Button
                    type="button"
                    variant="default"
                    size="sm"
                    onClick={() => actions.approve(proposal)}
                  >
                    <Ic n="check" s={12} w={2.2} />
                    {t("Approve")}
                  </Button>
                  {proposal.parameterFields.length > 0 && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => actions.modify(proposal)}
                    >
                      {t("Approve with changes")}
                    </Button>
                  )}
                  <Button type="button" variant="outline" size="sm" onClick={() => actions.reject(proposal)}>
                    {t("Reject")}
                  </Button>
                </>
              ) : (
                !pending && (
                  <span className="ad-h">
                    {t("Decided {0}", formatUnixDateTimeMedium(proposal.updatedAt))}
                  </span>
                )
              )}
              <span className="sp" />
              <button type="button" className="xa" onClick={() => void setRunId(proposal.runId)}>
                <Ic n="timeline" s={13} />
                {t("Open the run")}
              </button>
            </div>
          </>
        )}
      </ReadSheet>
      <AgentRunSheet runId={proposal ? runId : null} onClose={() => void setRunId(null)} />
    </>
  );
}

/** The proposals table's sheet: the row's proposal. */
export function ProposalPanel({ open, onOpenChange, row }: DataTablePanelProps<AgentProposalRow>) {
  return <ProposalSheet proposal={open ? row : null} onClose={() => onOpenChange(false)} />;
}
