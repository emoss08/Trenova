import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentPlanPreviewDocument,
  AgentPlanPreviewFieldsFragmentDoc,
  AgentProposalPreviewDocument,
  AgentProposalPreviewFieldsFragmentDoc,
  MyPlanPreviewDocument,
  MyProposalPreviewDocument,
  type AgentPlanPreviewFieldsFragment,
  type AgentPreviewCoverage,
  type AgentPreviewMessageChannel,
  type AgentPreviewOperation,
  type AgentProposalPreviewFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type { AgentPreviewCoverage, AgentPreviewMessageChannel, AgentPreviewOperation };

/** What a proposal's write would do, as the server computed it for the reader. */
export type ProposalPreview = AgentProposalPreviewFieldsFragment;
export type PreviewRecordChange = ProposalPreview["changes"][number];
export type PreviewFieldChange = PreviewRecordChange["fields"][number];
export type PreviewRef = NonNullable<PreviewFieldChange["afterRef"]>;
export type PreviewRecordLink = NonNullable<PreviewRecordChange["record"]>;
export type PreviewMessage = NonNullable<PreviewRecordChange["message"]>;
export type PreviewMoney = NonNullable<PreviewRecordChange["money"]>;
export type PreviewWarning = ProposalPreview["warnings"][number];
export type PreviewReason = PreviewWarning["reasons"][number];

export type PlanStepPreview = {
  proposalId: string;
  step: number;
  preview: ProposalPreview;
};

/** What every pending step of a plan would do, in order, with the one digest that covers them. */
export type PlanPreview = Omit<AgentPlanPreviewFieldsFragment, "steps" | " $fragmentName"> & {
  steps: PlanStepPreview[];
};

/**
 * Who is asking. A person answering in their own conversation reads the
 * self-scoped preview, which needs only the assistant; an approver in the
 * decisions queue or AI Control reads the one that needs the proposal right.
 * The two agree for the same reader, but each surface asks through the door
 * its decision goes through.
 */
export type PreviewScope = "mine" | "approver";

type RequestOptions = { signal?: AbortSignal };

export type ProposalPreviewRequest = {
  scope: PreviewScope;
  id: string;
  /** The values an approver has in mind, over what was proposed; null previews it as proposed. */
  modifications: Record<string, unknown> | null;
};

export async function fetchProposalPreview(
  { scope, id, modifications }: ProposalPreviewRequest,
  { signal }: RequestOptions = {},
): Promise<ProposalPreview> {
  if (scope === "mine") {
    const data = await requestGraphQL({
      document: MyProposalPreviewDocument,
      operationName: "MyProposalPreview",
      variables: { id, modifications },
      signal,
    });

    return getFragmentData(AgentProposalPreviewFieldsFragmentDoc, data.myProposalPreview);
  }

  const data = await requestGraphQL({
    document: AgentProposalPreviewDocument,
    operationName: "AgentProposalPreview",
    variables: { id, modifications },
    signal,
  });

  return getFragmentData(AgentProposalPreviewFieldsFragmentDoc, data.agentProposalPreview);
}

export async function fetchPlanPreview(
  { scope, id }: { scope: PreviewScope; id: string },
  { signal }: RequestOptions = {},
): Promise<PlanPreview> {
  const masked =
    scope === "mine"
      ? (
          await requestGraphQL({
            document: MyPlanPreviewDocument,
            operationName: "MyPlanPreview",
            variables: { id },
            signal,
          })
        ).myPlanPreview
      : (
          await requestGraphQL({
            document: AgentPlanPreviewDocument,
            operationName: "AgentPlanPreview",
            variables: { id },
            signal,
          })
        ).agentPlanPreview;
  const plan = getFragmentData(AgentPlanPreviewFieldsFragmentDoc, masked);

  return {
    planId: plan.planId,
    digest: plan.digest,
    stale: plan.stale,
    withheldCount: plan.withheldCount,
    computedAt: plan.computedAt,
    steps: plan.steps.map((step) => ({
      proposalId: step.proposalId,
      step: step.step,
      preview: getFragmentData(AgentProposalPreviewFieldsFragmentDoc, step.preview),
    })),
  };
}
