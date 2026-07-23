import { Badge } from "@/components/ui/badge";
import type { NavItemBadgeKind } from "@/config/navigation.types";
import { useAttentionSummary } from "@/hooks/use-attention";

function EDIAttentionNavBadge() {
  const { data } = useAttentionSummary();

  const attentionCount = data?.ediAttention ?? 0;
  if (attentionCount <= 0) return null;

  return (
    <Badge
      variant="inactive"
      className="ml-auto max-h-4 px-1.5 text-2xs tabular-nums"
      title={`${attentionCount} EDI item(s) need attention: dead-lettered messages, quarantined files, or overdue acknowledgments`}
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
