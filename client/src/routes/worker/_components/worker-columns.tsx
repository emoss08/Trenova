/* eslint-disable react-refresh/only-export-components */
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { EditableDriverTypeBadge } from "@/components/editable-driver-type-badge";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { EditableWorkerTypeBadge } from "@/components/editable-worker-type-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { StatusBadge } from "@/components/status-badge";
import {
  complianceStatusChoices,
  driverTypeChoices,
  statusChoices,
  workerTypeChoices,
} from "@/lib/choices";
import { apiService } from "@/services/api";
import type { FleetCode } from "@/types/fleet-code";
import type { UsState } from "@/types/us-state";
import type { DriverType, Worker, WorkerType } from "@/types/worker";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback } from "react";
import { toast } from "sonner";

function WorkerStatusCell({ row }: { row: Worker }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: Worker["status"]) => {
      if (!row.id) return;
      await apiService.workerService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["worker-list"],
      });

      toast.success("Worker status updated successfully");
    },
    [row.id, queryClient],
  );

  return (
    <EditableStatusBadge
      status={row.status}
      options={statusChoices}
      onStatusChange={handleStatusChange}
    />
  );
}

function WorkerTypeCell({ row }: { row: Worker }) {
  const queryClient = useQueryClient();

  const handleTypeChange = useCallback(
    async (newType: WorkerType) => {
      if (!row.id) return;
      await apiService.workerService.patch(row.id, {
        type: newType,
      });

      await queryClient.invalidateQueries({
        queryKey: ["worker-list"],
      });

      toast.success("Worker type updated successfully");
    },
    [row.id, queryClient],
  );

  return (
    <EditableWorkerTypeBadge
      workerType={row.type}
      options={workerTypeChoices}
      onWorkerTypeChange={handleTypeChange}
    />
  );
}

function DriverTypeCell({ row }: { row: Worker }) {
  const queryClient = useQueryClient();

  const handleDriverTypeChange = useCallback(
    async (newDriverType: DriverType) => {
      if (!row.id) return;
      await apiService.workerService.patch(row.id, {
        driverType: newDriverType,
      });

      await queryClient.invalidateQueries({
        queryKey: ["worker-list"],
      });

      toast.success("Driver type updated successfully");
    },
    [row.id, queryClient],
  );

  return (
    <EditableDriverTypeBadge
      driverType={row.driverType}
      options={driverTypeChoices}
      onDriverTypeChange={handleDriverTypeChange}
    />
  );
}

export function getColumns(): ColumnDef<Worker>[] {
  return [
    {
      accessorKey: "wholeName",
      header: "Name",
      cell: ({ row }) => {
        const { firstName, lastName, wholeName } = row.original;
        return <p>{wholeName || `${firstName} ${lastName}`}</p>;
      },
      meta: {
        label: "Name",
        apiField: "wholeName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <WorkerStatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => <WorkerTypeCell row={row.original} />,
      size: 140,
      minSize: 120,
      maxSize: 160,
      meta: {
        label: "Type",
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
      header: "Driver Type",
      cell: ({ row }) => <DriverTypeCell row={row.original} />,
      size: 140,
      minSize: 120,
      maxSize: 160,
      meta: {
        label: "Driver Type",
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
      header: "Fleet Code",
      cell: ({ row }) => {
        const { fleetCode } = row.original;
        if (!fleetCode) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<FleetCode, Worker>
            entity={fleetCode}
            config={{
              basePath: "/dispatch/configuration-files/fleet-codes",
              getId: (fleetCode) => fleetCode.id,
              getDisplayText: (fleetCode) => fleetCode.code,
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
        label: "Fleet Code",
        apiField: "fleetCode.code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "state",
      header: "State",
      cell: ({ row }) => {
        const { state } = row.original;
        if (!state) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<UsState, Worker>
            entity={state}
            config={{
              basePath: "#",
              getId: (state) => state.id,
              getDisplayText: (state) => state.abbreviation,
              getHeaderText: "State",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        label: "State",
        apiField: "state.abbreviation",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "profile.complianceStatus",
      header: "Compliance",
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
        label: "Compliance",
        apiField: "profile.complianceStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: complianceStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return (
          <HoverCardTimestamp
            className="shrink-0"
            timestamp={row.original.createdAt}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        label: "Created At",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
