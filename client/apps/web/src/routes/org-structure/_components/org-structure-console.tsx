import { usePermission } from "@/hooks/use-permission";
import {
  fetchHeadcount,
  fetchJobPositions,
  HEADCOUNT_KEY,
  JOB_POSITIONS_KEY,
  type HeadcountRow,
  type JobPositionRow,
} from "@/lib/graphql/org-structure";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { headcountShare, jobDepartmentLabel } from "@trenova/shared/lib/org-structure";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon } from "lucide-react";
import { useState } from "react";
import { DelegationPanel } from "./delegation-panel";
import { PositionDialog } from "./position-dialog";

export default function OrgStructureConsole() {
  const { allowed: canRead } = usePermission(Resource.JobPosition, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.JobPosition, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.JobPosition, Operation.Update);
  const [dialog, setDialog] = useState<{ position: JobPositionRow | null } | null>(null);

  const headcountQuery = useQuery({
    queryKey: [HEADCOUNT_KEY],
    queryFn: ({ signal }) => fetchHeadcount({ signal }),
    enabled: canRead,
  });
  const positionsQuery = useQuery({
    queryKey: [JOB_POSITIONS_KEY],
    queryFn: ({ signal }) => fetchJobPositions(undefined, { signal }),
    enabled: canRead,
  });

  if (!canRead) return null;

  if (headcountQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  const headcount = headcountQuery.data;
  const positions = positionsQuery.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      {headcount ? (
        <>
          <section className="grid grid-cols-3 gap-3">
            <Figure label="Active workers" value={String(headcount.activeTotal)} />
            <Figure
              label="Driving positions"
              value={String(headcount.driverTotal)}
              detail="Everyone who needs a CDL"
            />
            <Figure
              label="Off the roster"
              value={String(headcount.terminated)}
              detail="Terminated or inactive"
            />
          </section>

          <div className="grid gap-4 lg:grid-cols-3">
            <HeadcountBreakdown
              title="By terminal"
              rows={headcount.byFleet}
              total={headcount.activeTotal}
            />
            <HeadcountBreakdown
              title="By department"
              rows={headcount.byDepartment}
              total={headcount.activeTotal}
              labelOf={(row) => jobDepartmentLabel(row.label)}
            />
            <HeadcountBreakdown
              title="By position"
              rows={headcount.byPosition}
              total={headcount.activeTotal}
            />
          </div>
        </>
      ) : null}

      <section className="rounded-lg border p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-medium">Positions</h3>
            <p className="text-muted-foreground text-xs">
              What somebody does, as distinct from where they do it. A terminal is a fleet code;
              this is the title the headcount is counted by.
            </p>
          </div>
          {canCreate ? (
            <Button size="sm" onClick={() => setDialog({ position: null })}>
              <PlusIcon className="size-3.5" />
              Add a position
            </Button>
          ) : null}
        </div>

        {positionsQuery.isLoading ? (
          <Skeleton className="mt-3 h-24 w-full" />
        ) : positions.length === 0 ? (
          <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
            No positions yet. Until there are, the roster can only be counted by terminal.
          </p>
        ) : (
          <ul className="mt-3 flex flex-col gap-1.5">
            {positions.map((position) => (
              <li
                key={position.id}
                className="flex flex-wrap items-center justify-between gap-2 border-t pt-1.5 text-xs first:border-t-0 first:pt-0"
              >
                <span className="flex min-w-0 flex-wrap items-center gap-2">
                  <span className="font-medium">{position.title}</span>
                  <span className="text-muted-foreground tabular-nums">{position.code}</span>
                  <Badge variant="secondary">{jobDepartmentLabel(position.department)}</Badge>
                  {position.isDrivingPosition ? <Badge variant="info">Driving</Badge> : null}
                  {position.flsaExempt ? <Badge variant="purple">Exempt</Badge> : null}
                  {position.status !== "Active" ? <Badge variant="inactive">Archived</Badge> : null}
                  {position.reportsTo ? (
                    <span className="text-muted-foreground truncate">
                      reports to {position.reportsTo.title}
                    </span>
                  ) : null}
                </span>
                {canUpdate ? (
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => setDialog({ position })}
                    aria-label={`Edit ${position.title}`}
                  >
                    Edit
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </section>

      <DelegationPanel />

      <PositionDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        position={dialog?.position ?? null}
        positions={positions}
      />
    </div>
  );
}

function HeadcountBreakdown({
  title,
  rows,
  total,
  labelOf,
}: {
  title: string;
  rows: HeadcountRow[];
  total: number;
  labelOf?: (row: HeadcountRow) => string;
}) {
  return (
    <section className="rounded-lg border p-4">
      <h3 className="text-sm font-medium">{title}</h3>
      {rows.length === 0 ? (
        <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
          Nobody on the roster.
        </p>
      ) : (
        <ul className="mt-3 flex flex-col gap-2">
          {rows.map((row) => (
            <li key={row.key || row.label} className="text-xs">
              <div className="flex items-center justify-between gap-2">
                <span className="flex min-w-0 items-center gap-2">
                  {row.color ? (
                    <span
                      className="size-2 shrink-0 rounded-full"
                      style={{ backgroundColor: row.color }}
                    />
                  ) : null}
                  <span className="truncate">{labelOf ? labelOf(row) : row.label}</span>
                </span>
                <span className="text-muted-foreground shrink-0 tabular-nums">
                  {row.workers}
                  {row.terminated > 0 ? ` · ${row.terminated} left` : ""}
                </span>
              </div>
              <div className="bg-muted mt-1 h-1.5 w-full overflow-hidden rounded-full">
                <div
                  className={cn("bg-primary/60 h-full rounded-full")}
                  style={{ width: `${headcountShare(row.workers, total)}%` }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Figure({ label, value, detail }: { label: string; value: string; detail?: string }) {
  return (
    <div className="rounded-lg border px-3 py-2">
      <p className="text-muted-foreground text-[11px]">{label}</p>
      <p className="text-lg font-semibold tabular-nums">{value}</p>
      {detail ? <p className="text-muted-foreground text-[11px]">{detail}</p> : null}
    </div>
  );
}
