import { queries } from "@/lib/queries";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  CAPABILITIES,
  getProfile,
  isCapabilitySectionVisible,
} from "@trenova/shared/lib/capability";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import {
  describeExceedance,
  dimensionMeter,
  dimensionRows,
  escortSummary,
  formatMeasurement,
  hasOpenRequirements,
  isOversize,
  pickupIsTooSoon,
  requirementStateCode,
  restrictionSummary,
  unverifiedJurisdictions,
  type DimensionMeterTone,
  type DimensionRow,
} from "@trenova/shared/lib/permit";
import { cn } from "@trenova/shared/lib/utils";
import type {
  Permit,
  PermitAssessment,
  PermitRequirement,
  PermitStatus,
  RequirementStatus,
} from "@trenova/shared/types/permit";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import {
  CalendarClockIcon,
  CircleDollarSignIcon,
  ClockIcon,
  InfoIcon,
  RulerIcon,
  ShieldQuestionIcon,
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
      className="border-border border-t pt-4"
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
  const openCount = requirements.filter((requirement) => requirement.status === "Open").length;

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
      <DimensionGrid rows={rows} />

      {!assessment.routeResolved && (
        <EmptyNotice>
          No jurisdiction rules matched the stops on this shipment, so no limits were checked. This
          is not the same as a legal load — add stop locations, or confirm the states on this route
          have jurisdiction rules configured.
        </EmptyNotice>
      )}

      {requirements.length > 0 && (
        <div className="rounded-lg border">
          <CardHeader
            title={`Permits required (${requirements.length})`}
            meta={
              openCount > 0 ? (
                <span className="text-2xs text-destructive font-medium tabular-nums">
                  {openCount} open
                </span>
              ) : (
                <span className="text-2xs text-muted-foreground">All resolved</span>
              )
            }
          />
          <ScrollArea className="rounded-b-lg" viewportClassName="max-h-64" maskHeight={16}>
            <ul className="divide-y">
              {requirements.map((requirement) => (
                <RequirementRow
                  // Derived requirements have no ID, so the jurisdiction and its
                  // place on the route are what actually identify a row.
                  key={`${requirement.stateId}-${requirement.routeSequence}`}
                  requirement={requirement}
                  onRecord={() => setRecording(requirement)}
                  onWaive={() => setWaiving(requirement)}
                />
              ))}
            </ul>
          </ScrollArea>
        </div>
      )}

      {!!permits?.length && (
        <div className="rounded-lg border">
          <CardHeader title={`Permits on file (${permits.length})`} />
          <ScrollArea className="rounded-b-lg" viewportClassName="max-h-64" maskHeight={16}>
            <ul className="divide-y">
              {permits.map((entry) => (
                <PermitRow key={entry.id} permit={entry} onEdit={() => setEditing(entry)} />
              ))}
            </ul>
          </ScrollArea>
        </div>
      )}

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {escorts.length > 0 && (
          <SummaryCard
            icon={<TruckIcon className="size-3.5" />}
            label="Escort vehicles"
            value={`${assessment.totalEscorts} for the trip`}
            hint="Counted once per role across the route, not once per state."
          >
            <ul className="space-y-0.5">
              {escorts.map((escort) => (
                <li key={escort.role} className="text-muted-foreground text-xs">
                  <span className="text-foreground">{escort.label}</span>
                  {escort.stateCodes.length > 0 && ` — required by ${escort.stateCodes.join(", ")}`}
                </li>
              ))}
            </ul>
          </SummaryCard>
        )}

        {restrictions.length > 0 && (
          <SummaryCard
            icon={<ClockIcon className="size-3.5" />}
            label="Movement restrictions"
            hint="Restrictions published by the permitting jurisdictions on this route. Trenova does not yet evaluate them against your appointment times."
          >
            <ul className="space-y-0.5">
              {restrictions.map((restriction) => (
                <li key={restriction.kind} className="text-muted-foreground text-xs">
                  <span className="text-foreground">{restriction.label}</span> —{" "}
                  {restriction.stateCodes.join(", ")}
                </li>
              ))}
            </ul>
          </SummaryCard>
        )}

        {assessment.maxLeadTimeDays > 0 && (
          <SummaryCard
            icon={<CalendarClockIcon className="size-3.5" />}
            label="Earliest feasible pickup"
            value={formatToUserTimezone(assessment.earliestPickup, {
              showTimeZone: false,
              showSeconds: false,
            })}
            hint="Derived from the slowest jurisdiction's permit lead time."
            tone={pickupTooSoon ? "warning" : undefined}
          >
            <p className="text-muted-foreground text-xs">
              {assessment.maxLeadTimeDays} day
              {assessment.maxLeadTimeDays === 1 ? "" : "s"} of permit lead time on this route.
            </p>
            {pickupTooSoon && (
              <p className="mt-1 text-xs font-medium text-yellow-700 dark:text-yellow-400">
                The booked pickup falls inside that window and cannot be permitted in time.
              </p>
            )}
          </SummaryCard>
        )}

        {(assessment.totalEstimatedFee ?? 0) > 0 && (
          <SummaryCard
            icon={<CircleDollarSignIcon className="size-3.5" />}
            label="Estimated permit fees"
            value={`$${(assessment.totalEstimatedFee ?? 0).toLocaleString()}`}
          >
            {assessment.feeIsBaseOnly && (
              <p className="text-muted-foreground text-xs">
                Base fees only — per-mile charges are excluded because per-state mileage is not
                available for this route, so the real cost will be higher where a jurisdiction
                charges by distance.
              </p>
            )}
          </SummaryCard>
        )}
      </div>

      {unverified.length > 0 && (
        <div className="flex items-start gap-2.5 rounded-lg border border-yellow-600/30 bg-yellow-600/10 px-3 py-2.5">
          <ShieldQuestionIcon className="mt-0.5 size-3.5 shrink-0 text-yellow-700 dark:text-yellow-400" />
          <div className="space-y-0.5">
            <p className="text-xs font-medium text-yellow-700 dark:text-yellow-400">
              Unconfirmed limits for {unverified.map((j) => j.stateCode).join(", ")}
            </p>
            <p className="text-muted-foreground text-xs">
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

const METER_FILL: Record<DimensionMeterTone, string> = {
  ok: "bg-success",
  near: "bg-warning",
  over: "bg-destructive",
};

function DimensionGrid({ rows }: { rows: DimensionRow[] }) {
  const overCount = rows.filter((row) => row.exceeded).length;
  const hasLimits = rows.some((row) => row.limit !== null);

  return (
    <div className="rounded-lg border p-3">
      <div className="mb-2.5 flex items-center justify-between gap-2">
        <span className="text-2xs text-muted-foreground font-medium tracking-wider uppercase">
          Dimensions vs tightest limit
        </span>
        {hasLimits && (
          <span
            className={cn(
              "text-2xs font-medium tabular-nums",
              overCount > 0 ? "text-destructive" : "text-muted-foreground",
            )}
          >
            {overCount > 0 ? `${overCount} of ${rows.length} over` : "All within limits"}
          </span>
        )}
      </div>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        {rows.map((row) => (
          <DimensionTile key={row.key} row={row} />
        ))}
      </div>
    </div>
  );
}

function DimensionTile({ row }: { row: DimensionRow }) {
  const meter = dimensionMeter(row);

  return (
    <div
      className={cn(
        "rounded-md border px-2.5 py-2",
        row.exceeded && "border-destructive/40 bg-destructive/5",
      )}
    >
      <div className="mb-1.5 flex items-center justify-between gap-2">
        <span className="text-2xs text-muted-foreground font-medium">{row.label}</span>
        {row.headroom !== null &&
          (row.exceeded ? (
            <span className="bg-destructive/15 text-2xs text-destructive rounded-full px-1.5 py-px font-semibold tabular-nums">
              {formatMeasurement(Math.abs(row.headroom), row.unit)} over
            </span>
          ) : (
            <span className="text-2xs text-muted-foreground tabular-nums">
              {formatMeasurement(row.headroom, row.unit)} left
            </span>
          ))}
      </div>
      {meter && (
        <div className="bg-muted mb-1.5 h-1.5 w-full overflow-hidden rounded-full">
          <div
            className={cn("h-full rounded-full transition-all", METER_FILL[meter.tone])}
            style={{ width: `${meter.percentOfLimit}%` }}
          />
        </div>
      )}
      <div className="flex items-baseline justify-between gap-2">
        <span
          className={cn("text-xs font-semibold tabular-nums", row.exceeded && "text-destructive")}
        >
          {formatMeasurement(row.actual, row.unit)}
        </span>
        <span className="text-2xs text-muted-foreground tabular-nums">
          {row.limit === null ? (
            "no limit matched"
          ) : (
            <>
              / {formatMeasurement(row.limit, row.unit)}
              {row.governingStateCode && (
                <span className="ml-1 uppercase">{row.governingStateCode}</span>
              )}
            </>
          )}
        </span>
      </div>
    </div>
  );
}

const REQUIREMENT_STATUS_VARIANT: Record<RequirementStatus, BadgeVariant> = {
  Open: "inactive",
  Satisfied: "active",
  Waived: "outline",
  Superseded: "outline",
};

function RequirementRow({
  requirement,
  onRecord,
  onWaive,
}: {
  requirement: PermitRequirement;
  onRecord: () => void;
  onWaive: () => void;
}) {
  const code = requirementStateCode(requirement);
  const stateName = requirement.provenance?.stateName;
  const exceedances = requirement.exceedances ?? [];
  // Actions need a persisted row to act on. A derived requirement has no ID yet
  // because the shipment has not been saved since the engine ran, and offering
  // a button that cannot resolve a target would be worse than offering none.
  const actionable = requirement.status === "Open" && !!requirement.id;

  return (
    <li className="flex flex-wrap items-start justify-between gap-x-3 gap-y-2 px-3 py-2.5">
      <div className="flex min-w-0 flex-1 items-start gap-2.5">
        <StateChip code={code} />
        <div className="min-w-0 space-y-1">
          <p className="text-xs font-medium">
            {stateName ? `${stateName} permit` : code ? `${code} permit` : "Permit required"}
          </p>
          {exceedances.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {requirement.isSuperload && (
                <span className="text-2xs rounded-sm bg-orange-600/15 px-1.5 py-px font-medium text-orange-700 dark:text-orange-400">
                  superload
                </span>
              )}
              {exceedances.map((exceedance) => (
                <span
                  key={exceedance.trigger}
                  className="bg-muted text-2xs text-muted-foreground rounded-sm px-1.5 py-px tabular-nums"
                >
                  {describeExceedance(exceedance)}
                </span>
              ))}
            </div>
          )}
          <p className="text-2xs text-muted-foreground">
            {requirement.leadTimeDays} day
            {requirement.leadTimeDays === 1 ? "" : "s"} lead time
            {requirement.validityDays > 0 && ` · valid ${requirement.validityDays} days`}
          </p>
          {requirement.status === "Waived" && requirement.waiverReason && (
            <p className="text-2xs text-muted-foreground">Waived: {requirement.waiverReason}</p>
          )}
        </div>
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
        <Badge variant={REQUIREMENT_STATUS_VARIANT[requirement.status]}>{requirement.status}</Badge>
      </div>
    </li>
  );
}

const PERMIT_STATUS_VARIANT: Record<PermitStatus, BadgeVariant> = {
  Active: "active",
  Pending: "warning",
  Expired: "inactive",
  Void: "outline",
};

function PermitRow({ permit, onEdit }: { permit: Permit; onEdit: () => void }) {
  return (
    <li className="flex flex-wrap items-start justify-between gap-x-3 gap-y-2 px-3 py-2.5">
      <div className="flex min-w-0 flex-1 items-start gap-2.5">
        <StateChip code={permit.state?.abbreviation ?? ""} />
        <div className="min-w-0 space-y-0.5">
          <p className="text-xs font-medium tabular-nums">{permit.permitNumber}</p>
          <p className="text-2xs text-muted-foreground">
            {permit.expiresAt
              ? `Expires ${formatToUserTimezone(permit.expiresAt, {
                  showTimeZone: false,
                  showTime: false,
                })}`
              : "No expiry recorded"}
          </p>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        {/* A permit number keyed wrong, or an expiry that turns out to fall
            short of the last stop, is corrected here rather than by recording
            a second permit beside the first. */}
        <Button type="button" variant="outline" size="xxs" onClick={onEdit}>
          Edit
        </Button>
        <Badge variant={PERMIT_STATUS_VARIANT[permit.status]}>{permit.status}</Badge>
      </div>
    </li>
  );
}

function StateChip({ code }: { code: string }) {
  return (
    <span className="bg-muted/60 text-2xs flex size-7 shrink-0 items-center justify-center rounded-md border font-semibold uppercase">
      {code || "—"}
    </span>
  );
}

function CardHeader({ title, meta }: { title: string; meta?: React.ReactNode }) {
  return (
    <div className="border-border flex items-center justify-between gap-2 border-b px-3 py-2">
      <span className="text-2xs text-muted-foreground font-medium tracking-wider uppercase">
        {title}
      </span>
      {meta}
    </div>
  );
}

function SummaryCard({
  icon,
  label,
  value,
  hint,
  tone,
  children,
}: {
  icon: React.ReactNode;
  label: string;
  value?: string;
  hint?: string;
  tone?: "warning";
  children?: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "bg-muted/40 rounded-lg border px-3 py-2.5",
        tone === "warning" && "border-yellow-600/30 bg-yellow-600/10",
      )}
    >
      <div className="flex items-center gap-1.5">
        <span
          className={cn(
            "text-muted-foreground",
            tone === "warning" && "text-yellow-700 dark:text-yellow-400",
          )}
        >
          {icon}
        </span>
        <span className="text-2xs text-muted-foreground font-medium tracking-wider uppercase">
          {label}
        </span>
        {hint && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="text-muted-foreground cursor-help">
                  <InfoIcon className="size-3" />
                </span>
              }
            />
            <TooltipContent side="top" sideOffset={8} className="max-w-xs">
              {hint}
            </TooltipContent>
          </Tooltip>
        )}
      </div>
      {value && <p className="mt-1.5 text-sm font-medium tabular-nums">{value}</p>}
      {children && <div className={value ? "mt-1" : "mt-1.5"}>{children}</div>}
    </div>
  );
}

function EmptyNotice({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2.5 rounded-lg border border-dashed px-3 py-2.5">
      <RulerIcon className="text-muted-foreground mt-0.5 size-3.5 shrink-0" />
      <p className="text-muted-foreground text-xs">{children}</p>
    </div>
  );
}
