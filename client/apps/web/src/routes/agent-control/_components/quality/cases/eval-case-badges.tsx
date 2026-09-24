import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  phaseTone,
  type BadgeAttrProps,
  type BadgeClassAttrProps,
} from "@trenova/shared/lib/status-phase";
import type { AgentEvalCaseSource, AgentEvalCaseStatus } from "@trenova/graphql/generated/graphql";

const CASE_STATUS: Record<AgentEvalCaseStatus, BadgeAttrProps> = {
  Candidate: {
    phase: "awaiting",
    text: "Candidate",
    description: "Captured and waiting for an administrator to activate it",
  },
  Active: { phase: "active", text: "Active", description: "Replayed in every suite" },
  Quarantined: {
    phase: "attention",
    text: "Quarantined",
    description: "Kept but not replayed until someone looks at it",
  },
  Retired: { phase: "closed", text: "Retired", description: "Kept for its history only" },
};

const CASE_SOURCE: Record<AgentEvalCaseSource, BadgeClassAttrProps> = {
  DecidedProposal: { accent: "accent-teal", text: "Decided proposal" },
  ThumbsUp: { accent: "accent-violet", text: "Liked reply" },
  Curated: { accent: "accent-indigo", text: "Written by hand" },
};

export const CASE_STATUS_ACTION: Record<AgentEvalCaseStatus, string> = {
  Candidate: "Move back to candidates",
  Active: "Activate",
  Quarantined: "Quarantine",
  Retired: "Retire",
};

export const CASE_STATUS_MOVED: Record<AgentEvalCaseStatus, string> = {
  Candidate: "Case moved back to candidates",
  Active: "Case activated",
  Quarantined: "Case quarantined",
  Retired: "Case retired",
};

export function EvalCaseStatusBadge({ value, t }: { value: AgentEvalCaseStatus; t: TranslateFn }) {
  const attrs = CASE_STATUS[value];

  return (
    <Badge
      variant={phaseTone(attrs.phase)}
      title={attrs.description ? t(attrs.description) : undefined}
    >
      {t(attrs.text)}
    </Badge>
  );
}

export function EvalCaseSourceBadge({ value, t }: { value: AgentEvalCaseSource; t: TranslateFn }) {
  const attrs = CASE_SOURCE[value];

  return <Badge variant={attrs.accent}>{t(attrs.text)}</Badge>;
}

export const evalCaseStatusChoices = Object.entries(CASE_STATUS).map(([value, attrs]) => ({
  value,
  label: attrs.text,
}));

export const evalCaseSourceChoices = Object.entries(CASE_SOURCE).map(([value, attrs]) => ({
  value,
  label: attrs.text,
}));
