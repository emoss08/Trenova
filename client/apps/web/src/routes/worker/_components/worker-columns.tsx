/* eslint-disable react-refresh/only-export-components */
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { EditableDriverTypeBadge } from "@/components/editable-driver-type-badge";
import { EditableWorkerTypeBadge } from "@/components/editable-worker-type-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { drugAlcoholStatusMeta } from "@trenova/shared/lib/drug-alcohol";
import { safetyRatingMeta } from "@trenova/shared/lib/safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { WORKER_LEAVE_TYPE_LABELS } from "@trenova/shared/types/worker";
import { formatUnixDate, getTodayDate } from "@trenova/shared/lib/date";
import { formatTenure } from "@trenova/shared/lib/tenure";
import { cn } from "@trenova/shared/lib/utils";
import {
  complianceStatusChoices,
  driverTypeChoices,
  drugAlcoholStatusChoices,
  safetyRatingChoices,
  statusChoices,
  trainingHealthChoices,
  workerTypeChoices,
} from "@/lib/choices";
import { patchWorker } from "@/lib/graphql/worker-mutations";
import type { WorkerRow } from "@/lib/graphql/worker-table";
import type { DriverType, WorkerType } from "@trenova/shared/types/worker";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";
import { toast } from "sonner";

function WorkerTypeCell({ row }: { row: WorkerRow }) {
  const t = useT();

  const queryClient = useQueryClient();

  const handleTypeChange = useCallback(
    async (newType: WorkerType) => {
      if (!row.id) return;
      await patchWorker(row.id, {
        type: newType,
      });

      await queryClient.invalidateQueries({
        queryKey: ["worker-list"],
      });

      toast.success(t("Worker type updated successfully"));
    },
    [row.id, queryClient, t],
  );

  return (
    <EditableWorkerTypeBadge
      workerType={row.type}
      options={workerTypeChoices}
      onWorkerTypeChange={handleTypeChange}
    />
  );
}

function DriverTypeCell({ row }: { row: WorkerRow }) {
  const t = useT();

  const queryClient = useQueryClient();

  const handleDriverTypeChange = useCallback(
    async (newDriverType: DriverType) => {
      if (!row.id) return;
      await patchWorker(row.id, {
        driverType: newDriverType,
      });

      await queryClient.invalidateQueries({
        queryKey: ["worker-list"],
      });

      toast.success(t("Driver type updated successfully"));
    },
    [row.id, queryClient, t],
  );

  return (
    <EditableDriverTypeBadge
      driverType={row.driverType}
      options={driverTypeChoices}
      onDriverTypeChange={handleDriverTypeChange}
    />
  );
}

function TenureCell({
  hireDate,
  terminationDate,
}: {
  hireDate?: number | null;
  terminationDate?: number | null;
}) {
  const label = formatTenure(hireDate, terminationDate, getTodayDate());
  const ended = Boolean(terminationDate && terminationDate > 0);
  return (
    <span
      className={cn("tabular-nums", ended && "text-muted-foreground")}
      title={
        hireDate
          ? `Hired ${formatUnixDate(hireDate)}${ended ? `, left ${formatUnixDate(terminationDate)}` : ""}`
          : undefined
      }
    >
      {label}
    </span>
  );
}

