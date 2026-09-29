import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { decideMyProposals } from "@/lib/graphql/agent-decisions";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import { invalidateProposalViews, markProposalDecided } from "@/lib/proposal-cache";
import { queries } from "@/lib/queries";
import type { AssistantProposal } from "@/types/assistant";
import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { CheckIcon, ListChecksIcon } from "lucide-react";
import { useDecisionFollowUp } from "./decision-follow-up";
import { batchPreviewDigests } from "./proposal-preview/preview-gate";
import { presentProposal } from "./proposal-presenters";
import { classifyProposal } from "./proposal-state";

type BatchOutcome = {
  approved: number;
  total: number;
  failures: { proposalId: string; summary: string; error: string }[];
};

/** The digest of each preview this person has on screen, read from the cache the cards fill. */
function shownDigests(
  queryClient: QueryClient,
  proposals: readonly AssistantProposal[],
): Map<string, string> {
  const shown = new Map<string, string>();
  for (const proposal of proposals) {
    const preview = queryClient.getQueryData<ProposalPreview>(
      queries.agentPreview.proposal("mine", proposal.id).queryKey,
    );
    if (preview?.digest) {
      shown.set(proposal.id, preview.digest);
    }
  }

  return shown;
}

/**
 * One answer for several waiting changes of the same kind.
 *
 * Five invoices to post used to be five cards approved one at a time. When
 * more than one proposal of one tool waits in the conversation, this offers to
 * approve them together. Each still runs as its own decision, carrying the
 * digest of the preview its card showed; one the person never had on screen
 * goes without one and is recorded as approved unreviewed. What did not go
 * through is listed with the reason each card also shows.
 */
export function ProposalBatchBar({
  proposals,
  threadId,
}: {
  proposals: AssistantProposal[];
  threadId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const followUp = useDecisionFollowUp();
  const waiting = proposals.filter((proposal) => classifyProposal(proposal) === "awaiting");
  const tools = new Set(waiting.map((proposal) => proposal.toolName));

  const approveMutation = useMutation({
    mutationFn: async (batch: AssistantProposal[]): Promise<BatchOutcome> => {
      const ids = batch.map((proposal) => proposal.id);
      const results = await decideMyProposals(ids, {
        decision: "Accepted",
        previewDigests: batchPreviewDigests(ids, shownDigests(queryClient, batch)),
      });
      const byId = new Map(batch.map((proposal) => [proposal.id, proposal]));

      return {
        approved: results.filter((result) => result.executed).length,
        total: results.length,
        failures: results
          .filter((result) => (result.error ?? "") !== "")
          .map((result) => {
            const proposal = byId.get(result.proposalId);
            return {
              proposalId: result.proposalId,
              summary: proposal ? presentProposal(proposal).summary : "",
              error: result.error ?? "",
            };
          }),
      };
    },
    onSuccess: async (outcome, batch) => {
      const failed = new Set(outcome.failures.map((failure) => failure.proposalId));
      for (const proposal of batch) {
        if (!failed.has(proposal.id)) {
          markProposalDecided(queryClient, proposal.id, "Accepted");
        }
      }
      await invalidateProposalViews(queryClient, threadId);
      const first = batch.find((proposal) => !failed.has(proposal.id));
      if (first) {
        followUp?.(first.id);
      }
    },
    onError: (error) => {
      handleMutationError({ error, resourceName: "Proposals" });
      void invalidateProposalViews(queryClient, threadId);
    },
  });

  const outcome = approveMutation.data;
  if (outcome === undefined && (waiting.length < 2 || tools.size !== 1)) {
    return null;
  }

  const title = presentProposal(waiting[0] ?? proposals[0]).title;

  return (
    <div className="border-border-subtle flex flex-col gap-2 rounded-lg border px-3 py-2.5">
      {outcome === undefined ? (
        <div className="flex flex-wrap items-center gap-2">
          <ListChecksIcon className="text-foreground-muted size-3.5 shrink-0" />
          <span className="text-foreground-muted min-w-0 flex-1 text-xs">
            {t(
              "{0, plural, one {# change is waiting} other {# changes are waiting}}: {1}",
              waiting.length,
              title,
            )}
          </span>
          <Button
            size="sm"
            onClick={() => approveMutation.mutate(waiting)}
            disabled={approveMutation.isPending}
            isLoading={approveMutation.isPending}
          >
            <CheckIcon className="size-3.5" />
            {t("Approve all {0}", waiting.length)}
          </Button>
        </div>
      ) : (
        <BatchOutcomeLine outcome={outcome} />
      )}
    </div>
  );
}

function BatchOutcomeLine({ outcome }: { outcome: BatchOutcome }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-1.5 text-xs">
      <span>{t("{0} of {1} approved.", outcome.approved, outcome.total)}</span>
      {outcome.failures.length > 0 && (
        <Alert size="sm" variant="destructive">
          <AlertDescription>
            <ul className="flex flex-col gap-0.5">
              {outcome.failures.map((failure) => (
                <li key={failure.proposalId}>
                  {failure.summary} {failure.error}
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}
