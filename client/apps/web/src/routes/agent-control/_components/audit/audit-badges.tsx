import type { AiAuditEventOutcome, AiAuditExportStatus } from "@/lib/graphql/ai-audit";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { useMemo } from "react";
import { HeldByChips } from "../safety/safety-badges";
import { auditExportStatusAttrs, auditOutcomeAttrs, auditTierLabel } from "./audit-model";

export function AuditOutcomeBadge({ outcome }: { outcome: AiAuditEventOutcome }) {
  const t = useT();
  const attrs = useMemo(() => auditOutcomeAttrs(t), [t]);
  const entry = attrs[outcome];

  return <Badge variant={phaseTone(entry.phase)}>{entry.text}</Badge>;
}

export function AuditExportStatusBadge({ status }: { status: AiAuditExportStatus }) {
  const t = useT();
  const attrs = useMemo(() => auditExportStatusAttrs(t), [t]);
  const entry = attrs[status];

  return <Badge variant={phaseTone(entry.phase)}>{entry.text}</Badge>;
}

/** The tier a call ran at, and what held it below running on its own. */
export function AuditTierCell({
  tier,
  heldBy,
}: {
  tier: string | null;
  heldBy: readonly string[];
}) {
  const t = useT();

  if (!tier) {
    return <span className="text-foreground-subtle">—</span>;
  }

  return (
    <span className="flex min-w-0 flex-col items-start gap-1">
      <span>{auditTierLabel(t, tier)}</span>
      {heldBy.length > 0 ? <HeldByChips heldBy={heldBy} /> : null}
    </span>
  );
}
