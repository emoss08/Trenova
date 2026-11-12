import { approvalFailure } from "@/lib/approval-outcome";
import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { decideAgentPlan, decideAgentProposal } from "@/lib/graphql/agent-decisions";
import { attention } from "@/lib/queries/attention";
import { deskActivity } from "@/lib/queries/desk-activity";
import { deskAttention } from "@/lib/queries/desk-attention";
import { watchtower } from "@/lib/queries/watchtower";
import { isPreviewConflict } from "@/components/assistant/proposal-preview/preview-gate";
import {
  reviewTally,
  useReviewReports,
  type ReviewTally,
} from "@/components/assistant/proposal-preview/reviewed-change";
import type { AssistantProposal } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useRef, useState } from "react";
import { useDecisionSteps } from "./use-decision-steps";
import {
  PENDING_DECISIONS_KEY,
  PENDING_DECISION_SUMMARY_KEY,
} from "../decisions/use-pending-decisions";

export type ItemDecisionPhase = "idle" | "deciding" | "done" | "failed" | "declined";

export type ItemDecision = {
  /** The changes the decision covers: one proposal, or a plan's steps in order. */
  proposals: AssistantProposal[];
  planId: string | null;
  loading: boolean;
  failedToRead: boolean;
  /** Whether the server approves it only with the digest of a preview the person was shown. */
  reviewed: boolean;
  onReport: ReturnType<typeof useReviewReports>["onReport"];
  tally: ReviewTally;
  phase: ItemDecisionPhase;
  /** Why it did not go through, in the server's words. */
  failure: string;
  decidedAt: number | null;
  approve: () => Promise<void>;
  decline: (reason: string) => Promise<void>;
  reset: () => void;
};

/** The changes an item of the Decisions lens stands for: its proposal, or every step of its plan. */
function useDecisionChanges(item: DeskAttentionItem | null) {
  return useDecisionSteps(
    item?.sourceKind === "AgentProposal"
      ? { proposalId: item.sourceId }
      : item?.sourceKind === "AgentPlan"
        ? { planId: item.sourceId }
        : null,
  );
}

/**
 * Deciding one change from Needs you, where it was raised: the same approval
 * the Decisions queue makes, carrying the digest of every preview the server
 * asks to be shown, and saying plainly when it did not go through. Nothing is
 * marked done until the server answers.
 */
export function useItemDecision(item: DeskAttentionItem | null): ItemDecision {
  const t = useT();
  const queryClient = useQueryClient();
  const changes = useDecisionChanges(item);
  const proposals = changes.data ?? [];
  const planId = item?.sourceKind === "AgentPlan" ? item.sourceId : null;
  const { reports, shown, onReport } = useReviewReports();
  const reviewed = proposals.some((proposal) => proposal.reviewRequired);
  const reviewIds = reviewed ? (planId ? [planId] : proposals.map((proposal) => proposal.id)) : [];
  const tally = reviewTally(reviewIds, reports);

  const [phase, setPhase] = useState<ItemDecisionPhase>("idle");
  const [failure, setFailure] = useState("");
  const [decidedAt, setDecidedAt] = useState<number | null>(null);
  const [seenItem, setSeenItem] = useState(item?.id ?? null);
  const sending = useRef(false);
  if ((item?.id ?? null) !== seenItem) {
    setSeenItem(item?.id ?? null);
    setPhase("idle");
    setFailure("");
    setDecidedAt(null);
  }

  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: deskAttention._def }),
      queryClient.invalidateQueries({ queryKey: deskActivity._def }),
      queryClient.invalidateQueries({ queryKey: watchtower._def }),
      queryClient.invalidateQueries({ queryKey: attention._def }),
      queryClient.invalidateQueries({ queryKey: [PENDING_DECISIONS_KEY] }),
      queryClient.invalidateQueries({ queryKey: [PENDING_DECISION_SUMMARY_KEY] }),
    ]);

  const decide = async (accepted: boolean, reason: string) => {
    const target = proposals[0];
    if (!item || !target || sending.current) {
      return;
    }
    sending.current = true;
    setPhase("deciding");
    const input = {
      decision: accepted ? ("Accepted" as const) : ("Rejected" as const),
      reasonCode: accepted ? "approved_from_needs" : "declined_from_needs",
      note: accepted ? undefined : reason,
    };
    try {
      if (planId) {
        const { plan, refused } = await decideAgentPlan(planId, {
          ...input,
          previewDigest: accepted ? shown.get(planId) : undefined,
        });
        if (accepted && plan.status === "Failed") {
          setFailure(
            refused
              ? t("It was refused because the record changed: {0}", plan.failureError)
              : plan.failureError,
          );
          setPhase("failed");
        } else {
          setPhase(accepted ? "done" : "declined");
        }
      } else {
        const result = await decideAgentProposal(target.id, {
          ...input,
          previewDigest: accepted ? shown.get(target.id) : undefined,
        });
        const why = approvalFailure(result);
        if (accepted && why !== null) {
          setFailure(why);
          setPhase("failed");
        } else {
          setPhase(accepted ? "done" : "declined");
        }
      }
      setDecidedAt(Math.floor(Date.now() / 1000));
      void refresh();
    } catch (error) {
      setFailure(
        isPreviewConflict(error)
          ? t("This changed since you opened it. Look at it again before you approve.")
          : error instanceof Error
            ? error.message
            : t("The decision didn't go through."),
      );
      setPhase("failed");
    } finally {
      sending.current = false;
    }
  };

  return {
    proposals,
    planId,
    loading: changes.isPending && changes.fetchStatus !== "idle",
    failedToRead: changes.isError,
    reviewed,
    onReport,
    tally,
    phase,
    failure,
    decidedAt,
    approve: () => decide(true, ""),
    decline: (reason) => decide(false, reason),
    reset: () => {
      setPhase("idle");
      setFailure("");
    },
  };
}
