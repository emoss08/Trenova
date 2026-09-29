import { decisionRequestOf } from "@/components/assistant/decision-requests";
import { RequestedDecision } from "@/components/assistant/requested-decision";
import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * A proposal the assistant put back in front of the person to decide. The
 * decision lives on the proposal, so this is the proposal's own card, and
 * reads as decided the moment it is, wherever that happened. A card kept for
 * a plan, or for several proposals of one tool, shows them as the thread does.
 */
export function DecisionRequestArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const request = decisionRequestOf({ proposalId: artifact.proposalId, ...artifact.payload });

  if (artifact.proposalId === "" || request === null) {
    return (
      <ArtifactNotice kind={artifact.kind}>
        {t("This decision no longer names a proposal.")}
      </ArtifactNotice>
    );
  }

  return (
    <div className="animate-rise flex min-h-0 flex-1 flex-col overflow-y-auto p-4">
      <RequestedDecision request={request} threadId={artifact.threadId} />
    </div>
  );
}
