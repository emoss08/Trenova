import { Button } from "@trenova/shared/components/ui/button";
import { decisionRequestOf } from "@/components/assistant/decision-requests";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { decideMyPlan, decideMyProposal, decideMyProposals } from "@/lib/graphql/agent-decisions";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { queries } from "@/lib/queries";
import type { AssistantArtifact, AssistantProposal } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { Notice } from "../artifact-body-parts";

/** Where a decision stands, read from the proposals or plan behind it. */
function decisionState(statuses: readonly string[]): "pending" | "approved" | "dismissed" {
  if (statuses.length === 0 || statuses.some((status) => status === "Pending")) {
    return "pending";
  }
  return statuses.some((status) => status === "Rejected" || status === "Expired")
    ? "dismissed"
    : "approved";
}

const DECISION_ROWS = 8;

/**
 * A decision the agent asked for: what it covers, each change, and the way to
 * make it here. Approving and setting aside go through the same decision
 * mutations as the approval box, and the state follows the proposals however
 * they are decided.
 */
export function DeskDecisionBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const queryClient = useQueryClient();
  const request = decisionRequestOf({ proposalId: artifact.proposalId, ...artifact.payload });
  const proposalsQuery = useQuery(queries.assistant.proposals(artifact.threadId));
  const plansQuery = useQuery({
    ...queries.assistant.plans(artifact.threadId),
    enabled: request !== null && request.planId !== "",
  });
  const all = proposalsQuery.data?.results ?? [];
  const plan =
    request?.planId !== ""
      ? (plansQuery.data?.results.find((candidate) => candidate.id === request?.planId) ?? null)
      : null;
  const proposals =
    request === null
      ? []
      : plan !== null
        ? all
            .filter((proposal) => proposal.planId === plan.id)
            .sort((a, b) => a.planStep - b.planStep)
        : request.proposalIds
            .map((id) => all.find((candidate) => candidate.id === id))
            .filter((proposal): proposal is AssistantProposal => proposal !== undefined);
  const state =
    plan !== null
      ? plan.status === "Pending"
        ? "pending"
        : plan.status === "Rejected" || plan.status === "Expired"
          ? "dismissed"
          : "approved"
      : decisionState(proposals.map((proposal) => proposal.status));
  const ids = proposals
    .filter((proposal) => proposal.status === "Pending")
    .map((proposal) => proposal.id);

  const decide = useApiMutation({
    mutationFn: async (decision: "Accepted" | "Rejected") => {
      const input = { decision, reasonCode: decision === "Rejected" ? "not_now" : "" };
      if (plan !== null) {
        await decideMyPlan(plan.id, input);
      } else if (ids.length === 1) {
        await decideMyProposal(ids[0], input);
      } else {
        await decideMyProposals(ids, input);
      }
    },
    onSuccess: () => invalidateProposalViews(queryClient, artifact.threadId),
    resourceName: "Decision",
  });

  if (artifact.proposalId === "" || request === null) {
    return <Notice>{t("This decision no longer names a proposal.")}</Notice>;
  }
  if (proposalsQuery.isPending) {
    return <Notice>{t("Loading…")}</Notice>;
  }

  const first = proposals[0];
  const view = first ? presentProposal(first) : null;
  const scope = [
    view?.summary ?? "",
    first ? humanizeToolName(first.toolName) : "",
    t("{0, plural, one {# change} other {# changes}}", proposals.length),
  ].filter(Boolean);

  return (
    <div className={cn("dk-ax-pad dk-ax-dec", `dk-s-${state}`)}>
      <div className="dk-ax-dech">
        <b>{artifact.title}</b>
        <span className={cn("dk-ax-dst", state !== "pending" && `dk-${state}`)}>
          {state === "pending"
            ? t("Waiting on you")
            : state === "approved"
              ? t("Approved")
              : t("Set aside")}
        </span>
      </div>
      <p className="dk-ax-note">{scope.join(" · ")}</p>
      <div className="dk-ax-decl">
        {proposals.slice(0, DECISION_ROWS).map((proposal) => {
          const shown = presentProposal(proposal);
          const [lead, change] = shown.highlights;
          return (
            <div key={proposal.id}>
              <span className="dk-ax-id">{lead ? String(lead.value) : proposal.id.slice(-8)}</span>
              <span>{shown.title}</span>
              <s>{change ? change.label : ""}</s>
              <i>→</i>
              <em>{change ? String(change.value) : humanizeToolName(proposal.status)}</em>
            </div>
          );
        })}
        {proposals.length > DECISION_ROWS && (
          <div className="dk-ax-decm">{t("+ {0} more", proposals.length - DECISION_ROWS)}</div>
        )}
      </div>
      {state === "pending" && ids.length + (plan !== null ? 1 : 0) > 0 && (
        <div className="dk-ax-acts">
          <Button
            variant="quiet"
            disabled={decide.isPending}
            onClick={() => decide.mutate("Rejected")}
          >
            {t("Not now")}
          </Button>
          <span className="flex-1" />
          <Button disabled={decide.isPending} onClick={() => decide.mutate("Accepted")}>
            {t("{0, plural, one {Approve # change} other {Approve # changes}}", proposals.length)}
          </Button>
        </div>
      )}
    </div>
  );
}
