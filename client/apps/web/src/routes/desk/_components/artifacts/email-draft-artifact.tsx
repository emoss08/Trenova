import { ProposalCard } from "@/components/assistant/proposal-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useMemo } from "react";
import { emailDraftFrom } from "./artifact-payloads";

/**
 * An outbound message as a draft a person can read whole: who it goes to,
 * the subject, the body. The decision lives on the proposal it views, so
 * approving, changing and rejecting are the proposal's own card under the
 * draft, and a draft sent from anywhere reads as sent here.
 */
export function EmailDraftArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const draft = useMemo(() => emailDraftFrom(artifact), [artifact]);
  const proposalsQuery = useQuery({
    ...queries.assistant.proposals(artifact.threadId),
    enabled: artifact.proposalId !== "",
  });
  const proposal =
    proposalsQuery.data?.results.find((candidate) => candidate.id === artifact.proposalId) ?? null;

  return (
    <div className="animate-rise flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
      <DescriptionList layout="inline">
        <DescriptionItem label={t("Kind")}>{humanizeToolName(draft.tool)}</DescriptionItem>
        {draft.to.length > 0 && (
          <DescriptionItem label={t("To")}>
            <span className="break-words">{draft.to.join(", ")}</span>
          </DescriptionItem>
        )}
        {draft.subject !== "" && (
          <DescriptionItem label={t("Subject")}>{draft.subject}</DescriptionItem>
        )}
      </DescriptionList>

      {draft.body !== "" ? (
        <div className="bg-sunken rounded-lg px-4 py-3 text-sm leading-relaxed whitespace-pre-wrap">
          {draft.body}
        </div>
      ) : (
        <p className="text-muted-foreground text-sm">
          {t("The message is written from the organization's template when it is sent.")}
        </p>
      )}

      {draft.rationale !== "" && (
        <p className="text-muted-foreground text-xs leading-relaxed">{draft.rationale}</p>
      )}

      {proposalsQuery.isLoading ? (
        <Skeleton className="h-20" />
      ) : proposal ? (
        <ProposalCard proposal={proposal} threadId={artifact.threadId} />
      ) : (
        <p className="text-muted-foreground text-xs">
          {t("The proposal behind this draft is no longer in the conversation.")}
        </p>
      )}
    </div>
  );
}
