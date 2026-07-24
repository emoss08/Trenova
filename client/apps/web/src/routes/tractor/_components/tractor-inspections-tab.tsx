import type { VehicleInspection } from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate, formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn, metersToMiles, pluralize, toTitleCase } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import {
  CableIcon,
  CheckIcon,
  ChevronDownIcon,
  ClipboardCheckIcon,
  MapPinIcon,
  OctagonAlertIcon,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";

const STALE_TIME_MS = 60_000;
const INSPECTION_LIMIT = 50;

type InspectionDefect = {
  id?: string;
  defectType?: string;
  comment?: string;
  resolved?: boolean;
  resolvedAt?: number | null;
};

const SAFETY_STATUS_META: Record<string, { label: string; variant: BadgeVariant }> = {
  safe: { label: "Safe", variant: "active" },
  unsafe: { label: "Unsafe", variant: "inactive" },
  resolved: { label: "Resolved", variant: "warning" },
};

function getSafetyStatusMeta(status: string): { label: string; variant: BadgeVariant } {
  return SAFETY_STATUS_META[status] ?? { label: toTitleCase(status), variant: "secondary" };
}

function parseDefects(defects: unknown): InspectionDefect[] {
  return Array.isArray(defects) ? (defects as InspectionDefect[]) : [];
}

function InspectionsEmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-dashed p-6 text-center">
      {icon}
      <p className="mt-2 text-sm font-medium">{title}</p>
      <p className="mx-auto mt-1 max-w-md text-xs text-muted-foreground">{description}</p>
      {action}
    </div>
  );
}

function InspectionsErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="rounded-lg border border-dashed p-6 text-center">
      <OctagonAlertIcon className="mx-auto size-5 text-destructive" />
      <p className="mt-2 text-sm font-medium">{message}</p>
      <Button type="button" variant="outline" size="sm" className="mt-3" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function InspectionsLoadingState() {
  return (
    <div className="flex flex-col gap-2">
      <Skeleton className="h-16 w-full rounded-lg" />
      <Skeleton className="h-16 w-full rounded-lg" />
      <Skeleton className="h-16 w-full rounded-lg" />
    </div>
  );
}

