import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { driverTypeChoices, statusChoices, workerTypeChoices } from "@/lib/choices";
import { patchWorker } from "@/lib/graphql/worker-mutations";
import { workerTableGraphQLConfig, type WorkerRow } from "@/lib/graphql/worker-table";
import { usePermission } from "@/hooks/use-permission";
import { PlanFeature } from "@/lib/plan-feature";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { worker } from "@/lib/queries/worker";
import type { DataTableToolbarSlots, DockAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Worker } from "@trenova/shared/types/worker";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useRouter } from "@tanstack/react-router";
import { useSearchValue } from "@trenova/shared/hooks/use-search-state";
import { searchString } from "@trenova/shared/lib/router/search-codec";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  CheckCircleIcon,
  GraduationHat01Icon,
  Truck01Icon,
  User01Icon,
} from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { BulkAssignTrainingDialog } from "./bulk-assign-training-dialog";
import { MissingCredentialMenu } from "./missing-credential-menu";
import { getColumns } from "./worker-columns";
import { NEW_WORKER_PATH, workerPath } from "../use-worker-state";
import { captureRosterContext, rosterLabel, useRosterContextStore } from "../roster-context-store";

export const MISSING_CREDENTIAL_PARAM = "missing";

export default function WorkerTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const [trainingTargets, setTrainingTargets] = useState<WorkerRow[] | null>(null);
  const { allowed: readsCredentials } = usePermission(Resource.WorkerCredential, Operation.Read);
  const trainingOnPlan = usePlanFeature(PlanFeature.WorkforceTalent).available;
  const navigate = useNavigate();
  const router = useRouter();
  const setRosterContext = useRosterContextStore((state) => state.setContext);
  const [missingParam, setMissing] = useSearchValue(MISSING_CREDENTIAL_PARAM, searchString);
  const withoutCredential = missingParam || null;
  const setWithoutCredential = (code: string | null) => void setMissing(code, { history: "push" });
  const graphql = useMemo(() => workerTableGraphQLConfig(withoutCredential), [withoutCredential]);
  const choices = useQuery({
    ...worker.credentialTypeChoices(),
    enabled: withoutCredential !== null,
  });
  const missingName = choices.data?.find((type) => type.code === withoutCredential)?.name ?? null;
  const toolbar: DataTableToolbarSlots | undefined = readsCredentials
    ? {
        trailing: (
          <MissingCredentialMenu value={withoutCredential} onChange={setWithoutCredential} />
        ),
        chips:
          withoutCredential === null
            ? undefined
            : {
                items: [
                  {
                    key: "withoutCredential",
                    label: missingName ? t("Missing {0}", missingName) : t("Missing a credential"),
                    onRemove: () => setWithoutCredential(null),
                  },
                ],
                onClear: () => setWithoutCredential(null),
              },
      }
    : undefined;

  const handleBulkStatusUpdate = useCallback(
    async (rows: WorkerRow[], status: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
          status: status as Worker["status"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: t("Updating status..."),
        success: t(
          "{0, plural, one {Updated # worker successfully} other {Updated # workers successfully}}",
          rows.length,
        ),
        error: t("Failed to update status"),
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient, t],
  );

  const handleBulkTypeUpdate = useCallback(
    async (rows: WorkerRow[], type: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
          type: type as Worker["type"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: t("Updating worker type..."),
        success: t(
          "{0, plural, one {Updated # worker successfully} other {Updated # workers successfully}}",
          rows.length,
        ),
        error: t("Failed to update worker type"),
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient, t],
  );

  const handleBulkDriverTypeUpdate = useCallback(
    async (rows: WorkerRow[], driverType: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
          driverType: driverType as Worker["driverType"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: t("Updating driver type..."),
        success: t(
          "{0, plural, one {Updated # worker successfully} other {Updated # workers successfully}}",
          rows.length,
        ),
        error: t("Failed to update driver type"),
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<WorkerRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
      {
        id: "type-update",
        type: "select",
        label: t("Update type"),
        loadingLabel: t("Updating..."),
        icon: User01Icon,
        options: workerTypeChoices,
        onSelect: handleBulkTypeUpdate,
        clearSelectionOnSuccess: true,
      },
      {
        id: "driver-type-update",
        type: "select",
        label: t("Update driver type"),
        loadingLabel: t("Updating..."),
        icon: Truck01Icon,
        options: driverTypeChoices,
        onSelect: handleBulkDriverTypeUpdate,
        clearSelectionOnSuccess: true,
      },
      ...(trainingOnPlan
        ? [
            {
              id: "assign-training",
              label: t("Assign training"),
              icon: GraduationHat01Icon,
              onClick: (rows: WorkerRow[]) => setTrainingTargets(rows),
            },
          ]
        : []),
    ],
    [handleBulkStatusUpdate, handleBulkTypeUpdate, handleBulkDriverTypeUpdate, trainingOnPlan, t],
  );

  return (
    <>
      <DataTable<WorkerRow>
        name="Worker"
        emptyTitle={t("No workers yet")}
        queryKey="worker-list"
        resource={Resource.Worker}
        columns={columns}
        graphql={graphql}
        toolbar={toolbar}
        dockActions={dockActions}
        enableRowSelection
        onAddRecord={() => void navigate({ to: NEW_WORKER_PATH })}
        onRowClick={(row) => {
          const workerId = row.original.id;
          if (!workerId) return;
          setRosterContext(
            captureRosterContext(queryClient, {
              workerId,
              graphql,
              search: router.state.location.searchStr,
              label: (options) =>
                rosterLabel(t, {
                  options,
                  missingCredential: withoutCredential === null ? null : { name: missingName },
                  now: getTodayDate(),
                }),
            }),
          );
          void navigate({ to: workerPath(workerId) });
        }}
      />
      {trainingOnPlan ? (
        <BulkAssignTrainingDialog
          open={trainingTargets !== null}
          onOpenChange={(open) => {
            if (!open) setTrainingTargets(null);
          }}
          workers={trainingTargets ?? []}
        />
      ) : null}
    </>
  );
}
