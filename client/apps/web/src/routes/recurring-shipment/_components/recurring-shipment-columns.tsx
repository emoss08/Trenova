import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import { describeCron } from "@/lib/cron";
import { cn } from "@/lib/utils";
import type { RecurringShipment, RecurringShipmentStatus } from "@/types/recurring-shipment";
import { type ColumnDef } from "@tanstack/react-table";
import { ArrowRightIcon } from "lucide-react";

export const recurringShipmentStatusChoices = [
  { value: "Active", label: "Active" },
  { value: "Paused", label: "Paused" },
  { value: "Expired", label: "Expired" },
];

const statusStyles: Record<RecurringShipmentStatus, string> = {
  Active: "border-green-600/30 bg-green-600/10 text-green-700 dark:text-green-400",
  Paused: "border-amber-600/30 bg-amber-600/10 text-amber-700 dark:text-amber-400",
  Expired: "border-muted-foreground/30 bg-muted text-muted-foreground",
};

export function RecurringShipmentStatusBadge({ status }: { status: RecurringShipmentStatus }) {
  return (
    <Badge variant="outline" className={cn("font-medium", statusStyles[status])}>
      {status}
    </Badge>
  );
}

function LaneCell({ row }: { row: RecurringShipment }) {
  const origin = row.originLocation?.name ?? row.originLocation?.code;
  const destination = row.destinationLocation?.name ?? row.destinationLocation?.code;

  if (!origin && !destination) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <div className="flex items-center gap-1.5 text-sm">
      <span className="truncate">{origin ?? "—"}</span>
      <ArrowRightIcon className="size-3 shrink-0 text-muted-foreground" />
      <span className="truncate">{destination ?? "—"}</span>
    </div>
  );
}

function ScheduleCell({ row }: { row: RecurringShipment }) {
  return (
    <div className="flex flex-col">
      <span className="text-sm">{describeCron(row.cronExpression)}</span>
      <span className="text-2xs text-muted-foreground">{row.timezone}</span>
    </div>
  );
}

export function getColumns(): ColumnDef<RecurringShipment>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <RecurringShipmentStatusBadge status={row.original.status} />,
      size: 110,
      minSize: 100,
      maxSize: 140,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: recurringShipmentStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      size: 220,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "customer",
      header: "Customer",
      cell: ({ row }) => (
        <span className="truncate">
          {row.original.customer?.name ?? row.original.customer?.code ?? "—"}
        </span>
      ),
      size: 180,
    },
    {
      id: "lane",
      header: "Lane",
      cell: ({ row }) => <LaneCell row={row.original} />,
      size: 280,
      minSize: 220,
    },
    {
      id: "schedule",
      header: "Schedule",
      cell: ({ row }) => <ScheduleCell row={row.original} />,
      size: 200,
    },
    {
      accessorKey: "nextOccurrenceAt",
      header: "Next Pickup",
      cell: ({ row }) =>
        row.original.nextOccurrenceAt ? (
          <HoverCardTimestamp timestamp={row.original.nextOccurrenceAt} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 180,
      meta: {
        apiField: "nextOccurrenceAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "generationCount",
      header: "Generated",
      cell: ({ row }) => {
        const count = row.original.generationCount ?? 0;
        const max = row.original.maxOccurrences;
        return (
          <span className="text-sm tabular-nums">
            {count}
            {max ? ` / ${max}` : ""}
          </span>
        );
      },
      size: 110,
      meta: {
        apiField: "generationCount",
        filterable: false,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
