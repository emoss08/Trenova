import type { DocumentPacketItem, DocumentPacketSummary } from "@/types/document";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  ChevronDownIcon,
  ClockIcon,
  FileWarningIcon,
  XCircleIcon,
} from "lucide-react";
import { useState } from "react";
import { Badge } from "../ui/badge";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "../ui/collapsible";

interface PacketCompletenessPanelProps {
  summary: DocumentPacketSummary;
}

type BadgeVariant = "teal" | "inactive" | "pink" | "warning" | "purple" | "outline";

function getStatusBadgeVariant(
  status: DocumentPacketItem["status"] | DocumentPacketSummary["status"],
): BadgeVariant {
  switch (status) {
    case "Complete":
      return "teal";
    case "Missing":
    case "Incomplete":
      return "inactive";
    case "Expired":
      return "pink";
    case "ExpiringSoon":
      return "warning";
    case "NeedsReview":
      return "purple";
    default:
      return "outline";
  }
}

function getStatusIcon(status: DocumentPacketItem["status"]) {
  switch (status) {
    case "Complete":
      return <CheckCircle2Icon className="size-4 text-teal-600" />;
    case "Missing":
      return <XCircleIcon className="size-4 text-red-500" />;
    case "Expired":
      return <FileWarningIcon className="size-4 text-pink-500" />;
    case "ExpiringSoon":
      return <ClockIcon className="size-4 text-amber-500" />;
    case "NeedsReview":
      return <AlertTriangleIcon className="size-4 text-purple-500" />;
  }
}

function getStatusLabel(status: DocumentPacketItem["status"]): string {
  switch (status) {
    case "ExpiringSoon":
      return "Expiring Soon";
    case "NeedsReview":
      return "Needs Review";
    default:
      return status;
  }
}

const STATUS_PRIORITY: DocumentPacketItem["status"][] = [
  "Missing",
  "Expired",
  "ExpiringSoon",
  "NeedsReview",
  "Complete",
];

function groupByStatus(items: DocumentPacketItem[]) {
  const groups = new Map<DocumentPacketItem["status"], DocumentPacketItem[]>();
  for (const status of STATUS_PRIORITY) {
    const filtered = items.filter((item) => item.status === status);
    if (filtered.length > 0) {
      groups.set(status, filtered);
    }
  }
  return groups;
}

function PacketItem({ item }: { item: DocumentPacketItem }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border bg-background px-3 py-2">
      <div className="flex items-center gap-2.5 min-w-0">
        {getStatusIcon(item.status)}
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium truncate">{item.documentTypeName}</span>
            <span className="text-xs text-muted-foreground">{item.documentTypeCode}</span>
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {item.required && (
          <Badge variant="outline" className="text-2xs">
            Required
          </Badge>
        )}
        {item.allowMultiple && (
          <Badge variant="outline" className="text-2xs">
            Multiple
          </Badge>
        )}
        {item.documentCount > 0 ? (
          <Badge variant="secondary" className="text-2xs">
            {item.documentCount} doc{item.documentCount !== 1 ? "s" : ""}
          </Badge>
        ) : (
          <Badge variant="outline" className="text-2xs text-muted-foreground">
            No docs
          </Badge>
        )}
        {item.expirationRequired && item.status === "ExpiringSoon" && (
          <Badge variant="warning" className="text-2xs">
            {item.expirationWarningDays}d warning
          </Badge>
        )}
      </div>
    </div>
  );
}

export function PacketCompletenessPanel({ summary }: PacketCompletenessPanelProps) {
  const [isOpen, setIsOpen] = useState(false);
  const grouped = groupByStatus(summary.items);

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className="rounded-lg border bg-card">
        <CollapsibleTrigger className="flex w-full items-center justify-between px-4 py-3 hover:bg-accent/50 transition-colors cursor-pointer">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium">Packet Status</span>
            <Badge variant={getStatusBadgeVariant(summary.status)}>
              {summary.status === "ExpiringSoon"
                ? "Expiring Soon"
                : summary.status === "NeedsReview"
                  ? "Needs Review"
                  : summary.status}
            </Badge>
            <span className="text-sm text-muted-foreground">
              {summary.satisfiedRules}/{summary.totalRules} rules satisfied
            </span>
            {summary.missingRequired > 0 && (
              <Badge variant="inactive" className="text-2xs">
                {summary.missingRequired} missing
              </Badge>
            )}
            {summary.expired > 0 && (
              <Badge variant="pink" className="text-2xs">
                {summary.expired} expired
              </Badge>
            )}
            {summary.expiringSoon > 0 && (
              <Badge variant="warning" className="text-2xs">
                {summary.expiringSoon} expiring
              </Badge>
            )}
            {summary.needsReview > 0 && (
              <Badge variant="purple" className="text-2xs">
                {summary.needsReview} review
              </Badge>
            )}
          </div>
          <ChevronDownIcon
            className={`size-4 text-muted-foreground transition-transform ${isOpen ? "rotate-180" : ""}`}
          />
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className="border-t px-4 py-3 space-y-3">
            {[...grouped.entries()].map(([status, items]) => (
              <div key={status} className="space-y-1.5">
                <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {getStatusIcon(status)}
                  <span>
                    {getStatusLabel(status)} ({items.length})
                  </span>
                </div>
                <div className="space-y-1">
                  {items.map((item) => (
                    <PacketItem key={item.documentTypeId} item={item} />
                  ))}
                </div>
              </div>
            ))}
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
