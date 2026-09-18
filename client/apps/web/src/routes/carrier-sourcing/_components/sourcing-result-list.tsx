import { StatusDot, riskTone } from "@/components/carrier-intelligence/status-dot";
import { cityStateLabel } from "@/lib/carrier-intelligence";
import { carrierPanelPath } from "@/lib/carrier-links";
import {
  candidateAuthorityAgeDays,
  countFindings,
  type SourcingCandidate,
} from "@/lib/carrier-sourcing";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useRef, useState, type KeyboardEvent } from "react";
import { Link } from "react-router";
import { useSourcingLabels } from "./use-sourcing-labels";

const ROW_GRID =
  "grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-3 md:grid-cols-[minmax(0,1fr)_7rem_4.5rem_6rem_7rem_5.5rem]";

export type SourcingResultRowProps = {
  candidate: SourcingCandidate;
  canImport: boolean;
  tabIndex: number;
  onOpen: (candidate: SourcingCandidate) => void;
  onImport: (candidate: SourcingCandidate) => void;
  onFocusRow?: () => void;
  onKeyDown?: (event: KeyboardEvent<HTMLButtonElement>) => void;
  buttonRef?: (element: HTMLButtonElement | null) => void;
};

export function candidateMetaLine(
  candidate: SourcingCandidate,
  t: ReturnType<typeof useT>,
): string {
  const identity = candidate.profile.identity;
  return [
    t("USDOT {0}", candidate.dotNumber),
    identity?.docketNumber ? `${identity.docketPrefix || "MC"} ${identity.docketNumber}` : null,
    cityStateLabel(identity?.physicalAddress?.city, identity?.physicalAddress?.state),
  ]
    .filter(Boolean)
    .join(" · ");
}

export function SourcingResultRow({
  candidate,
  canImport,
  tabIndex,
  onOpen,
  onImport,
  onFocusRow,
  onKeyDown,
  buttonRef,
}: SourcingResultRowProps) {
  const t = useT();
  const labels = useSourcingLabels();
  const identity = candidate.profile.identity;
  const name = candidate.legalName || identity?.legalName || t("Unnamed carrier");
  const dba = identity?.dbaName && identity.dbaName !== name ? identity.dbaName : null;
  const powerUnits = candidate.profile.fleet?.powerUnits ?? null;
  const age = labels.age(candidateAuthorityAgeDays(candidate));
  const counts = countFindings(candidate.findings);

  return (
    <li
      className={cn(ROW_GRID, "group hover:bg-muted/50 relative h-14 border-b last:border-b-0")}
      data-testid="sourcing-result"
      data-dot-number={candidate.dotNumber}
    >
      <button
        ref={buttonRef}
        type="button"
        tabIndex={tabIndex}
        onClick={() => onOpen(candidate)}
        onFocus={onFocusRow}
        onKeyDown={onKeyDown}
        aria-label={t("Open {0}", name)}
        className="focus-visible:bg-muted/50 focus-visible:ring-ring absolute inset-0 z-0 cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-inset"
      />
      <div className="pointer-events-none relative flex min-w-0 flex-col gap-0.5">
        <div className="flex min-w-0 items-baseline gap-1.5">
          <span className="truncate text-sm font-medium">{name}</span>
          {dba ? <span className="text-muted-foreground truncate text-xs">{dba}</span> : null}
        </div>
        <span className="text-muted-foreground truncate text-xs tabular-nums">
          {candidateMetaLine(candidate, t)}
        </span>
      </div>
      <span className="pointer-events-none relative hidden truncate text-sm tabular-nums md:block">
        {powerUnits !== null ? (
          t("{0} {1, plural, one {truck} other {trucks}}", formatNumber(powerUnits), powerUnits)
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
      </span>
      <span className="pointer-events-none relative hidden text-sm tabular-nums md:block">
        {age ?? <span className="text-muted-foreground">—</span>}
      </span>
      <span className="pointer-events-none relative hidden items-center gap-2 text-sm md:flex">
        <StatusDot tone={riskTone(candidate.riskLevel)} />
        <span className="truncate">{labels.riskLabel(candidate.riskLevel)}</span>
      </span>
      <span className="text-muted-foreground pointer-events-none relative hidden items-center gap-2 text-xs md:flex">
        {counts.blockers > 0 ? (
          <>
            <StatusDot tone="critical" className="size-1.5 [&>span]:size-1.5" />
            <span className="tabular-nums">
              {t("{0, plural, one {# blocker} other {# blockers}}", counts.blockers)}
            </span>
          </>
        ) : counts.advisories > 0 ? (
          <span className="tabular-nums">
            {t("{0, plural, one {# advisory} other {# advisories}}", counts.advisories)}
          </span>
        ) : (
          <span>{t("None")}</span>
        )}
      </span>
      <div className="relative flex justify-end">
        {candidate.existingCarrierId ? (
          <Link
            to={carrierPanelPath(candidate.existingCarrierId, "intelligence")}
            className="text-muted-foreground hover:text-foreground focus-visible:ring-ring rounded-sm text-xs outline-none focus-visible:ring-2"
            data-testid="existing-carrier-link"
          >
            {t("In Trenova")}
          </Link>
        ) : canImport && !candidate.notFound ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onImport(candidate)}
            className="opacity-100 focus-visible:opacity-100 md:opacity-0 md:group-focus-within:opacity-100 md:group-hover:opacity-100"
          >
            {t("Import")}
          </Button>
        ) : null}
      </div>
    </li>
  );
}

