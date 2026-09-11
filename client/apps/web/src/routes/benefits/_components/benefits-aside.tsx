import { useT } from "@trenova/shared/i18n/use-t";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { BenefitEnrollmentListRow } from "@/lib/graphql/benefits";
import { endingSoon, recentDeclines, startingSoon } from "@/lib/benefits-console";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { CalendarClockIcon, CalendarPlusIcon, CircleSlashIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { enrollmentWorkerName } from "./plan-enrollments-sheet";

const SHOWN_LIMIT = 6;

function workerHref(workerId: string): string {
  return `/hr/workers?entityId=${workerId}&modType=edit`;
}

function Loading() {
  return (
    <div className="flex flex-col gap-2 p-3">
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-4 w-1/2" />
    </div>
  );
}

function EntryList({
  entries,
  detail,
}: {
  entries: readonly BenefitEnrollmentListRow[];
  detail: (entry: BenefitEnrollmentListRow) => string;
}) {
  const t = useT();

  const shown = entries.slice(0, SHOWN_LIMIT);
  const hidden = entries.length - shown.length;
  return (
    <>
      <ul className="divide-y">
        {shown.map((entry) => (
          <li key={entry.id}>
            <Link
              to={workerHref(entry.workerId)}
              className="hover:bg-accent flex flex-col gap-0.5 px-3 py-2 leading-tight transition-colors"
            >
              <span className="flex items-center justify-between gap-2">
                <span className="truncate text-xs font-medium">{enrollmentWorkerName(entry)}</span>
                <span className="text-muted-foreground text-2xs shrink-0 truncate">
                  {entry.benefitPlan?.name ?? "A plan"}
                </span>
              </span>
              <span className="text-muted-foreground text-2xs">{detail(entry)}</span>
            </Link>
          </li>
        ))}
      </ul>
      {hidden > 0 ? (
        <p className="text-muted-foreground text-2xs border-t px-3 py-1.5">{t("and {0} more", hidden)}</p>
      ) : null}
    </>
  );
}

type BenefitsAsideProps = {
  openEnrollments: readonly BenefitEnrollmentListRow[] | undefined;
  declined: readonly BenefitEnrollmentListRow[] | undefined;
  now: number;
};

/**
 * What is moving: cover arranged but not begun, cover with an end date in
 * sight, and the decisions not to take it. None of the three is visible from
 * the plan list, which only counts who is on a plan today.
 */
export function BenefitsAside({ openEnrollments, declined, now }: BenefitsAsideProps) {
  const t = useT();

  const starting = useMemo(() => startingSoon(openEnrollments ?? [], now), [openEnrollments, now]);
  const ending = useMemo(() => endingSoon(openEnrollments ?? [], now), [openEnrollments, now]);
  const declines = useMemo(() => recentDeclines(declined ?? []), [declined]);

  return (
    <aside className="flex min-w-0 flex-col gap-4">
      <SectionPanel
        title={t("Starting soon")}
        icon={<CalendarPlusIcon />}
        count={starting.length}
        help={t("Cover that has been arranged but has not begun, soonest first.")}
      >
        {!openEnrollments ? (
          <Loading />
        ) : starting.length === 0 ? (
          <SectionPanelQuiet>{t("No cover is waiting to begin.")}</SectionPanelQuiet>
        ) : (
          <EntryList
            entries={starting}
            detail={(entry) => `From ${formatUnixDate(entry.effectiveFrom)}`}
          />
        )}
      </SectionPanel>

      <SectionPanel
        title={t("Ending soon")}
        icon={<CalendarClockIcon />}
        count={ending.length}
        help={t("Cover with an end date inside the next thirty days, soonest first.")}
      >
        {!openEnrollments ? (
          <Loading />
        ) : ending.length === 0 ? (
          <SectionPanelQuiet>{t("Nothing ends in the next thirty days.")}</SectionPanelQuiet>
        ) : (
          <EntryList
            entries={ending}
            detail={(entry) =>
              entry.effectiveTo ? `Until ${formatUnixDate(entry.effectiveTo)}` : "Ending"
            }
          />
        )}
      </SectionPanel>

      <SectionPanel
        title={t("Recently declined")}
        icon={<CircleSlashIcon />}
        count={declines.length}
        help={t("People who waived a plan they were offered, most recent decision first.")}
      >
        {!declined ? (
          <Loading />
        ) : declines.length === 0 ? (
          <SectionPanelQuiet>{t("Nobody has declined cover.")}</SectionPanelQuiet>
        ) : (
          <EntryList
            entries={declines}
            detail={(entry) =>
              entry.waivedReason
                ? `${formatUnixDate(entry.effectiveFrom)} · ${entry.waivedReason}`
                : formatUnixDate(entry.effectiveFrom)
            }
          />
        )}
      </SectionPanel>
    </aside>
  );
}
