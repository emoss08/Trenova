import { RelativeTime } from "@/components/carrier-intelligence/relative-time";
import { StatusDot, type StatusTone } from "@/components/carrier-intelligence/status-dot";
import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { CARRIER_INTEL_VENDOR_STATES, INTEL_EMPTY_VALUE } from "@/lib/carrier-intelligence";
import { carrierPanelPath } from "@/lib/carrier-links";
import type { CarrierMonitoringEnrollmentRow } from "@/lib/graphql/carrier-monitoring-table";
import type {
  CarrierIntelDesiredState,
  CarrierIntelEnrollmentReason,
  CarrierIntelVendorState,
} from "@trenova/graphql/generated/graphql";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

const VENDOR_STATE_TONES: Record<CarrierIntelVendorState, StatusTone> = {
  Active: "success",
  PendingAdd: "low",
  PendingRemove: "low",
  Removed: "neutral",
  Failed: "critical",
  Unknown: "neutral",
};

const DESIRED_STATE_TONES: Record<CarrierIntelDesiredState, StatusTone> = {
  Enrolled: "success",
  NotEnrolled: "neutral",
};

const DESIRED_STATES: readonly CarrierIntelDesiredState[] = ["Enrolled", "NotEnrolled"];

const ENROLLMENT_REASONS: readonly CarrierIntelEnrollmentReason[] = [
  "PolicyAllActive",
  "PolicyRecentUse",
  "AssignedOrTendered",
  "Manual",
  "SelfMonitor",
  "CustomerBroker",
];

export function enrollmentInSync(row: CarrierMonitoringEnrollmentRow): boolean {
  if (row.desiredState === "Enrolled") {
    return row.vendorState === "Active";
  }
  return row.vendorState === "Removed" || row.vendorState === "Unknown";
}

function DotText({ tone, children }: { tone: StatusTone; children: string }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-2">
      <StatusDot tone={tone} />
      <span className="truncate">{children}</span>
    </span>
  );
}

function SubjectCell({ row, t }: { row: CarrierMonitoringEnrollmentRow; t: TranslateFn }) {
  const name = row.subjectName || t("USDOT {0}", row.dotNumber);
  return (
    <div className="flex min-w-0 flex-col">
      {row.carrierId ? (
        <Link
          to={carrierPanelPath(row.carrierId, "intelligence")}
          className="truncate font-medium hover:underline"
          onClick={(event) => event.stopPropagation()}
        >
          {name}
        </Link>
      ) : (
        <span className="truncate font-medium">{name}</span>
      )}
      <span className="text-muted-foreground text-xs tabular-nums">
        {row.docketNumber
          ? t("USDOT {0} · MC {1}", row.dotNumber, row.docketNumber)
          : t("USDOT {0}", row.dotNumber)}
      </span>
    </div>
  );
}

function LastErrorCell({ error }: { error: string | null }) {
  if (!error) {
    return <span className="text-muted-foreground">{INTEL_EMPTY_VALUE}</span>;
  }
  return (
    <Tooltip>
      <TooltipTrigger render={<span className="text-muted-foreground block truncate" />}>
        {error}
      </TooltipTrigger>
      <TooltipContent className="max-w-sm">{error}</TooltipContent>
    </Tooltip>
  );
}

export function getEnrollmentColumns(
  t: TranslateFn,
  labels: CarrierIntelLabels,
): ColumnDef<CarrierMonitoringEnrollmentRow>[] {
  return [
    {
      accessorKey: "subjectName",
      header: t("Carrier"),
      cell: ({ row }) => <SubjectCell row={row.original} t={t} />,
      size: 260,
      minSize: 180,
      maxSize: 360,
      meta: {
        label: t("Carrier"),
        apiField: "subjectName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "desiredState",
      header: t("Desired"),
      cell: ({ row }) => (
        <DotText tone={DESIRED_STATE_TONES[row.original.desiredState]}>
          {labels.desiredState[row.original.desiredState]}
        </DotText>
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        label: t("Desired"),
        apiField: "desiredState",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: DESIRED_STATES.map((state) => ({
          value: state,
          label: labels.desiredState[state],
        })),
        defaultFilterOperator: "eq",
        exportValue: (row: CarrierMonitoringEnrollmentRow) => labels.desiredState[row.desiredState],
      },
    },
    {
      accessorKey: "vendorState",
      header: t("Provider"),
      cell: ({ row }) => {
        const inSync = enrollmentInSync(row.original);
        return (
          <span
            className="inline-flex"
            title={
              inSync ? undefined : t("The provider has not caught up with what Trenova wants yet.")
            }
          >
            <DotText tone={VENDOR_STATE_TONES[row.original.vendorState]}>
              {labels.vendorState[row.original.vendorState]}
            </DotText>
          </span>
        );
      },
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        label: t("Provider"),
        apiField: "vendorState",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: CARRIER_INTEL_VENDOR_STATES.map((state) => ({
          value: state,
          label: labels.vendorState[state],
        })),
        defaultFilterOperator: "eq",
        exportValue: (row: CarrierMonitoringEnrollmentRow) => labels.vendorState[row.vendorState],
      },
    },
    {
      accessorKey: "reason",
      header: t("Reason"),
      cell: ({ row }) => (
        <span className="text-muted-foreground truncate">
          {labels.enrollmentReason[row.original.reason]}
        </span>
      ),
      size: 180,
      minSize: 130,
      maxSize: 240,
      meta: {
        label: t("Reason"),
        apiField: "reason",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ENROLLMENT_REASONS.map((reason) => ({
          value: reason,
          label: labels.enrollmentReason[reason],
        })),
        defaultFilterOperator: "eq",
        exportValue: (row: CarrierMonitoringEnrollmentRow) => labels.enrollmentReason[row.reason],
      },
    },
    {
      accessorKey: "lastSyncedAt",
      header: t("Last synced"),
      cell: ({ row }) => (
        <RelativeTime
          timestamp={row.original.lastSyncedAt}
          fallback={t("Never")}
          className="text-muted-foreground"
        />
      ),
      size: 130,
      minSize: 110,
      maxSize: 180,
      meta: {
        label: t("Last synced"),
        apiField: "lastSyncedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "failureCount",
      header: t("Failures"),
      cell: ({ row }) => (
        <span
          className={cn(
            "tabular-nums",
            row.original.failureCount > 0 ? "text-destructive" : "text-muted-foreground",
          )}
        >
          {row.original.failureCount.toLocaleString()}
        </span>
      ),
      size: 100,
      minSize: 80,
      maxSize: 140,
      meta: {
        label: t("Failures"),
        apiField: "failureCount",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "lastError",
      header: t("Last error"),
      cell: ({ row }) => <LastErrorCell error={row.original.lastError} />,
      size: 260,
      minSize: 140,
      maxSize: 420,
      meta: {
        label: t("Last error"),
        apiField: "lastError",
        filterable: false,
        sortable: false,
      },
    },
  ];
}
