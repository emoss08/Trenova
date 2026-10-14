import { usePermission } from "@/hooks/use-permission";
import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { AuditView } from "../../ai-control-tabs";
import { FigsSkeleton, HeroSkeleton, type FigSkeleton } from "../kit/tab-skeleton";

/** What the trail shows at first: when, agent, what, tool, outcome, person, record and more. */
const TRAIL_COLUMNS = 10;
/** Every column of an export, from when it was asked for to its download. */
export const EXPORT_COLUMNS = 11;

/** Whether the chain is signed, how far it is sealed, and what its last check found. */
const FIGURES: readonly FigSkeleton[] = [
  [40, 84, 120],
  [92, 104, 112],
  [80, 96, 184],
];

/**
 * The head of the audit trail while the chain's status is read: Nova's sentence with the
 * check it always offers, and the chain's three figures.
 */
export function ChainStatusSkeleton() {
  return (
    <div className="contents" aria-busy="true">
      <HeroSkeleton control={{ button: 124, note: 152 }} />
      <FigsSkeleton items={FIGURES} />
    </div>
  );
}

/**
 * The trail while its code is read: the range, agent and person it is narrowed to, the
 * evaluations switch and the export, over the table of events.
 */
export function AuditTrailSkeleton() {
  const { allowed: canExport } = usePermission(Resource.AIAuditTrail, Operation.Export);
  const { allowed: canPickAgent } = usePermission(Resource.AgentDefinition, Operation.Read);

  return (
    <div className="flex min-w-0 flex-col gap-4" aria-busy="true">
      <div className="flex flex-wrap items-center gap-2" aria-hidden>
        <Skeleton className="h-8 w-32" />
        {canPickAgent && <Skeleton className="h-8 w-36" />}
        <Skeleton className="h-8 w-56" />
        <div className="flex items-center gap-2">
          <Skeleton className="h-3.5 w-6 rounded-full" />
          <Skeleton className="h-3 w-28" />
        </div>
        {canExport && (
          <div className="ml-auto flex items-center gap-2">
            <Skeleton className="h-8 w-32" />
          </div>
        )}
      </div>
      <DataTableSkeleton columnCount={TRAIL_COLUMNS} aria-hidden />
    </div>
  );
}

/** The view under the chain, as the view switch picks it. */
function AuditBodySkeleton({ view }: { view: AuditView }) {
  return view === "exports" ? (
    <DataTableSkeleton columnCount={EXPORT_COLUMNS} aria-hidden />
  ) : (
    <AuditTrailSkeleton />
  );
}

/** The Audit trail tab while its code is read: the chain's head and the view it opens on. */
export function AuditTabSkeleton({ view }: { view: AuditView }) {
  const t = useT();

  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading the audit trail")}>
      <ChainStatusSkeleton />
      <AuditBodySkeleton view={view} />
    </div>
  );
}
