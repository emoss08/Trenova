import { EntityRedirectLink } from "@/components/ui/link";
import { WorkerSchema } from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import { type ShipmentMove } from "@/types/move";
import { memo } from "react";

interface AssignmentDetailItemProps {
  label: string;
  baseUrl: string;
  entityId?: string;
  value?: string;
  className?: string;
}

const AssignmentDetailItem = memo(function AssignmentDetailItem({
  label,
  value,
  entityId,
  className,
  baseUrl,
}: AssignmentDetailItemProps) {
  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div className="flex items-center gap-1.5">
        <span className="text-xs text-muted-foreground">{label}:</span>
        <EntityRedirectLink
          baseUrl={baseUrl}
          entityId={entityId}
          className="text-xs font-medium max-w-[100px] truncate"
          value={value}
          modelOpen
        >
          {value ?? "-"}
        </EntityRedirectLink>
      </div>
    </div>
  );
});

const getFullName = (worker?: WorkerSchema) => {
  if (!worker) return undefined;
  const { firstName = "", lastName = "" } = worker;
  const fullName = [firstName, lastName].filter(Boolean).join(" ");
  return fullName || undefined;
};

export const AssignmentDetails = memo(function AssignmentDetails({
  move,
}: {
  move?: ShipmentMove;
}) {
  const assignment = move?.assignment;
  if (!assignment) return null;

  const hasAnyAssignment = Boolean(
    assignment.tractor ||
      assignment.trailer ||
      assignment.primaryWorker ||
      assignment.secondaryWorker,
  );

  if (!hasAnyAssignment) return null;

  return (
    <div className="grid grid-cols-2 gap-x-6 gap-y-2 items-center p-3 border-t border-sidebar-border bg-muted/30 rounded-b-md">
      <AssignmentDetailItem
        label="Tractor"
        entityId={assignment.tractor?.id}
        value={assignment.tractor?.code}
        baseUrl="/equipment/configurations/tractors"
      />
      <AssignmentDetailItem
        label="Trailer"
        entityId={assignment.trailer?.id}
        value={assignment.trailer?.code}
        baseUrl="/equipment/configurations/trailers"
      />
      <AssignmentDetailItem
        label="Primary"
        entityId={assignment.primaryWorker?.id}
        value={getFullName(assignment.primaryWorker ?? undefined)}
        baseUrl="/dispatch/configurations/workers"
      />
      <AssignmentDetailItem
        label="Secondary"
        entityId={assignment.secondaryWorker?.id}
        value={getFullName(assignment.secondaryWorker ?? undefined)}
        baseUrl="/dispatch/configurations/workers"
      />
    </div>
  );
});
