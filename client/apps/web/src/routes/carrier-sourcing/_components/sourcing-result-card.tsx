import { useT } from "@trenova/shared/i18n/use-t";
import {
  CarrierIntelProfileView,
  type CarrierIntelProfileCardId,
} from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { FindingList } from "@/components/carrier-intelligence/finding-list";
import { RiskLevelBadge } from "@/components/carrier-intelligence/risk-level-badge";
import { groupFindings } from "@/lib/carrier-intelligence";
import { carrierPanelPath } from "@/lib/carrier-links";
import { cityStateLabel, oldestActiveAuthorityAgeDays } from "@/lib/carrier-sourcing";
import type { CarrierIntelFinding, CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { formatNumber } from "@trenova/shared/i18n/format";
import { cn } from "@trenova/shared/lib/utils";
import { CheckCircle2Icon, ChevronDownIcon, DownloadIcon, RouteIcon } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router";

export const SOURCING_PROFILE_CARDS: readonly CarrierIntelProfileCardId[] = [
  "authority",
  "insurance",
  "safety",
  "fleet",
  "lanes",
];

export type SourcingCandidate = {
  dotNumber: string;
  legalName: string | null;
  existingCarrierId: string | null;
  riskLevel: string;
  findings: readonly CarrierIntelFinding[];
  profile: CarrierIntelProfile;
  laneMatches?: number;
  score?: number;
  notFound?: boolean;
};

export type SourcingResultCardProps = {
  candidate: SourcingCandidate;
  provider: string | null;
  canImport: boolean;
  ruleLabels?: Readonly<Record<string, string>>;
  onImport: (candidate: SourcingCandidate) => void;
  defaultExpanded?: boolean;
  extraBadges?: ReactNode;
};

function Stat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col">
      <dt className="text-muted-foreground text-2xs uppercase">{label}</dt>
      <dd className="truncate text-sm tabular-nums">{value}</dd>
    </div>
  );
}

