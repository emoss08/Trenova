import { Badge } from "@/components/ui/badge";
import type { NavItemBadgeKind } from "@/config/navigation.types";
import { queries } from "@/lib/queries";
import { usePermissionStore } from "@/stores/permission-store";
import { Operation, Resource } from "@/types/permission";
import { useQuery } from "@tanstack/react-query";

const EDI_ATTENTION_REFETCH_INTERVAL = 60_000;

type StatusCount = { status: string; count: number };

function countFor(counts: StatusCount[], status: string) {
  return counts.find((entry) => entry.status === status)?.count ?? 0;
}

function EDIAttentionNavBadge() {
  const canRead = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Read),
  );
  const { data } = useQuery({
    ...queries.edi.summary(),
    refetchInterval: EDI_ATTENTION_REFETCH_INTERVAL,
    enabled: canRead,
  });

  const summary = data?.ediSummary;
  if (!summary) return null;

  const attentionCount =
    countFor(summary.deliveryStatusCounts, "DeadLettered") +
    countFor(summary.inboundFileStatusCounts, "Quarantined") +
    summary.overdueAckCount;
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
