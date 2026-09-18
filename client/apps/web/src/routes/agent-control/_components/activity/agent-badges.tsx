import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  AgentAutonomyTier,
  AgentProposalStatus,
  AgentResolutionState,
  AgentRunStatus,
  AgentRunTrigger,
  AgentSeverity,
  AgentType,
} from "@trenova/graphql/generated/graphql";
import type { ComponentProps } from "react";

type Variant = NonNullable<ComponentProps<typeof Badge>["variant"]>;

const RUN_STATUS: Record<AgentRunStatus, { label: string; variant: Variant }> = {
  Pending: { label: "Pending", variant: "neutral" },
  GatheringContext: { label: "Gathering context", variant: "info" },
  Diagnosing: { label: "Working", variant: "info" },
  AwaitingDecision: { label: "Awaiting decision", variant: "warning" },
  Completed: { label: "Completed", variant: "success" },
  ShadowCompleted: { label: "Completed in shadow", variant: "info" },
  Failed: { label: "Failed", variant: "danger" },
};

const PROPOSAL_STATUS: Record<AgentProposalStatus, { label: string; variant: Variant }> = {
  Pending: { label: "Awaiting decision", variant: "warning" },
  Accepted: { label: "Accepted", variant: "success" },
  Modified: { label: "Accepted with changes", variant: "info" },
  Rejected: { label: "Rejected", variant: "danger" },
  Expired: { label: "Expired", variant: "neutral" },
  Superseded: { label: "Superseded", variant: "neutral" },
};

const TIER: Record<AgentAutonomyTier, { label: string; variant: Variant }> = {
  Propose: { label: "Proposes only", variant: "neutral" },
  ActWithApproval: { label: "Acts with approval", variant: "info" },
  AutoExecute: { label: "Acts on its own", variant: "brand" },
};

const TRIGGER: Record<AgentRunTrigger, { label: string; variant: Variant }> = {
  Manual: { label: "Manual", variant: "neutral" },
  Chat: { label: "Chat", variant: "info" },
  Scheduled: { label: "Scheduled", variant: "accent-teal" },
  Event: { label: "Event", variant: "accent-amber" },
  Continuous: { label: "Continuous", variant: "accent-violet" },
};

const AGENT_TYPE: Record<AgentType, string> = {
  BillingException: "Billing exception",
  DispatchAssignment: "Dispatch assignment",
  AssistantChat: "Assistant chat",
  General: "General",
};

const SEVERITY: Record<AgentSeverity, { label: string; variant: Variant }> = {
  Low: { label: "Low", variant: "neutral" },
  Medium: { label: "Medium", variant: "info" },
  High: { label: "High", variant: "warning" },
  Critical: { label: "Critical", variant: "danger" },
};

const RESOLUTION: Record<AgentResolutionState, { label: string; variant: Variant }> = {
  Open: { label: "Open", variant: "warning" },
  InReview: { label: "In review", variant: "info" },
  Resolved: { label: "Resolved", variant: "success" },
  Dismissed: { label: "Dismissed", variant: "neutral" },
};

function Labelled({
  entry,
  t,
}: {
  entry: { label: string; variant: Variant } | undefined;
  t: TranslateFn;
}) {
  if (!entry) {
    return null;
  }
  return <Badge variant={entry.variant}>{t(entry.label)}</Badge>;
}

export const RunStatusBadge = ({ value, t }: { value: AgentRunStatus; t: TranslateFn }) => (
  <Labelled entry={RUN_STATUS[value]} t={t} />
);
export const ProposalStatusBadge = ({
  value,
  t,
}: {
  value: AgentProposalStatus;
  t: TranslateFn;
}) => <Labelled entry={PROPOSAL_STATUS[value]} t={t} />;
export const TierBadge = ({ value, t }: { value: AgentAutonomyTier; t: TranslateFn }) => (
  <Labelled entry={TIER[value]} t={t} />
);
export const TriggerBadge = ({ value, t }: { value: AgentRunTrigger; t: TranslateFn }) => (
  <Labelled entry={TRIGGER[value]} t={t} />
);
export const SeverityBadge = ({ value, t }: { value: AgentSeverity; t: TranslateFn }) => (
  <Labelled entry={SEVERITY[value]} t={t} />
);
export const ResolutionBadge = ({ value, t }: { value: AgentResolutionState; t: TranslateFn }) => (
  <Labelled entry={RESOLUTION[value]} t={t} />
);

export function agentTypeLabel(value: AgentType, t: TranslateFn): string {
  return t(AGENT_TYPE[value] ?? value);
}

export const runStatusChoices = Object.entries(RUN_STATUS).map(([value, entry]) => ({
  value,
  label: entry.label,
}));
export const proposalStatusChoices = Object.entries(PROPOSAL_STATUS).map(([value, entry]) => ({
  value,
  label: entry.label,
}));
export const triggerChoices = Object.entries(TRIGGER).map(([value, entry]) => ({
  value,
  label: entry.label,
}));
export const severityChoices = Object.entries(SEVERITY).map(([value, entry]) => ({
  value,
  label: entry.label,
}));
export const resolutionChoices = Object.entries(RESOLUTION).map(([value, entry]) => ({
  value,
  label: entry.label,
}));