export function SourcingResultCard({
  candidate,
  provider,
  canImport,
  ruleLabels,
  onImport,
  defaultExpanded = false,
  extraBadges,
}: SourcingResultCardProps) {
  const t = useT();
  const [expanded, setExpanded] = useState(defaultExpanded);
  const { profile } = candidate;
  const identity = profile.identity;
  const grouped = useMemo(() => groupFindings(candidate.findings), [candidate.findings]);
  const keyFindings = useMemo(
    () => [...grouped.blockers, ...grouped.advisories].slice(0, 3),
    [grouped],
  );
  const authorityAge =
    oldestActiveAuthorityAgeDays(profile.authority) ?? identity?.dotAgeDays ?? null;
  const location = cityStateLabel(
    identity?.physicalAddress?.city,
    identity?.physicalAddress?.state,
  );
  const name = candidate.legalName || identity?.legalName || t("Unnamed carrier");
  const dash = t("Not reported");

  return (
    <Collapsible
      open={expanded}
      onOpenChange={setExpanded}
      render={<li />}
      className="bg-card flex flex-col rounded-lg border"
      data-testid="sourcing-result"
      data-dot-number={candidate.dotNumber}
    >
      <div className="flex flex-col gap-3 p-3">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex flex-wrap items-center gap-1.5">
              <h3 className="truncate text-sm font-semibold">{name}</h3>
              <RiskLevelBadge level={candidate.riskLevel} />
              {candidate.score !== undefined ? (
                <Badge
                  variant="outline"
                  className="max-h-5 tabular-nums"
                  title={t(
                    "Sourcing score: starts at 100, loses points for risk and findings, gains points for lane matches.",
                  )}
                >
                  {t("Score {0}", formatNumber(candidate.score, { maximumFractionDigits: 0 }))}
                </Badge>
              ) : null}
              {candidate.laneMatches !== undefined && candidate.laneMatches > 0 ? (
                <Badge variant="teal" className="max-h-5 tabular-nums">
                  <RouteIcon aria-hidden />
                  {t(
                    "{0, plural, one {# lane load match} other {# lane load matches}}",
                    candidate.laneMatches,
                  )}
                </Badge>
              ) : null}
              {extraBadges}
            </div>
            <p className="text-muted-foreground text-xs">
              {[
                t("USDOT {0}", candidate.dotNumber),
                identity?.docketNumber
                  ? `${identity.docketPrefix || "MC"} ${identity.docketNumber}`
                  : null,
                identity?.dbaName ? t("DBA {0}", identity.dbaName) : null,
                location,
              ]
                .filter(Boolean)
                .join(" · ")}
            </p>
          </div>
          <div className="flex items-center gap-1.5">
            {candidate.existingCarrierId ? (
              <Button
                size="sm"
                variant="outline"
                nativeButton={false}
                render={
                  <Link
                    to={carrierPanelPath(candidate.existingCarrierId, "intelligence")}
                    data-testid="existing-carrier-link"
                  />
                }
              >
                <CheckCircle2Icon className="size-3.5 text-green-600" />
                {t("Already in Trenova")}
              </Button>
            ) : canImport ? (
              <Button type="button" size="sm" onClick={() => onImport(candidate)}>
                <DownloadIcon className="size-3.5" />
                {t("Import")}
              </Button>
            ) : null}
            <CollapsibleTrigger
              render={
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  aria-label={expanded ? t("Hide details") : t("Show details")}
                />
              }
            >
              <ChevronDownIcon
                className={cn("size-4 transition-transform", expanded && "rotate-180")}
              />
            </CollapsibleTrigger>
          </div>
        </div>
        <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Stat
            label={t("Power units")}
            value={
              profile.fleet?.powerUnits != null ? formatNumber(profile.fleet.powerUnits) : dash
            }
          />
          <Stat
            label={t("Drivers")}
            value={profile.fleet?.drivers != null ? formatNumber(profile.fleet.drivers) : dash}
          />
          <Stat
            label={t("Authority age")}
            value={
              authorityAge !== null
                ? t("{0, plural, one {# day} other {# days}}", authorityAge)
                : dash
            }
          />
          <Stat
            label={t("Findings")}
            value={
              grouped.blockers.length + grouped.advisories.length === 0 ? (
                t("None")
              ) : (
                <span className="flex items-center gap-1">
                  {grouped.blockers.length > 0 ? (
                    <Badge variant="inactive" className="max-h-5 tabular-nums">
                      {t(
                        "{0, plural, one {# blocker} other {# blockers}}",
                        grouped.blockers.length,
                      )}
                    </Badge>
                  ) : null}
                  {grouped.advisories.length > 0 ? (
                    <Badge variant="warning" className="max-h-5 tabular-nums">
                      {t(
                        "{0, plural, one {# advisory} other {# advisories}}",
                        grouped.advisories.length,
                      )}
                    </Badge>
                  ) : null}
                </span>
              )
            }
          />
        </dl>
        {keyFindings.length > 0 ? (
          <ul className="flex flex-col gap-1" aria-label={t("Key findings")}>
            {keyFindings.map((finding) => (
              <li
                key={`${finding.action}-${finding.code}`}
                className={cn(
                  "text-xs",
                  finding.action === "Block"
                    ? "text-red-700 dark:text-red-400"
                    : "text-yellow-700 dark:text-yellow-400",
                )}
              >
                {finding.message}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      <CollapsibleContent className="flex flex-col gap-3 border-t p-3">
        {expanded ? (
          <>
            <FindingList
              findings={candidate.findings}
              ruleLabels={ruleLabels}
              emptyMessage={t("No findings. Every enabled rule passed for this carrier.")}
            />
            <CarrierIntelProfileView
              profile={profile}
              provider={provider}
              notFound={candidate.notFound}
              cards={SOURCING_PROFILE_CARDS}
            />
          </>
        ) : null}
      </CollapsibleContent>
    </Collapsible>
  );
}
