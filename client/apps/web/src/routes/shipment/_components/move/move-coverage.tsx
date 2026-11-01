import { CarrierTrackingLink } from "@/components/carrier-assignment/carrier-tracking-link";
import { RateConfirmationActions } from "@/components/carrier-assignment/rate-confirmation-actions";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { ChevronRightIcon, User01Icon, UserPlus01Icon } from "@trenova/shared/components/icons";
import {
  Avatar,
  AvatarBadge,
  AvatarFallback,
  AvatarImage,
} from "@trenova/shared/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatCurrency, getNameInitials } from "@trenova/shared/lib/utils";
import type { Assignment, CarrierAssignment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";

type WorkerSummary = NonNullable<Assignment["primaryWorker"]>;

function workerText(worker: WorkerSummary | null | undefined): string {
  if (!worker) return "";
  return (
    worker.wholeName?.trim() ||
    [worker.firstName, worker.lastName]
      .map((part) => part?.trim())
      .filter(Boolean)
      .join(" ")
  );
}

function money(value: string | number | null | undefined, currency?: string | null): string {
  const amount = typeof value === "string" ? Number(value) : (value ?? 0);
  return formatCurrency(Number.isFinite(amount) ? amount : 0, currency || "USD");
}

const TENDER_DOT: Record<CarrierAssignment["status"], string> = {
  Pending: "bg-warning",
  Confirmed: "bg-success",
  Canceled: "bg-danger",
};

/** What a carrier tender's status means, in the words of the person reading it. */
function tenderStatusText(status: CarrierAssignment["status"], t: TranslateFn): string {
  switch (status) {
    case "Pending":
      return t("Waiting for the carrier to confirm the rate");
    case "Confirmed":
      return t("The carrier confirmed the rate for this move");
    case "Canceled":
      return t("The carrier assignment was canceled");
  }
}

/**
 * Who covers the move, as a small tile: their photo, or their initials, with a dot on its corner
 * for where a carrier's tender stands, explained on hover.
 */
function CoverageAvatar({
  name,
  imageUrl,
  tender,
}: {
  name: string;
  imageUrl?: string | null;
  tender?: CarrierAssignment["status"];
}) {
  const t = useT();
  return (
    <Avatar size="sm" className="rounded-full after:rounded-full">
      {imageUrl ? <AvatarImage src={imageUrl} alt="" className="rounded-full" /> : null}
      <AvatarFallback className="text-2xs rounded-full font-medium">
        {getNameInitials(name, "?")}
      </AvatarFallback>
      {tender ? (
        <Tooltip>
          <TooltipTrigger
            render={
              <AvatarBadge data-tender={tender} className={cn("ring-card", TENDER_DOT[tender])} />
            }
          />
          <TooltipContent side="top">{tenderStatusText(tender, t)}</TooltipContent>
        </Tooltip>
      ) : null}
    </Avatar>
  );
}

/** One fact of the coverage: its label on the left, its value on the right; nothing when empty. */
function Fact({
  label,
  children,
  className,
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  if (!children) return null;
  return (
    <div className="flex items-baseline justify-between gap-4 text-xs">
      <dt className="text-foreground-muted shrink-0">{label}</dt>
      <dd className={cn("min-w-0 truncate text-right", className)}>{children}</dd>
    </div>
  );
}

/**
 * The carrier covering a move, in one line (who, where the tender stands, what it costs) that
 * opens to the rest: the carrier's pro number, driver, tractor and trailer, the cost's parts, the
 * rate confirmation and the tracking link.
 */
function CarrierCoverage({
  carrierAssignment,
  moveId,
  canTrack,
}: {
  carrierAssignment: CarrierAssignment;
  moveId?: string;
  canTrack: boolean;
}) {
  const t = useT();
  const currency = carrierAssignment.currencyCode;
  const name = carrierAssignment.carrier?.name || t("External carrier");
  const equipment = [
    carrierAssignment.externalTractorNumber
      ? t("Tractor {0}", carrierAssignment.externalTractorNumber)
      : null,
    carrierAssignment.externalTrailerNumber
      ? t("Trailer {0}", carrierAssignment.externalTrailerNumber)
      : null,
  ].filter(Boolean);
  const facts = {
    pro: carrierAssignment.proNumber,
    driver: carrierAssignment.externalDriverName,
    equipment: equipment.join(" · "),
  };

  return (
    <Collapsible className="animate-rise border-t [animation-delay:120ms]">
      <div className="flex items-center gap-1 py-2 pr-3 pl-4">
        <CollapsibleTrigger className="group ui-focus-ring flex min-w-0 flex-1 items-center gap-2.5 rounded-md py-0.5 text-left">
          <CoverageAvatar name={name} tender={carrierAssignment.status} />
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="truncate text-xs font-medium">
              {name}
              <span className="sr-only">
                {" · "}
                {tenderStatusText(carrierAssignment.status, t)}
              </span>
            </span>
            <span className="text-foreground-muted text-2xs truncate">
              {[
                carrierAssignment.carrier?.scac,
                facts.driver,
                facts.pro ? t("Pro {0}", facts.pro) : null,
              ]
                .filter(Boolean)
                .join(" · ") || t("Carrier")}
            </span>
          </span>
          <span className="shrink-0 text-xs font-medium tabular-nums">
            {money(carrierAssignment.totalCost, currency)}
          </span>
          <ChevronRightIcon
            aria-hidden
            className="text-foreground-subtle size-3.5 shrink-0 transition-transform group-data-[panel-open]:rotate-90"
          />
        </CollapsibleTrigger>
      </div>
      <CollapsibleContent className="ease-settle h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 data-ending-style:h-0 data-starting-style:h-0">
        <div className="animate-expand-in flex flex-col gap-3 px-4 pt-1 pb-4">
          {(facts.pro || facts.driver || facts.equipment) && (
            <dl className="flex flex-col gap-1.5">
              <Fact label={t("Pro number")} className="font-mono">
                {facts.pro}
              </Fact>
              <Fact label={t("Driver")}>{facts.driver}</Fact>
              <Fact label={t("Equipment")} className="tabular-nums">
                {facts.equipment}
              </Fact>
            </dl>
          )}
          <dl
            aria-label={t("Carrier cost")}
            className="flex flex-col gap-1.5 border-t pt-3 tabular-nums"
          >
            <Fact label={t("Base rate")}>{money(carrierAssignment.baseAmount, currency)}</Fact>
            <Fact label={t("Fuel surcharge")}>
              {money(carrierAssignment.fuelSurcharge, currency)}
            </Fact>
            <Fact label={t("Accessorials")}>
              {money(carrierAssignment.accessorialTotal, currency)}
            </Fact>
          </dl>
          {moveId && (carrierAssignment.id || canTrack) && (
            <div className="flex flex-col divide-y rounded-md border">
              {carrierAssignment.id && (
                <div className="px-2.5 py-2">
                  <RateConfirmationActions
                    moveId={moveId}
                    carrierAssignmentId={carrierAssignment.id}
                  />
                </div>
              )}
              {canTrack && (
                <div className="px-2.5 py-2">
                  <CarrierTrackingLink
                    moveId={moveId}
                    carrierName={carrierAssignment.carrier?.name}
                  />
                </div>
              )}
            </div>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

/**
 * The company driver covering a move, in one line: who drives, with whom, and the tractor and
 * trailer. The move carries all of it from the shipment read; an assignment just made in this
 * session may arrive without its people and equipment, so those are read for it alone.
 */
function DriverCoverage({ assignment }: { assignment: Assignment }) {
  const t = useT();
  const embedded = !!(
    assignment.primaryWorker ||
    assignment.tractor ||
    assignment.trailer ||
    assignment.secondaryWorker
  );
  const { data: fetched } = useQuery({
    queryKey: ["assignment", assignment.id],
    queryFn: ({ signal }) => apiService.assignmentService.get(assignment.id!, { signal }),
    enabled: !!assignment.id && !embedded,
    staleTime: 5 * 60 * 1000,
  });
  const summary = embedded ? assignment : (fetched ?? assignment);
  const primary = workerText(summary.primaryWorker);
  const secondary = workerText(summary.secondaryWorker);
  const equipment = [
    summary.tractor?.code ? t("Tractor {0}", summary.tractor.code) : null,
    summary.trailer?.code ? t("Trailer {0}", summary.trailer.code) : null,
  ].filter(Boolean);

  const driver = primary
    ? secondary
      ? t("{0} and {1}", primary, secondary)
      : primary
    : t("Driver assigned");

  return (
    <div className="animate-rise flex items-center gap-2.5 border-t px-4 py-2.5 [animation-delay:120ms]">
      <CoverageAvatar name={primary || driver} imageUrl={summary.primaryWorker?.profilePicUrl} />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-xs font-medium">{driver}</span>
        <span className="text-foreground-muted text-2xs truncate tabular-nums">
          {equipment.length > 0 ? equipment.join(" · ") : t("Company driver")}
        </span>
      </span>
    </div>
  );
}

/**
 * Who covers the move: the carrier, the company driver, or, before anyone does, a prompt to
 * assign one (or why it cannot be assigned yet).
 */
export function MoveCoverage({
  carrierAssignment,
  assignment,
  moveId,
  canTrack,
  assignBlockedReason,
  onAssign,
}: {
  /** The active carrier tender, if a carrier covers the move. */
  carrierAssignment: CarrierAssignment | null;
  assignment: Assignment | null | undefined;
  moveId?: string;
  canTrack: boolean;
  /** Why the move cannot be assigned now; null when it can. */
  assignBlockedReason: string | null;
  onAssign: () => void;
}) {
  const t = useT();

  if (carrierAssignment) {
    return (
      <CarrierCoverage carrierAssignment={carrierAssignment} moveId={moveId} canTrack={canTrack} />
    );
  }
  if (assignment?.id) {
    return <DriverCoverage assignment={assignment} />;
  }

  return (
    <div className="animate-rise flex items-center gap-2.5 border-t px-4 py-2.5 [animation-delay:120ms]">
      <span
        aria-hidden
        className="border-border-strong text-foreground-subtle grid size-6 shrink-0 place-items-center rounded-full border border-dashed"
      >
        <User01Icon className="size-3" />
      </span>
      <span className="text-foreground-muted min-w-0 flex-1 truncate text-xs">
        {assignBlockedReason ?? t("No driver or carrier yet")}
      </span>
      {!assignBlockedReason && (
        <Button type="button" variant="outline" size="xxs" onClick={onAssign}>
          <UserPlus01Icon className="size-3" />
          {t("Assign")}
        </Button>
      )}
    </div>
  );
}
