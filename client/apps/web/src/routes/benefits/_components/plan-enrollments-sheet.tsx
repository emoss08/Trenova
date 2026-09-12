import { useT } from "@trenova/shared/i18n/use-t";
import type { BenefitEnrollmentListRow, BenefitPlanRow } from "@/lib/graphql/benefits";
import {
  ENROLLMENT_STANDING_LABELS,
  enrollmentStanding,
  enrollmentStandingTone,
  matchesEnrollmentSearch,
} from "@/lib/benefits-console";
import { useQuery } from "@tanstack/react-query";
import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { coverageTierLabel, formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { initials } from "@trenova/shared/lib/utils";
import { SearchIcon } from "lucide-react";
import { Link } from "react-router";
import { useMemo, useState } from "react";
import { planEnrollmentsQuery } from "./queries";

function workerHref(workerId: string): string {
  return `/hr/workers?entityId=${workerId}&modType=edit`;
}

export function enrollmentWorkerName(entry: BenefitEnrollmentListRow): string {
  return entry.worker ? `${entry.worker.firstName} ${entry.worker.lastName}` : entry.workerId;
}

type PlanEnrollmentsSheetProps = {
  plan: BenefitPlanRow | null;
  now: number;
  onOpenChange: (open: boolean) => void;
};

/**
 * Everyone who has ever been put on a plan, and where each of them stands
 * today. Declines and ended cover are kept on the list rather than filtered
 * out: "who declined this" is asked at audit as often as "who is on it".
 */
export function PlanEnrollmentsSheet({ plan, now, onOpenChange }: PlanEnrollmentsSheetProps) {
  const t = useT();

  const [query, setQuery] = useState("");
  const enrollments = useQuery({
    ...planEnrollmentsQuery(plan?.id ?? ""),
    enabled: Boolean(plan),
  });
  const rows = useMemo(() => {
    const entries = (enrollments.data ?? []).map((entry) => ({
      entry,
      standing: enrollmentStanding(entry, now),
    }));
    return entries
      .filter(({ entry }) => matchesEnrollmentSearch(entry, query))
      .sort((a, b) => standingRank(a.standing) - standingRank(b.standing));
  }, [enrollments.data, now, query]);
  const covered = rows.filter(
    ({ standing }) => standing === "covered" || standing === "ending",
  ).length;

  return (
    <Sheet open={Boolean(plan)} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{plan?.name ?? t("Plan")}</SheetTitle>
          <SheetDescription>
            {plan
              ? t(
                  "{0} · {1} · {2} employee, {3} employer per period",
                  plan.code,
                  plan.planYear,
                  formatMinor(plan.employeeCostMinor, plan.currencyCode),
                  formatMinor(plan.employerCostMinor, plan.currencyCode),
                )
              : t("Loading")}
          </SheetDescription>
        </SheetHeader>

        <div className="flex flex-col gap-3 overflow-y-auto px-4 pb-4">
          <div className="flex items-center justify-between gap-2">
            <Input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("Find a name or terminal")}
              aria-label={t("Find on this plan")}
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
              inputContainerClassName="w-60 max-w-full"
            />
            {enrollments.data ? (
              <span className="text-muted-foreground text-xs tabular-nums">
                {t("{0} covered · {1} on record", covered, rows.length)}
              </span>
            ) : null}
          </div>

          {enrollments.isLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-2/3" />
            </div>
          ) : rows.length === 0 ? (
            <p className="text-muted-foreground rounded-lg border border-dashed p-4 text-sm">
              {query
                ? t("Nobody on this plan matches that.")
                : t("Nobody has been put on this plan yet.")}
            </p>
          ) : (
            <ul
              className="bg-card divide-y overflow-hidden rounded-lg border"
              aria-label={t("People on the plan")}
            >
              {rows.map(({ entry, standing }) => {
                const name = enrollmentWorkerName(entry);
                const fleet = entry.worker?.fleetCode ?? null;
                return (
                  <li key={entry.id}>
                    <Link
                      to={workerHref(entry.workerId)}
                      className="hover:bg-accent grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 transition-colors"
                    >
                      <span className="flex min-w-0 items-center gap-2.5">
                        <Avatar size="sm">
                          {entry.worker?.profilePicUrl ? (
                            <AvatarImage src={entry.worker.profilePicUrl} alt="" />
                          ) : null}
                          <AvatarFallback className="text-2xs font-medium">
                            {entry.worker
                              ? initials(entry.worker.firstName, entry.worker.lastName)
                              : "?"}
                          </AvatarFallback>
                        </Avatar>
                        <span className="flex min-w-0 flex-col leading-tight">
                          <span className="flex min-w-0 items-center gap-1.5">
                            <span className="truncate text-sm font-medium">{name}</span>
                            {fleet ? (
                              <span className="text-muted-foreground text-2xs flex items-center gap-1">
                                <span
                                  aria-hidden
                                  className="size-1.5 rounded-full"
                                  style={{ backgroundColor: fleet.color }}
                                />
                                {fleet.code}
                              </span>
                            ) : null}
                          </span>
                          <span className="text-muted-foreground text-xs">
                            {describeEntry(entry, standing)}
                          </span>
                        </span>
                      </span>
                      <span className="flex items-center gap-2">
                        {standing !== "declined" && standing !== "ended" ? (
                          <span className="font-mono text-xs tabular-nums">
                            {formatMinor(entry.employeeCostMinor, entry.benefitPlan?.currencyCode)}
                          </span>
                        ) : null}
                        <Badge variant={enrollmentStandingTone(standing)}>
                          {ENROLLMENT_STANDING_LABELS[standing]}
                        </Badge>
                      </span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function standingRank(standing: ReturnType<typeof enrollmentStanding>): number {
  switch (standing) {
    case "ending":
      return 0;
    case "starting":
      return 1;
    case "covered":
      return 2;
    case "declined":
      return 3;
    default:
      return 4;
  }
}

function describeEntry(
  entry: BenefitEnrollmentListRow,
  standing: ReturnType<typeof enrollmentStanding>,
): string {
  switch (standing) {
    case "declined":
      return entry.waivedReason
        ? `Declined ${formatUnixDate(entry.effectiveFrom)} · ${entry.waivedReason}`
        : `Declined ${formatUnixDate(entry.effectiveFrom)}`;
    case "ended":
      return `${coverageTierLabel(entry.coverageTier)} · ended ${
        entry.effectiveTo ? formatUnixDate(entry.effectiveTo) : ""
      }`.trim();
    case "starting":
      return `${coverageTierLabel(entry.coverageTier)} · from ${formatUnixDate(entry.effectiveFrom)}`;
    case "ending":
      return `${coverageTierLabel(entry.coverageTier)} · until ${
        entry.effectiveTo ? formatUnixDate(entry.effectiveTo) : ""
      }`.trim();
    default:
      return `${coverageTierLabel(entry.coverageTier)} · since ${formatUnixDate(entry.effectiveFrom)}`;
  }
}
