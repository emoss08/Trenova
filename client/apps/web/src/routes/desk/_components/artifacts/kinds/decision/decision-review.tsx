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
import { useEffect } from "react";

type Shown = (id: string, digest: string) => void;

/**
 * The preview of one change the server approves only once it has been shown.
 * It reports its digest when it is on screen and current; a preview of a record
 * that moved since is shown but never counts as reviewed, since the server
 * refuses it.
 */
export function ReviewedChange({
  proposal,
  onShown,
}: {
  proposal: AssistantProposal;
  onShown: Shown;
}) {
  const t = useT();
  const query = useProposalPreview({ scope: "mine", id: proposal.id });
  const digest = query.data && !query.data.stale ? query.data.digest : "";
  useEffect(() => {
    if (digest !== "") {
      onShown(proposal.id, digest);
    }
  }, [digest, onShown, proposal.id]);

  return (
    <section
      className="flex min-w-0 flex-col gap-2"
      aria-label={t("Preview of {0}", t(presentProposal(proposal).title))}
    >
      <b className="text-sm font-medium">{t(presentProposal(proposal).title)}</b>
      <PreviewLoadState query={query} changed={false} density="compact">
        {(preview) => <ProposalPreview preview={preview} density="compact" />}
      </PreviewLoadState>
    </section>
  );
}

/** The preview of a plan with a step the server approves only once it has been shown. */
export function ReviewedPlan({ planId, onShown }: { planId: string; onShown: Shown }) {
  const query = usePlanPreview({ scope: "mine", id: planId });
  const digest = query.data && !query.data.stale ? query.data.digest : "";
  useEffect(() => {
    if (digest !== "") {
      onShown(planId, digest);
    }
  }, [digest, onShown, planId]);

  return (
    <PreviewLoadState query={query} changed={false} density="compact">
      {(plan) => <PlanPreview plan={plan} density="compact" />}
    </PreviewLoadState>
  );
}
