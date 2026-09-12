import { useT } from "@trenova/shared/i18n/use-t";
import type { DocumentPacketItem, DocumentPacketSummary } from "@trenova/shared/types/document";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  ChevronDownIcon,
  ClockIcon,
  FileWarningIcon,
  XCircleIcon,
} from "lucide-react";
import { useState } from "react";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";

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
  const t = useT();

  return (
    <div className="bg-background flex items-center justify-between gap-3 rounded-md border px-3 py-2">
      <div className="flex min-w-0 items-center gap-2.5">
        {getStatusIcon(item.status)}
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-medium">{item.documentTypeName}</span>
            <span className="text-muted-foreground text-xs">{item.documentTypeCode}</span>
          </div>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {item.required && (
          <Badge variant="outline" className="text-2xs">
            {t("Required")}
          </Badge>
        )}
        {item.allowMultiple && (
          <Badge variant="outline" className="text-2xs">
            {t("Multiple")}
          </Badge>
        )}
        {item.documentCount > 0 ? (
          <Badge variant="secondary" className="text-2xs">
            {t("{0, plural, one {# doc} other {# docs}}", item.documentCount)}
          </Badge>
        ) : (
          <Badge variant="outline" className="text-2xs text-muted-foreground">
            {t("No docs")}
          </Badge>
        )}
        {item.expirationRequired && item.status === "ExpiringSoon" && (
          <Badge variant="warning" className="text-2xs">
            {t("{0}d warning", item.expirationWarningDays)}
          </Badge>
        )}
      </div>
    </div>
  );
}

export function PacketCompletenessPanel({ summary }: PacketCompletenessPanelProps) {
  const t = useT();

  const [isOpen, setIsOpen] = useState(false);
  const grouped = groupByStatus(summary.items);

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className="bg-card rounded-lg border">
        <CollapsibleTrigger className="hover:bg-accent/50 flex w-full cursor-pointer items-center justify-between px-4 py-3 transition-colors">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium">{t("Packet Status")}</span>
            <Badge variant={getStatusBadgeVariant(summary.status)}>
              {summary.status === "ExpiringSoon"
                ? t("Expiring Soon")
                : summary.status === "NeedsReview"
                  ? t("Needs Review")
                  : summary.status}
            </Badge>
            <span className="text-muted-foreground text-sm">
              {t("{0}/{1} rules satisfied", summary.satisfiedRules, summary.totalRules)}
            </span>
            {summary.missingRequired > 0 && (
              <Badge variant="inactive" className="text-2xs">
                {t("{0} missing", summary.missingRequired)}
              </Badge>
            )}
            {summary.expired > 0 && (
              <Badge variant="pink" className="text-2xs">
                {t("{0} expired", summary.expired)}
              </Badge>
            )}
            {summary.expiringSoon > 0 && (
              <Badge variant="warning" className="text-2xs">
                {t("{0} expiring", summary.expiringSoon)}
              </Badge>
            )}
            {summary.needsReview > 0 && (
              <Badge variant="purple" className="text-2xs">
                {t("{0} review", summary.needsReview)}
              </Badge>
            )}
          </div>
          <ChevronDownIcon
            className={`text-muted-foreground size-4 transition-transform ${isOpen ? "rotate-180" : ""}`}
          />
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className="space-y-3 border-t px-4 py-3">
            {[...grouped.entries()].map(([status, items]) => (
              <div key={status} className="space-y-1.5">
                <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wider uppercase">
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
