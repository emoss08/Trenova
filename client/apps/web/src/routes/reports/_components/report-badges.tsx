import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import {
  REPORT_DEFINITION_STATUS_LABELS,
  REPORT_RUN_STATUS_LABELS,
  REPORT_VISIBILITY_LABELS,
} from "@/types/report";

const RUN_STATUS_VARIANTS: Record<string, BadgeVariant> = {
  queued: "info",
  running: "purple",
  succeeded: "active",
  failed: "inactive",
  canceled: "outline",
  expired: "warning",
};

const DEFINITION_STATUS_VARIANTS: Record<string, BadgeVariant> = {
  draft: "outline",
  active: "active",
  archived: "secondary",
  needs_attention: "warning",
};

export function ReportRunStatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={RUN_STATUS_VARIANTS[status] ?? "secondary"}>
      {status === "running" && <Spinner className="size-3" />}
      {REPORT_RUN_STATUS_LABELS[status] ?? status}
    </Badge>
  );
}

export function ReportDefinitionStatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={DEFINITION_STATUS_VARIANTS[status] ?? "secondary"}>
      {REPORT_DEFINITION_STATUS_LABELS[status] ?? status}
    </Badge>
  );
}

export function ReportVisibilityBadge({ visibility }: { visibility: string }) {
  return (
    <Badge variant={visibility === "shared" ? "teal" : "outline"}>
      {REPORT_VISIBILITY_LABELS[visibility] ?? visibility}
    </Badge>
  );
}

export function ReportFormatBadge({ format }: { format: string }) {
  return <Badge variant="secondary">{format.toUpperCase()}</Badge>;
}
