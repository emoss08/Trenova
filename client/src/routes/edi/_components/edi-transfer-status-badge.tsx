import { Badge } from "@/components/ui/badge";

type EDITransferStatusBadgeProps = {
  status: string;
};

export function EDITransferStatusBadge({ status }: EDITransferStatusBadgeProps) {
  const final = ["Approved", "Rejected", "Expired", "Canceled", "Failed"].includes(status);
  const active = ["Submitted", "MappingRequired", "PendingApproval"].includes(status);

  return <Badge variant={final ? "outline" : active ? "secondary" : "default"}>{status}</Badge>;
}
