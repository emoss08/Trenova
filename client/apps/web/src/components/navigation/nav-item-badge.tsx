import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { NavItemBadgeKind } from "@/config/navigation.types";
import { useAttentionSummary } from "@/hooks/use-attention";

function EDIAttentionNavBadge() {
  const t = useT();
  const { data } = useAttentionSummary();

  const attentionCount = data?.ediAttention ?? 0;
  if (attentionCount <= 0) return null;

  return (
    <Badge
      variant="danger"
      className="text-2xs ml-auto max-h-4 px-1.5 tabular-nums"
      title={t(
        "{0, plural, one {# EDI item needs attention} other {# EDI items need attention}}: dead-lettered messages, quarantined files, or overdue acknowledgments",
        attentionCount,
      )}
    >
      {attentionCount > 99 ? "99+" : attentionCount}
    </Badge>
  );
}

export function NavItemBadge({ badge }: { badge?: NavItemBadgeKind }) {
  if (badge === "edi-attention") {
    return <EDIAttentionNavBadge />;
  }
  return null;
}
