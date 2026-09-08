import { usePermission } from "@/hooks/use-permission";
import {
  HEADCOUNT_KEY,
  JOB_POSITIONS_KEY,
  updateJobPosition,
  type JobPositionRow,
} from "@/lib/graphql/org-structure";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { DelegationPanel } from "./delegation-panel";
import { OrgAside } from "./org-aside";
import { OrgOverview } from "./org-overview";
import { OrgStructureEmpty } from "./org-structure-empty";
import { OrgStructureSkeleton } from "./org-structure-skeleton";
import { PositionDialog } from "./position-dialog";
import { PositionHoldersSheet } from "./position-holders-sheet";
import { PositionTree } from "./position-tree";
import {
  delegationsGivenQuery,
  delegationsReceivedQuery,
  headcountQuery,
  jobPositionsQuery,
} from "./queries";

const EMPTY_POSITIONS: JobPositionRow[] = [];

type DialogState = { position: JobPositionRow | null; reportsTo: string | null };

export default function OrgStructureConsole() {
  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.JobPosition, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.JobPosition, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.JobPosition, Operation.Update);
  const { allowed: canReadCover } = usePermission(Resource.ApprovalDelegation, Operation.Read);
  const userId = useAuthStore((state) => state.user?.id);
  const [dialog, setDialog] = useState<DialogState | null>(null);
  const [holders, setHolders] = useState<JobPositionRow | null>(null);
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const headcount = useQuery({ ...headcountQuery(), enabled: canRead });
  const positionsResult = useQuery({ ...jobPositionsQuery(), enabled: canRead });
  const coverEnabled = canRead && canReadCover && Boolean(userId);
  const given = useQuery({ ...delegationsGivenQuery(userId ?? ""), enabled: coverEnabled });
  const received = useQuery({ ...delegationsReceivedQuery(userId ?? ""), enabled: coverEnabled });

  // Moving a position is an ordinary update carrying everything else as it
  // was; there is no move mutation, and inventing one server-side for a
  // single field would be a second path to keep the reporting-line check on.
  const move = useMutation({
    mutationFn: ({
      position,
      reportsToPositionId,
    }: {
      position: JobPositionRow;
      reportsToPositionId: string | null;
    }) =>
      updateJobPosition({
        id: position.id,
        version: position.version,
        code: position.code,
        title: position.title,
        description: position.description ?? undefined,
        department: position.department,
        flsaExempt: position.flsaExempt,
        isDrivingPosition: position.isDrivingPosition,
        reportsToPositionId: reportsToPositionId ?? undefined,
        status: position.status === "Active" ? "Active" : "Inactive",
      }),
    onSuccess: (_data, { position, reportsToPositionId }) => {
      const parent = reportsToPositionId
        ? positionsResult.data?.find((row) => row.id === reportsToPositionId)?.title
        : null;
      toast.success(
        parent
          ? `${position.title} now reports to ${parent}`
          : `${position.title} is now top level`,
      );
      void queryClient.invalidateQueries({ queryKey: [JOB_POSITIONS_KEY] });
      void queryClient.invalidateQueries({ queryKey: [HEADCOUNT_KEY] });
    },
    onError: (error: Error) => toast.error("Could not move it", { description: error.message }),
  });

  const positions = positionsResult.data ?? EMPTY_POSITIONS;
  // The overview counts cover the user is party to on either side; a
  // delegation from the user to themselves would appear in both lists.
  const delegations = useMemo(() => {
    if (!given.data || !received.data) return undefined;
    const seen = new Set<string>();
    return [...given.data, ...received.data].filter((row) => {
      if (seen.has(row.id)) return false;
      seen.add(row.id);
      return true;
    });
  }, [given.data, received.data]);

  if (!canRead) return null;

  if (headcount.isLoading || positionsResult.isLoading) {
    return <OrgStructureSkeleton showCover={canReadCover} />;
  }

  const summary = headcount.data;
  const noRoster = summary !== undefined && summary.activeTotal === 0 && positions.length === 0;
  const openAdd = (reportsTo: string | null) => setDialog({ position: null, reportsTo });
  const openEdit = (position: JobPositionRow) => setDialog({ position, reportsTo: null });
  // The only way in without a chart to hang it on; absent for anybody who
  // could not save what the dialog would ask for.
  const addFirstPosition = canCreate ? () => openAdd(null) : undefined;

  return (
    <div className="flex flex-col gap-4">
      <OrgOverview
        headcount={summary}
        positions={positionsResult.data}
        delegations={delegations}
        showCover={canReadCover}
        now={now}
      />

      {noRoster ? (
        <OrgStructureEmpty
          title="Nothing to count yet"
          description={
            "Positions are the titles the roster is counted by. Add one, then give workers a " +
            "position from their record and the chart fills in."
          }
          onAddPosition={addFirstPosition}
        />
      ) : (
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
          <div className="flex min-w-0 flex-col gap-4">
            {positions.length === 0 ? (
              <OrgStructureEmpty
                title="No positions yet"
                description={
                  "Until there are, the roster can only be counted by terminal. Add one, then " +
                  "give workers a position from their record and the chart fills in."
                }
                onAddPosition={addFirstPosition}
              />
            ) : (
              <PositionTree
                positions={positions}
                byPosition={summary?.byPosition ?? []}
                canCreate={canCreate}
                canUpdate={canUpdate}
                moving={move.isPending ? move.variables.position.id : null}
                onAdd={openAdd}
                onEdit={openEdit}
                onMove={(position, reportsToPositionId) =>
                  move.mutate({ position, reportsToPositionId })
                }
                onOpenHolders={setHolders}
              />
            )}
            <DelegationPanel />
          </div>
          <OrgAside
            byFleet={summary?.byFleet ?? []}
            byDepartment={summary?.byDepartment ?? []}
            total={summary?.activeTotal ?? 0}
            positions={positions}
            byPosition={summary?.byPosition ?? []}
            canUpdate={canUpdate}
            onEdit={openEdit}
          />
        </div>
      )}

      <PositionDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        position={dialog?.position ?? null}
        positions={positions}
        defaultReportsToPositionId={dialog?.reportsTo ?? null}
      />
      <PositionHoldersSheet position={holders} onOpenChange={(open) => !open && setHolders(null)} />
    </div>
  );
}
