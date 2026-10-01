import { decisionRequestOf } from "@/components/assistant/decision-requests";
import { RequestedDecisionRecords } from "@/components/assistant/decision-record";
import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * A decision the assistant asked the person to make. The decision lives on
 * the proposal and is made in the approval box under the conversation, so
 * this keeps the proposal's record, which reads as decided the moment it is,
 * wherever that happened, and while it waits offers the way to the box. One
 * kept for a plan, or for several proposals of one tool, shows them as the
 * thread does.
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
      <RequestedDecisionRecords request={request} threadId={artifact.threadId} />
    </div>
  );
}
