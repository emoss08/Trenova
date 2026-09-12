import { translate } from "@trenova/shared/i18n/runtime";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { ptoStatusChoices, ptoTypeChoices } from "@/lib/choices";
import type { WorkerPTORow } from "@/lib/graphql/worker-table";
import { PTOStatusBadge, PTOTypeBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDate, inclusiveDays } from "@trenova/shared/lib/date";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { PTOStatus } from "@trenova/shared/types/worker";

type PTODecision = {
  actor: string;
  verb: string;
  note: string | null;
};

type PTOActorRef = { name: string } | null | undefined;

export type PTODecisionSource = {
  status: PTOStatus;
  rejectionReason?: string | null;
  cancellationReason?: string | null;
  approver?: PTOActorRef;
  rejector?: PTOActorRef;
  cancelledBy?: PTOActorRef;
};

export type PTODaysSource = {
  days?: string | null;
  startDate: number;
  endDate: number;
};

export function ptoDecision(pto: PTODecisionSource): PTODecision | null {
  switch (pto.status) {
    case "Approved":
      return pto.approver ? { actor: pto.approver.name, verb: "Approved by", note: null } : null;
    case "Rejected":
      return pto.rejector
        ? { actor: pto.rejector.name, verb: "Rejected by", note: pto.rejectionReason ?? null }
        : null;
    case "Cancelled":
      return pto.cancelledBy
        ? {
            actor: pto.cancelledBy.name,
            verb: "Cancelled by",
            note: pto.cancellationReason ?? null,
          }
        : null;
    case "Requested":
      return null;
    default:
      return null;
  }
}

export function ptoDaysOf(pto: PTODaysSource): number {
  const stored = pto.days != null ? Number(pto.days) : Number.NaN;
  if (Number.isFinite(stored) && stored > 0) {
    return stored;
  }
  return inclusiveDays(pto.startDate, pto.endDate);
}

function DecisionCell({ pto }: { pto: WorkerPTORow }) {
  const decision = ptoDecision(pto);
  if (!decision) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <div className="flex min-w-0 flex-col leading-tight">
      <span className="truncate">
        <span className="text-muted-foreground">{decision.verb} </span>
        {decision.actor}
      </span>
      {decision.note ? (
        <span className="text-muted-foreground truncate text-xs" title={decision.note}>
          {decision.note}
        </span>
      ) : null}
    </div>
  );
}

export function getColumns(): ColumnDef<WorkerPTORow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <PTOStatusBadge status={status} />;
      },
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ptoStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "worker.firstName",
      header: "First Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "worker.firstName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "worker.lastName",
      header: "Last Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "worker.lastName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => {
        const type = row.original.type;
        return <PTOTypeBadge type={type} />;
      },
      meta: {
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ptoTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "startDate",
      header: "Start",
      cell: ({ row }) => (
        <span className="font-table tracking-tight tabular-nums">
          {formatUnixDate(row.original.startDate)}
        </span>
      ),
      meta: {
        apiField: "startDate",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "endDate",
      header: "End",
      cell: ({ row }) => (
        <span className="font-table tracking-tight tabular-nums">
          {formatUnixDate(row.original.endDate)}
        </span>
      ),
      meta: {
        apiField: "endDate",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "days",
      header: "Days",
      cell: ({ row }) => (
        <span className="font-table tabular-nums">
          {ptoDaysOf(row.original)}
          {row.original.autoApproved ? (
            <Badge variant="outline" className="ml-1.5 px-1 py-0 text-[10px]">
              {translate("Auto")}
            </Badge>
          ) : null}
        </span>
      ),
      meta: {
        apiField: "days",
        filterable: false,
        sortable: true,
        exportValue: (row: WorkerPTORow) => ptoDaysOf(row),
      },
    },
    {
      accessorKey: "balanceAfterDays",
      header: "Balance After",
      cell: ({ row }) =>
        row.original.balanceAfterDays != null ? (
          <span className="font-table tabular-nums">
            {Number(row.original.balanceAfterDays).toFixed(2)}
          </span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      meta: {
        apiField: "balanceAfterDays",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "reason",
      header: "Reason",
      cell: ({ row }) => (
        <span className="block max-w-[240px] truncate" title={row.original.reason}>
          {row.original.reason}
        </span>
      ),
      meta: {
        apiField: "reason",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "decision",
      header: "Decision",
      cell: ({ row }) => <DecisionCell pto={row.original} />,
      enableSorting: false,
      meta: {
        apiField: "decision",
        filterable: false,
        sortable: false,
        exportValue: (row: WorkerPTORow) => {
          const decision = ptoDecision(row);
          if (!decision) return "";
          return decision.note
            ? `${decision.verb} ${decision.actor}: ${decision.note}`
            : `${decision.verb} ${decision.actor}`;
        },
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      cell: ({ row }) => {
        if (!row.original.createdAt) return "-";

        return (
          <HoverCardTimestamp
            className="font-table tracking-tight"
            timestamp={row.original.createdAt}
          />
        );
      },
    },
  ];
}
