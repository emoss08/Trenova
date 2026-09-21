import { ProposalCard } from "@/components/assistant/proposal-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
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
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
      <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-sm">
        <dt className="text-muted-foreground">{t("Kind")}</dt>
        <dd>{humanizeToolName(draft.tool)}</dd>
        {draft.to.length > 0 && (
          <>
            <dt className="text-muted-foreground">{t("To")}</dt>
            <dd className="break-words">{draft.to.join(", ")}</dd>
          </>
        )}
        {draft.subject !== "" && (
          <>
            <dt className="text-muted-foreground">{t("Subject")}</dt>
            <dd className="font-medium">{draft.subject}</dd>
          </>
        )}
      </dl>

      {draft.body !== "" ? (
        <div className="bg-sunken rounded-md px-4 py-3 text-sm leading-relaxed whitespace-pre-wrap">
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