export type SourcingResultListProps = {
  candidates: readonly SourcingCandidate[];
  canImport: boolean;
  onOpen: (candidate: SourcingCandidate) => void;
  onImport: (candidate: SourcingCandidate) => void;
};

export function SourcingResultListHeader() {
  const t = useT();
  return (
    <div
      className={cn(ROW_GRID, "text-muted-foreground hidden h-8 border-b text-xs md:grid")}
      aria-hidden
    >
      <span>{t("Carrier")}</span>
      <span>{t("Fleet")}</span>
      <span>{t("Authority")}</span>
      <span>{t("Risk")}</span>
      <span>{t("Findings")}</span>
      <span />
    </div>
  );
}

export function SourcingResultList({
  candidates,
  canImport,
  onOpen,
  onImport,
}: SourcingResultListProps) {
  const t = useT();
  const [activeIndex, setActiveIndex] = useState(0);
  const buttons = useRef<(HTMLButtonElement | null)[]>([]);
  const boundedActive = Math.min(activeIndex, Math.max(candidates.length - 1, 0));

  const focusRow = useCallback((index: number) => {
    buttons.current[index]?.focus();
  }, []);

  const handleKeyDown = (index: number) => (event: KeyboardEvent<HTMLButtonElement>) => {
    const last = candidates.length - 1;
    switch (event.key) {
      case "ArrowDown":
      case "j":
        event.preventDefault();
        focusRow(Math.min(index + 1, last));
        break;
      case "ArrowUp":
      case "k":
        event.preventDefault();
        focusRow(Math.max(index - 1, 0));
        break;
      case "Home":
        event.preventDefault();
        focusRow(0);
        break;
      case "End":
        event.preventDefault();
        focusRow(last);
        break;
      default:
        break;
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border">
      <SourcingResultListHeader />
      <ul aria-label={t("Carriers")}>
        {candidates.map((candidate, index) => (
          <SourcingResultRow
            key={candidate.dotNumber || index}
            candidate={candidate}
            canImport={canImport}
            tabIndex={index === boundedActive ? 0 : -1}
            onOpen={onOpen}
            onImport={onImport}
            onFocusRow={() => setActiveIndex(index)}
            onKeyDown={handleKeyDown(index)}
            buttonRef={(element) => {
              buttons.current[index] = element;
            }}
          />
        ))}
      </ul>
    </div>
  );
}

export function SourcingResultListSkeleton({ rows = 6 }: { rows?: number }) {
  const t = useT();
  return (
    <div className="overflow-hidden rounded-lg border" aria-busy aria-label={t("Loading carriers")}>
      <SourcingResultListHeader />
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className={cn(ROW_GRID, "h-14 border-b last:border-b-0")}>
          <div className="flex flex-col gap-1.5">
            <Skeleton className="h-3.5 w-48" />
            <Skeleton className="h-3 w-64" />
          </div>
          <Skeleton className="hidden h-3.5 w-20 md:block" />
          <Skeleton className="hidden h-3.5 w-12 md:block" />
          <Skeleton className="hidden h-3.5 w-16 md:block" />
          <Skeleton className="hidden h-3.5 w-20 md:block" />
          <span />
        </div>
      ))}
    </div>
  );
}

export function formatCarrierCount(count: number, t: ReturnType<typeof useT>): string {
  return t("{0} {1, plural, one {carrier} other {carriers}}", formatNumber(count), count);
}
