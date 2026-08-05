import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  CAPABILITIES,
  getProfile,
  isCapabilitySectionVisible,
} from "@trenova/shared/lib/capability";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import {
  describeRequirement,
  dimensionRows,
  escortSummary,
  formatMeasurement,
  hasOpenRequirements,
  isOversize,
  pickupIsTooSoon,
  restrictionSummary,
  unverifiedJurisdictions,
  type DimensionRow,
} from "@trenova/shared/lib/permit";
import { cn } from "@trenova/shared/lib/utils";
import type { Permit, PermitAssessment, PermitRequirement } from "@trenova/shared/types/permit";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import {
  CircleAlertIcon,
  ClockIcon,
  RulerIcon,
  ShieldQuestionIcon,
  TriangleAlertIcon,
  TruckIcon,
} from "lucide-react";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { PermitRecordDialog, PermitWaiveDialog } from "./permit-dialogs";

export default function LoadEnvelopePanel() {
  const { control } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" });
  const version = useWatch({ control, name: "version" });
  const moves = useWatch({ control, name: "moves" });

  const { data: shipmentUIPolicy } = useQuery({ ...queries.shipment.uiPolicy() });
  const profile = getProfile(shipmentUIPolicy);
  const visible = isCapabilitySectionVisible(profile, CAPABILITIES.dimensionalCargo);

  const { data: assessment, isPending } = useQuery({
    ...queries.shipment.permitAssessment(shipmentId as string, version),
    enabled: visible && !!shipmentId,
  });

  if (!visible || !shipmentId) return null;

  const scheduledPickupAt = moves?.[0]?.stops?.[0]?.scheduledWindowStart;

  return (
    <FormSection
      title="Load Envelope"
      description="Dimensions, jurisdiction limits, and permits derived from the cargo on this shipment"
      className="border-t border-border pt-4"
      action={assessment ? <EnvelopeStatusBadge assessment={assessment} /> : null}
    >
      {isPending ? (
        <Skeleton className="h-40 w-full rounded-lg" />
      ) : assessment ? (
        <EnvelopeBody
          assessment={assessment}
          scheduledPickupAt={scheduledPickupAt}
          shipmentId={shipmentId as string}
        />
      ) : (
        <EmptyNotice>The permit assessment could not be loaded for this shipment.</EmptyNotice>
      )}
    </FormSection>
  );
}

function EnvelopeStatusBadge({ assessment }: { assessment: PermitAssessment }) {
  if (hasOpenRequirements(assessment)) {
    return <Badge variant="inactive">Permits outstanding</Badge>;
  }
  if (isOversize(assessment)) {
    return <Badge variant="active">Permits in place</Badge>;
  }
  if (!assessment.routeResolved) {
    return <Badge variant="outline">Route not resolved</Badge>;
  }

  return <Badge variant="active">Legal on this route</Badge>;
}