function DefectRow({ defect }: { defect: InspectionDefect }) {
  return (
    <li className="flex items-start justify-between gap-3 px-4 py-2.5">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium">
          {defect.defectType ? toTitleCase(defect.defectType) : "Defect"}
        </p>
        {defect.comment ? (
          <p className="mt-0.5 text-xs text-muted-foreground">{defect.comment}</p>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {defect.resolved && defect.resolvedAt ? (
          <span className="text-xs text-muted-foreground">{formatUnixDate(defect.resolvedAt)}</span>
        ) : null}
        <Badge variant={defect.resolved ? "active" : "inactive"}>
          {defect.resolved ? "Resolved" : "Open"}
        </Badge>
      </div>
    </li>
  );
}

function InspectionHeader({ inspection }: { inspection: VehicleInspection }) {
  const safetyMeta = getSafetyStatusMeta(inspection.safetyStatus);
  const hasUnresolved = inspection.unresolvedDefectCount > 0;
  const hasDefects = inspection.defectCount > 0;

  const metaParts: { key: string; label: string; isLocation: boolean }[] = [];
  if (inspection.odometerMeters != null) {
    metaParts.push({
      key: "odometer",
      label: `${Math.round(metersToMiles(inspection.odometerMeters)).toLocaleString()} mi`,
      isLocation: false,
    });
  }
  if (inspection.location) {
    metaParts.push({ key: "location", label: inspection.location, isLocation: true });
  }

  return (
    <>
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant="outline">{toTitleCase(inspection.inspectionType)}</Badge>
          <Badge variant={safetyMeta.variant}>{safetyMeta.label}</Badge>
          {inspection.signed ? (
            <span className="inline-flex items-center gap-0.5 text-xs text-green-600 dark:text-green-400">
              <CheckIcon className="size-3" />
              Signed
            </span>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-muted-foreground">
          <span className="tabular-nums">{formatUnixDateTime(inspection.startedAt)}</span>
          {metaParts.map((part) => (
            <span key={part.key} className="flex items-center gap-1">
              <span aria-hidden>·</span>
              {part.isLocation ? <MapPinIcon className="size-3" /> : null}
              <span className="truncate tabular-nums">{part.label}</span>
            </span>
          ))}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {hasDefects ? (
          <span
            className={cn("text-xs", hasUnresolved ? "text-destructive" : "text-muted-foreground")}
          >
            {inspection.defectCount} {pluralize("defect", inspection.defectCount)}
            {hasUnresolved ? ` · ${inspection.unresolvedDefectCount} unresolved` : ""}
          </span>
        ) : (
          <span className="text-xs text-muted-foreground">No defects</span>
        )}
      </div>
    </>
  );
}

function InspectionRow({ inspection }: { inspection: VehicleInspection }) {
  const [open, setOpen] = useState(false);
  const defects = parseDefects(inspection.defects);

  if (defects.length === 0) {
    return (
      <div className="flex items-center gap-3 px-4 py-3">
        <InspectionHeader inspection={inspection} />
      </div>
    );
  }

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50"
        aria-label={`Toggle defects for ${toTitleCase(inspection.inspectionType)} inspection`}
      >
        <InspectionHeader inspection={inspection} />
        <ChevronDownIcon
          className={cn(
            "size-4 shrink-0 text-muted-foreground transition-transform duration-200",
            open && "rotate-180",
          )}
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ul className="divide-y divide-border border-t border-border bg-muted/20">
          {defects.map((defect, index) => (
            <DefectRow key={defect.id ?? `${defect.defectType}-${index}`} defect={defect} />
          ))}
        </ul>
      </CollapsibleContent>
    </Collapsible>
  );
}

export default function TractorInspectionsTab({ tractorId }: { tractorId?: string }) {
  const statusQuery = useQuery({
    ...queries.telematics.status(),
    staleTime: 5 * 60 * 1000,
  });
  const telematicsEnabled = statusQuery.data?.enabled ?? false;
  const hasTractor = typeof tractorId === "string" && tractorId.length > 0;

  const inspectionsQuery = useQuery({
    ...queries.telematics.vehicleInspections(tractorId, undefined, undefined, INSPECTION_LIMIT),
    enabled: telematicsEnabled && hasTractor,
    staleTime: STALE_TIME_MS,
  });

  if (statusQuery.isPending) {
    return <InspectionsLoadingState />;
  }

  if (statusQuery.isError) {
    return (
      <InspectionsErrorState
        message="Telematics status could not be loaded"
        onRetry={() => void statusQuery.refetch()}
      />
    );
  }

  if (!telematicsEnabled) {
    return (
      <InspectionsEmptyState
        icon={<CableIcon className="mx-auto size-6 text-muted-foreground" />}
        title="Telematics not connected"
        description="Connect your Samsara account to stream driver vehicle inspection reports (DVIR) and defect history for this tractor."
        action={
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            render={<Link to="/admin/integrations?type=Samsara" />}
          >
            Open Integrations
          </Button>
        }
      />
    );
  }

  if (inspectionsQuery.isPending) {
    return <InspectionsLoadingState />;
  }

  if (inspectionsQuery.isError) {
    return (
      <InspectionsErrorState
        message="Inspections could not be loaded"
        onRetry={() => void inspectionsQuery.refetch()}
      />
    );
  }

  if (inspectionsQuery.data.length === 0) {
    return (
      <InspectionsEmptyState
        icon={<ClipboardCheckIcon className="mx-auto size-6 text-muted-foreground" />}
        title="No inspections reported for this tractor."
        description="Driver vehicle inspection reports appear here once Samsara reports pre-trip and post-trip inspections."
      />
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div>
        <h3 className="text-sm font-semibold">Inspections</h3>
        <p className="text-xs text-muted-foreground">
          Driver vehicle inspection reports (DVIR) and defect history for this tractor.
        </p>
      </div>
      <div className="overflow-hidden rounded-lg border border-border">
        {inspectionsQuery.data.map((inspection) => (
          <InspectionRow key={inspection.id} inspection={inspection} />
        ))}
      </div>
    </div>
  );
}
