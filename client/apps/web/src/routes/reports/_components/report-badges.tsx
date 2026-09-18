import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import {
  REPORT_DEFINITION_STATUS_LABELS,
  REPORT_RUN_STATUS_LABELS,
  REPORT_VISIBILITY_LABELS,
} from "@/types/report";

const RUN_STATUS_VARIANTS: Record<string, BadgeVariant> = {
  queued: "info",
  running: "info",
  succeeded: "success",
  failed: "danger",
  canceled: "neutral",
  expired: "warning",
};

const DEFINITION_STATUS_VARIANTS: Record<string, BadgeVariant> = {
  draft: "neutral",
  active: "success",
  archived: "neutral",
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
    <Badge variant={visibility === "shared" ? "info" : "neutral"}>
      {REPORT_VISIBILITY_LABELS[visibility] ?? visibility}
    </Badge>
  );
}

export function ReportFormatBadge({ format }: { format: string }) {
  return <Badge variant="neutral">{format.toUpperCase()}</Badge>;
}
