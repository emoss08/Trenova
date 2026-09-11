import { useT } from "@trenova/shared/i18n/use-t";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { HeadcountRow, JobPositionRow } from "@/lib/graphql/org-structure";
import { vacantPositions } from "@/lib/org-chart";
import { Button } from "@trenova/shared/components/ui/button";
import { headcountShare, jobDepartmentLabel } from "@trenova/shared/lib/org-structure";
import { cn } from "@trenova/shared/lib/utils";
import { BriefcaseIcon, Building2Icon, LayersIcon } from "lucide-react";
import { useMemo } from "react";

const SHOWN_LIMIT = 6;

type BreakdownProps = {
  rows: readonly HeadcountRow[];
  total: number;
  labelOf?: (row: HeadcountRow) => string;
};

/**
 * One way of cutting the roster, as bars against the whole. The driver count
 * rides beside the total because a safety director reads this list to know
 * how many CDLs a terminal holds, not how many desks.
 */
function Breakdown({ rows, total, labelOf }: BreakdownProps) {
  const t = useT();

  if (rows.length === 0) return <SectionPanelQuiet>{t("Nobody on the roster.")}</SectionPanelQuiet>;
  return (
    <ul className="divide-y">
      {rows.map((row) => {
        const label = labelOf ? labelOf(row) : row.label;
        return (
          <li key={row.key || row.label} className="flex flex-col gap-1.5 px-3 py-2">
            <div className="flex items-center justify-between gap-2 text-xs">
              <span className="flex min-w-0 items-center gap-2">
                <span
                  aria-hidden
                  className={cn(
                    "size-2 shrink-0 rounded-full",
                    !row.color && "bg-muted-foreground/40",
                  )}
                  style={row.color ? { backgroundColor: row.color } : undefined}
                />
                <span className="truncate font-medium">{label}</span>
                {row.drivers > 0 && row.drivers !== row.workers ? (
                  <span className="text-muted-foreground tabular-nums">
                    {t("· {0} driving", row.drivers)}
                  </span>
                ) : null}
                {row.staff > 0 ? (
                  <span className="text-muted-foreground tabular-nums">{t("· {0} staff", row.staff)}</span>
                ) : null}
                {row.terminated > 0 ? (
                  <span className="text-muted-foreground tabular-nums">
                    {t("· {0} left", row.terminated)}
                  </span>
                ) : null}
              </span>
              <span className="text-muted-foreground shrink-0 tabular-nums">
                {row.workers + row.staff}
              </span>
            </div>
            <div
              role="img"
              aria-label={`${label}: ${row.workers + row.staff} of ${total}`}
              className="bg-muted h-1 w-full overflow-hidden rounded-full"
            >
              <div
                className="bg-brand/50 h-full rounded-full transition-[width] duration-700 ease-out motion-reduce:transition-none"
                style={{ width: `${headcountShare(row.workers + row.staff, total)}%` }}
              />
            </div>
          </li>
        );
      })}
    </ul>
  );
}

type OrgAsideProps = {
  byFleet: readonly HeadcountRow[];
  byDepartment: readonly HeadcountRow[];
  total: number;
  positions: readonly JobPositionRow[];
  byPosition: readonly HeadcountRow[];
  canUpdate: boolean;
  onEdit: (position: JobPositionRow) => void;
};

export function OrgAside({
  byFleet,
  byDepartment,
  total,
  positions,
  byPosition,
  canUpdate,
  onEdit,
}: OrgAsideProps) {
  const t = useT();

  const vacant = useMemo(() => vacantPositions(positions, byPosition), [positions, byPosition]);
  const shown = vacant.slice(0, SHOWN_LIMIT);

  return (
    <aside className="flex min-w-0 flex-col gap-4">
      <SectionPanel
        title={t("By terminal")}
        icon={<Building2Icon />}
        hint={`${total} active`}
        help={t("Active workers by the terminal they are assigned to, biggest first. Front-office users are not on this roster.")}
      >
        <Breakdown rows={byFleet} total={total} />
      </SectionPanel>
      <SectionPanel
        title={t("By department")}
        icon={<LayersIcon />}
        help={t("Active workers by the department of the position they hold. Somebody with no position is not counted here.")}
      >
        <Breakdown
          rows={byDepartment}
          total={total}
          labelOf={(row) => jobDepartmentLabel(row.label)}
        />
      </SectionPanel>
      <SectionPanel
        title={t("Titles nobody holds yet")}
        icon={<BriefcaseIcon />}
        help={t("Positions still open that nobody on either roster holds. Assign a worker or a user from the position's holders view.")}
        hint={vacant.length > 0 ? String(vacant.length) : undefined}
      >
        {vacant.length === 0 ? (
          <SectionPanelQuiet>{t("Every open title has at least one person in it.")}</SectionPanelQuiet>
        ) : (
          <>
            <ul className="divide-y">
              {shown.map((position) => (
                <li key={position.id} className="flex items-center justify-between gap-2 px-3 py-2">
                  <span className="flex min-w-0 flex-col leading-tight">
                    <span className="truncate text-xs font-medium">{t(position.title)}</span>
                    <span className="text-muted-foreground text-2xs">
                      {position.code} · {jobDepartmentLabel(position.department)}
                    </span>
                  </span>
                  {canUpdate ? (
                    <Button size="xs" variant="ghost" onClick={() => onEdit(position)}>
                      {t("Edit")}
                    </Button>
                  ) : null}
                </li>
              ))}
            </ul>
            {vacant.length > shown.length ? (
              <p className="text-muted-foreground text-2xs border-t px-3 py-1.5">
                {t("and {0} more", vacant.length - shown.length)}
              </p>
            ) : null}
          </>
        )}
      </SectionPanel>
    </aside>
  );
}
