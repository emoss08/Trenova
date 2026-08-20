import type { ShipmentFormSubmission } from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn, pluralize } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ChevronDownIcon, ClipboardListIcon, OctagonAlertIcon } from "lucide-react";
import { useMemo, useState } from "react";

const STALE_TIME_MS = 60_000;

function FormsErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="rounded-lg border border-dashed p-6 text-center">
      <OctagonAlertIcon className="text-destructive mx-auto size-5" />
      <p className="mt-2 text-sm font-medium">Telematics forms could not be loaded</p>
      <Button type="button" variant="outline" size="sm" className="mt-3" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function FormsEmptyState() {
  return (
    <div className="rounded-lg border border-dashed p-6 text-center">
      <ClipboardListIcon className="text-muted-foreground mx-auto size-5" />
      <p className="mt-2 text-sm font-medium">No telematics forms for this shipment yet.</p>
      <p className="text-muted-foreground mx-auto mt-1 max-w-md text-xs">
        Driver form submissions mapped to this shipment appear here once your telematics provider
        reports them.
      </p>
    </div>
  );
}

function FormsLoadingState() {
  return (
    <div className="flex flex-col gap-2">
      <Skeleton className="h-14 w-full rounded-lg" />
      <Skeleton className="h-14 w-full rounded-lg" />
    </div>
  );
}

function SubmissionRow({ submission }: { submission: ShipmentFormSubmission }) {
  const [open, setOpen] = useState(false);
  const hasFields = submission.fields.length > 0;

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        className="hover:bg-muted/50 flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors disabled:cursor-default disabled:hover:bg-transparent"
        disabled={!hasFields}
        aria-label={`Toggle fields for ${submission.templateName}`}
      >
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <Badge variant="outline" className="max-w-full truncate">
              {submission.templateName}
            </Badge>
          </div>
          <p className="text-muted-foreground mt-1 text-xs">
            {submission.workerName ? (
              <span className="text-foreground font-medium">{submission.workerName}</span>
            ) : null}
            {submission.workerName ? " · " : null}
            <span className="tabular-nums">{formatUnixDateTime(submission.submittedAt)}</span>
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {submission.applied ? (
            <Badge variant="active">
              Applied {submission.appliedFields} {pluralize("field", submission.appliedFields)}
            </Badge>
          ) : (
            <Badge variant="secondary">Not applied</Badge>
          )}
          {hasFields ? (
            <ChevronDownIcon
              className={cn(
                "text-muted-foreground size-4 transition-transform duration-200",
                open && "rotate-180",
              )}
            />
          ) : null}
        </div>
      </CollapsibleTrigger>
      {hasFields ? (
        <CollapsibleContent>
          <dl className="border-border bg-muted/20 grid grid-cols-1 gap-x-4 gap-y-2.5 border-t px-4 py-3 sm:grid-cols-2">
            {submission.fields.map((field, index) => (
              <div key={`${field.label}-${index}`} className="min-w-0">
                <dt className="flex items-center gap-1.5">
                  <span className="text-muted-foreground text-[11px]">{field.label}</span>
                  {field.type ? (
                    <span className="bg-muted text-muted-foreground rounded px-1 py-px text-[9.5px] font-medium">
                      {field.type}
                    </span>
                  ) : null}
                </dt>
                <dd className="text-sm break-words">{field.value || "—"}</dd>
              </div>
            ))}
          </dl>
        </CollapsibleContent>
      ) : null}
    </Collapsible>
  );
}

export function TelematicsFormsBlock({ shipmentId }: { shipmentId: string }) {
  const statusQuery = useQuery({
    ...queries.telematics.status(),
    staleTime: 5 * 60 * 1000,
  });
  const telematicsEnabled = statusQuery.data?.enabled ?? false;
  const hasShipment = shipmentId.length > 0;

  const submissionsQuery = useQuery({
    ...queries.telematics.shipmentFormSubmissions(shipmentId),
    enabled: telematicsEnabled && hasShipment,
    staleTime: STALE_TIME_MS,
  });

  const submissions = useMemo(
    () => [...(submissionsQuery.data ?? [])].sort((a, b) => b.submittedAt - a.submittedAt),
    [submissionsQuery.data],
  );

  if (statusQuery.isPending || !telematicsEnabled) {
    return null;
  }

  return (
    <section className="min-w-0">
      <h4 className="cc-label mb-1.5">Telematics forms</h4>
      {submissionsQuery.isPending ? (
        <FormsLoadingState />
      ) : submissionsQuery.isError ? (
        <FormsErrorState onRetry={() => void submissionsQuery.refetch()} />
      ) : submissions.length === 0 ? (
        <FormsEmptyState />
      ) : (
        <div className="border-border overflow-hidden rounded-lg border">
          <div className="divide-border divide-y">
            {submissions.map((submission) => (
              <SubmissionRow key={submission.id} submission={submission} />
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

export default TelematicsFormsBlock;
