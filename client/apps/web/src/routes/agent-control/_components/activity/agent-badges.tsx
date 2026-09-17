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
  Pending: { label: "Pending", variant: "outline" },
  GatheringContext: { label: "Gathering context", variant: "info" },
  Diagnosing: { label: "Working", variant: "info" },
  AwaitingDecision: { label: "Awaiting decision", variant: "warning" },
  Completed: { label: "Completed", variant: "active" },
  ShadowCompleted: { label: "Completed in shadow", variant: "purple" },
  Failed: { label: "Failed", variant: "inactive" },
};

const PROPOSAL_STATUS: Record<AgentProposalStatus, { label: string; variant: Variant }> = {
  Pending: { label: "Awaiting decision", variant: "warning" },
  Accepted: { label: "Accepted", variant: "active" },
  Modified: { label: "Accepted with changes", variant: "teal" },
  Rejected: { label: "Rejected", variant: "inactive" },
  Expired: { label: "Expired", variant: "outline" },
  Superseded: { label: "Superseded", variant: "outline" },
};

const TIER: Record<AgentAutonomyTier, { label: string; variant: Variant }> = {
  Propose: { label: "Proposes only", variant: "outline" },
  ActWithApproval: { label: "Acts with approval", variant: "indigo" },
  AutoExecute: { label: "Acts on its own", variant: "purple" },
};

const TRIGGER: Record<AgentRunTrigger, { label: string; variant: Variant }> = {
  Manual: { label: "Manual", variant: "outline" },
  Chat: { label: "Chat", variant: "info" },
  Scheduled: { label: "Scheduled", variant: "teal" },
  Event: { label: "Event", variant: "orange" },
  Continuous: { label: "Continuous", variant: "pink" },
};

const AGENT_TYPE: Record<AgentType, string> = {
  BillingException: "Billing exception",
  DispatchAssignment: "Dispatch assignment",
  AssistantChat: "Assistant chat",
  General: "General",
};

const SEVERITY: Record<AgentSeverity, { label: string; variant: Variant }> = {
  Low: { label: "Low", variant: "outline" },
  Medium: { label: "Medium", variant: "info" },
  High: { label: "High", variant: "warning" },
  Critical: { label: "Critical", variant: "inactive" },
};

const RESOLUTION: Record<AgentResolutionState, { label: string; variant: Variant }> = {
  Open: { label: "Open", variant: "warning" },
  InReview: { label: "In review", variant: "info" },
  Resolved: { label: "Resolved", variant: "active" },
  Dismissed: { label: "Dismissed", variant: "outline" },
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