function EnvelopeBody({
  assessment,
  scheduledPickupAt,
  shipmentId,
}: {
  assessment: PermitAssessment;
  scheduledPickupAt: number | null | undefined;
  shipmentId: string;
}) {
  const rows = dimensionRows(assessment);
  // The assessment's requirements are freshly derived and carry no ID, so the
  // per-row actions read the persisted set instead. Falling back to the derived
  // list keeps the summary honest on a shipment saved before the engine ran.
  const { data: persisted } = useQuery({
    ...queries.shipment.permitRequirements(shipmentId),
    enabled: !!shipmentId,
  });
  const { data: permits } = useQuery({
    ...queries.shipment.permits(shipmentId),
    enabled: !!shipmentId,
  });

  const derived = assessment.requirements ?? [];
  const requirements = persisted?.length ? persisted : derived;
  const escorts = escortSummary(requirements);
  const restrictions = restrictionSummary(assessment.jurisdictions);
  const unverified = unverifiedJurisdictions(assessment);
  const pickupTooSoon = pickupIsTooSoon(assessment, scheduledPickupAt);

  const [recording, setRecording] = useState<PermitRequirement | null>(null);
  const [waiving, setWaiving] = useState<PermitRequirement | null>(null);
  const [editing, setEditing] = useState<Permit | null>(null);

  if (assessment.measurements.widthFeet === 0 && assessment.measurements.lengthFeet === 0) {
    return (
      <EmptyNotice>
        Add length, width, and height to the commodity lines. Deck fit, permits, escorts, and lead
        time are all derived from those numbers.
      </EmptyNotice>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="rounded-lg border">
        <div className="grid grid-cols-12 gap-2 border-b border-border px-4 py-2 text-2xs text-muted-foreground uppercase">
          <span className="col-span-3">Dimension</span>
          <span className="col-span-3">Actual</span>
          <span className="col-span-3">Tightest limit</span>
          <span className="col-span-3">Headroom</span>
        </div>
        <div className="divide-y">
          {rows.map((row) => (
            <DimensionRowView key={row.key} row={row} />
          ))}
        </div>
      </div>

      {!assessment.routeResolved && (
        <EmptyNotice>
          No jurisdiction rules matched the stops on this shipment, so no limits were checked. This
          is not the same as a legal load — add stop locations, or confirm the states on this route
          have jurisdiction rules configured.
        </EmptyNotice>
      )}

      {requirements.length > 0 && (
        <div className="rounded-lg border">
          <div className="border-b border-border px-4 py-2 text-2xs text-muted-foreground uppercase">
            Permits required ({requirements.length})
          </div>
          <ul className="divide-y">
            {requirements.map((requirement) => (
              <RequirementRow
                // Derived requirements have no ID, so the jurisdiction and its
                // place on the route are what actually identify a row.
                key={`${requirement.stateId}-${requirement.routeSequence}`}
                requirement={requirement}
                shipmentId={shipmentId}
                onRecord={() => setRecording(requirement)}
                onWaive={() => setWaiving(requirement)}
              />
            ))}
          </ul>
        </div>
      )}

      {!!permits?.length && (
        <div className="rounded-lg border">
          <div className="border-b border-border px-4 py-2 text-2xs text-muted-foreground uppercase">
            Permits on file ({permits.length})
          </div>
          <ul className="divide-y">
            {permits.map((entry) => (
              <li key={entry.id} className="flex items-start justify-between gap-3 px-4 py-2">
                <div className="space-y-0.5">
                  <p className="text-xs font-medium">
                    {entry.state?.abbreviation ?? ""} {entry.permitNumber}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {entry.expiresAt
                      ? `Expires ${formatToUserTimezone(entry.expiresAt, {
                          showTimeZone: false,
                          showTime: false,
                        })}`
                      : "No expiry recorded"}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-1.5">
                  {/* A permit number keyed wrong, or an expiry that turns out to
                      fall short of the last stop, is corrected here rather than
                      by recording a second permit beside the first. */}
                  <Button
                    type="button"
                    variant="outline"
                    size="xxs"
                    onClick={() => setEditing(entry)}
                  >
                    Edit
                  </Button>
                  <Badge variant={entry.status === "Active" ? "active" : "outline"}>
                    {entry.status}
                  </Badge>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {escorts.length > 0 && (
          <SummaryCard
            icon={<TruckIcon className="size-3.5" />}
            title={`${assessment.totalEscorts} escort ${
              assessment.totalEscorts === 1 ? "vehicle" : "vehicles"
            } for the trip`}
            hint="Counted once per role across the route, not once per state."
          >
            <ul className="space-y-1">
              {escorts.map((escort) => (
                <li key={escort.role} className="text-xs text-muted-foreground">
                  {escort.label}
                  {escort.stateCodes.length > 0 && (
                    <span> — required by {escort.stateCodes.join(", ")}</span>
                  )}
                </li>
              ))}
            </ul>
          </SummaryCard>
        )}

        {restrictions.length > 0 && (
          <SummaryCard
            icon={<ClockIcon className="size-3.5" />}
            title="Movement restrictions"
            hint="Restrictions published by the permitting jurisdictions on this route. Trenova does not yet evaluate them against your appointment times."
          >
            <ul className="space-y-1">
              {restrictions.map((restriction) => (
                <li key={restriction.kind} className="text-xs text-muted-foreground">
                  {restriction.label} — {restriction.stateCodes.join(", ")}
                </li>
              ))}
            </ul>
          </SummaryCard>
        )}

        {assessment.maxLeadTimeDays > 0 && (
          <SummaryCard
            icon={<ClockIcon className="size-3.5" />}
            title={`Earliest feasible pickup — ${formatToUserTimezone(assessment.earliestPickup, {
              showTimeZone: false,
              showSeconds: false,
            })}`}
            hint="Derived from the slowest jurisdiction's permit lead time."
            tone={pickupTooSoon ? "warning" : undefined}
          >
            <p className="text-xs text-muted-foreground">
              {assessment.maxLeadTimeDays} day
              {assessment.maxLeadTimeDays === 1 ? "" : "s"} of permit lead time on this route.
            </p>
            {pickupTooSoon && (
              <p className="mt-1 text-xs font-medium text-warning">
                The booked pickup falls inside that window and cannot be permitted in time.
              </p>
            )}
          </SummaryCard>
        )}

        {(assessment.totalEstimatedFee ?? 0) > 0 && (
          <SummaryCard
            icon={<CircleAlertIcon className="size-3.5" />}
            title={`Estimated permit fees — $${(assessment.totalEstimatedFee ?? 0).toLocaleString()}`}
            hint="Base fees only."
          >
            {assessment.feeIsBaseOnly && (
              <p className="text-xs text-muted-foreground">
                Base fees only. Per-mile permit fees are excluded because per-state mileage is not
                available for this route, so the real cost will be higher where a jurisdiction
                charges by distance.
              </p>
            )}
          </SummaryCard>
        )}
      </div>

      {unverified.length > 0 && (
        <div className="flex items-start gap-2 rounded-lg border border-yellow-600/30 bg-yellow-600/10 px-4 py-3">
          <ShieldQuestionIcon className="mt-0.5 size-3.5 shrink-0 text-yellow-700 dark:text-yellow-400" />
          <div className="space-y-1">
            <p className="text-xs font-medium">
              Unconfirmed limits for {unverified.map((j) => j.stateCode).join(", ")}
            </p>
            <p className="text-xs text-muted-foreground">
              These thresholds came from Trenova&apos;s researched baseline and have not been
              confirmed against the issuing authority by your organization. Verify them in
              jurisdiction rules before relying on them for a permit filing.
            </p>
          </div>
        </div>
      )}

      <PermitRecordDialog
        open={recording !== null}
        onOpenChange={(next) => !next && setRecording(null)}
        shipmentId={shipmentId}
        requirement={recording}
      />
      <PermitWaiveDialog
        open={waiving !== null}
        onOpenChange={(next) => !next && setWaiving(null)}
        shipmentId={shipmentId}
        requirement={waiving}
      />
      <PermitRecordDialog
        open={editing !== null}
        onOpenChange={(next) => !next && setEditing(null)}
        shipmentId={shipmentId}
        requirement={null}
        permit={editing}
      />
    </div>
  );
}

function DimensionRowView({ row }: { row: DimensionRow }) {
  return (
    <div
      className={cn(
        "grid grid-cols-12 items-center gap-2 px-4 py-2",
        row.exceeded && "bg-destructive/10",
      )}
    >
      <span className="col-span-3 text-xs font-medium">{row.label}</span>
      <span className="col-span-3 text-xs text-muted-foreground">
        {formatMeasurement(row.actual, row.unit)}
      </span>
      <span className="col-span-3 text-xs text-muted-foreground">
        {row.limit === null ? (
          "—"
        ) : (
          <>
            {formatMeasurement(row.limit, row.unit)}
            {row.governingStateCode && (
              <span className="ml-1 text-2xs uppercase">{row.governingStateCode}</span>
            )}
          </>
        )}
      </span>
      <span
        className={cn(
          "col-span-3 text-xs",
          row.exceeded ? "font-medium text-destructive" : "text-muted-foreground",
        )}
      >
        {row.headroom === null
          ? "—"
          : row.exceeded
            ? `${formatMeasurement(Math.abs(row.headroom), row.unit)} over`
            : `${formatMeasurement(row.headroom, row.unit)} left`}
      </span>
    </div>
  );
}

function RequirementRow({
  requirement,
  onRecord,
  onWaive,
}: {
  requirement: PermitRequirement;
  shipmentId: string;
  onRecord: () => void;
  onWaive: () => void;
}) {
  const satisfied = requirement.status === "Satisfied";
  const waived = requirement.status === "Waived";
  // Actions need a persisted row to act on. A derived requirement has no ID yet
  // because the shipment has not been saved since the engine ran, and offering
  // a button that cannot resolve a target would be worse than offering none.
  const actionable = requirement.status === "Open" && !!requirement.id;

  return (
    <li className="flex items-start justify-between gap-3 px-4 py-2">
      <div className="space-y-0.5">
        <p className="text-xs font-medium">{describeRequirement(requirement)}</p>
        <p className="text-xs text-muted-foreground">
          {requirement.leadTimeDays} day
          {requirement.leadTimeDays === 1 ? "" : "s"} lead time
          {requirement.validityDays > 0 && `, valid ${requirement.validityDays} days`}
          {requirement.isSuperload && " · superload"}
        </p>
        {waived && requirement.waiverReason && (
          <p className="text-xs text-muted-foreground">Waived: {requirement.waiverReason}</p>
        )}
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        {actionable && (
          <>
            <Button type="button" variant="outline" size="xxs" onClick={onRecord}>
              Record permit
            </Button>
            <Button type="button" variant="ghost" size="xxs" onClick={onWaive}>
              Waive
            </Button>
          </>
        )}
        <Badge variant={satisfied ? "active" : waived ? "outline" : "inactive"}>
          {requirement.status}
        </Badge>
      </div>
    </li>
  );
}

function SummaryCard({
  icon,
  title,
  hint,
  tone,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  hint?: string;
  tone?: "warning";
  children?: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "rounded-lg border px-4 py-3",
        tone === "warning" && "border-yellow-600/30 bg-yellow-600/10",
      )}
    >
      <div className="flex items-center gap-1.5">
        <span className="text-muted-foreground">{icon}</span>
        <p className="text-xs font-medium">{title}</p>
        {hint && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="cursor-help text-muted-foreground">
                  <TriangleAlertIcon className="size-3" />
                </span>
              }
            />
            <TooltipContent side="top" sideOffset={8} className="max-w-xs">
              {hint}
            </TooltipContent>
          </Tooltip>
        )}
      </div>
      <div className="mt-1.5">{children}</div>
    </div>
  );
}

function EmptyNotice({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2 rounded-lg border border-dashed px-4 py-3">
      <RulerIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
      <p className="text-xs text-muted-foreground">{children}</p>
    </div>
  );
}