export function getColumns(t: TranslateFn): ColumnDef<WorkerRow>[] {
  return [
    {
      accessorKey: "wholeName",
      header: t("Name"),
      cell: ({ row }) => {
        const { firstName, lastName, wholeName } = row.original;
        return <p>{wholeName || `${firstName} ${lastName}`}</p>;
      },
      meta: {
        label: t("Name"),
        apiField: "wholeName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5">
          <StatusBadge status={row.original.status} />
          {row.original.leaveType ? (
            <Badge
              variant="outline"
              className="border-amber-500/40 bg-amber-500/10 px-1.5 py-0 text-[10px] text-amber-700 dark:text-amber-400"
              title={t("On leave")}
            >
              {t("{0} leave", WORKER_LEAVE_TYPE_LABELS[row.original.leaveType])}
            </Badge>
          ) : null}
        </span>
      ),
      size: 180,
      minSize: 120,
      maxSize: 220,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "tenure",
      header: t("Tenure"),
      cell: ({ row }) => (
        <TenureCell
          hireDate={row.original.profile?.hireDate}
          terminationDate={row.original.profile?.terminationDate}
        />
      ),
      size: 100,
      meta: { label: t("Tenure") },
    },
    {
      accessorKey: "type",
      header: t("Type"),
      cell: ({ row }) => <WorkerTypeCell row={row.original} />,
      size: 140,
      minSize: 120,
      maxSize: 160,
      meta: {
        label: t("Type"),
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: workerTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "driverType",
      header: t("Driver Type"),
      cell: ({ row }) => <DriverTypeCell row={row.original} />,
      size: 140,
      minSize: 120,
      maxSize: 160,
      meta: {
        label: t("Driver Type"),
        apiField: "driverType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: driverTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "fleetCode",
      header: t("Fleet Code"),
      cell: ({ row }) => {
        const { fleetCode } = row.original;
        if (!fleetCode) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<NonNullable<WorkerRow["fleetCode"]>, WorkerRow>
            entity={fleetCode}
            config={{
              basePath: "/dispatch/configuration-files/fleet-codes",
              getId: (fleetCode) => fleetCode.id,
              getDisplayText: (fleetCode) => fleetCode.code ?? "",
              color: {
                getColor: (fleetCode) => fleetCode.color,
              },
              getHeaderText: "Fleet Code",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        label: t("Fleet Code"),
        apiField: "fleetCode.code",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "state",
      header: t("State"),
      cell: ({ row }) => {
        const { state } = row.original;
        if (!state) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<NonNullable<WorkerRow["state"]>, WorkerRow>
            entity={state}
            config={{
              basePath: "#",
              getId: (state) => state.id,
              getDisplayText: (state) => state.abbreviation ?? "",
              getHeaderText: "State",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        label: t("State"),
        apiField: "state.abbreviation",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "profile.complianceStatus",
      header: t("Compliance"),
      cell: ({ row }) => {
        const complianceStatus = row.original.profile?.complianceStatus;
        if (!complianceStatus) {
          return <p className="text-muted-foreground">-</p>;
        }

        return <StatusBadge status={complianceStatus} />;
      },
      size: 130,
      minSize: 110,
      maxSize: 150,
      meta: {
        label: t("Compliance"),
        apiField: "profile.complianceStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: complianceStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "profile.trainingHealth",
      header: t("Training"),
      cell: ({ row }) => {
        const health = row.original.profile?.trainingHealth;
        if (!health) {
          return <p className="text-muted-foreground">-</p>;
        }
        return <TrainingHealthBadge health={health} />;
      },
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        label: t("Training"),
        apiField: "profile.trainingHealth",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: trainingHealthChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "profile.safetyRating",
      header: t("Safety"),
      cell: ({ row }) => {
        const profile = row.original.profile;
        if (!profile) {
          return <p className="text-muted-foreground">-</p>;
        }
        const meta = safetyRatingMeta(profile.safetyRating);
        return (
          <div className="flex items-center gap-2">
            <Badge variant={meta.badgeVariant}>{t(meta.label)}</Badge>
            <span className="text-muted-foreground text-xs tabular-nums">
              {profile.safetyScore}
            </span>
          </div>
        );
      },
      size: 140,
      minSize: 120,
      maxSize: 170,
      meta: {
        label: t("Safety"),
        apiField: "profile.safetyRating",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: safetyRatingChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "profile.drugAlcoholStatus",
      header: t("D&A"),
      cell: ({ row }) => {
        const status = row.original.profile?.drugAlcoholStatus;
        if (!status) {
          return <p className="text-muted-foreground">-</p>;
        }
        const meta = drugAlcoholStatusMeta(status);
        return (
          <Badge variant={meta.tone} title={meta.detail}>
            {t(meta.label)}
          </Badge>
        );
      },
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        label: t("D&A"),
        apiField: "profile.drugAlcoholStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: drugAlcoholStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "nextCredentialExpiry",
      accessorKey: "profile.nextCredentialExpiry",
      header: t("Expires Next"),
      cell: ({ row }) => {
        const expiry = row.original.profile?.nextCredentialExpiry;
        if (!expiry) {
          return <p className="text-muted-foreground">{t("Nothing expiring")}</p>;
        }
        const days = Math.ceil((expiry - getTodayDate()) / 86_400);
        return (
          <div className="flex flex-col">
            <span className="text-sm">{formatUnixDate(expiry)}</span>
            <span
              className={cn(
                "text-xs",
                days < 0 && "text-red-600 dark:text-red-400",
                days >= 0 && days <= 30 && "text-amber-600 dark:text-amber-400",
                days > 30 && "text-muted-foreground",
              )}
            >
              {days < 0
                ? t("Lapsed {0}d ago", Math.abs(days))
                : days === 0
                  ? t("Expires today")
                  : t("in {0}d", days)}
            </span>
          </div>
        );
      },
      size: 150,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: t("Expires Next"),
        apiField: "profile.nextCredentialExpiry",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "lte",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => {
        return <HoverCardTimestamp className="shrink-0" timestamp={row.original.createdAt} />;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        label: t("Created At"),
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
