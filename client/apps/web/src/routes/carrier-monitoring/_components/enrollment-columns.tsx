import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { carrierPanelPath } from "@/lib/carrier-links";
import type { CarrierMonitoringEnrollmentRow } from "@/lib/graphql/carrier-monitoring-table";
import type {
  CarrierIntelDesiredState,
  CarrierIntelVendorState,
} from "@trenova/graphql/generated/graphql";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

const VENDOR_STATE_VARIANTS: Record<CarrierIntelVendorState, BadgeVariant> = {
  Active: "active",
  PendingAdd: "info",
  PendingRemove: "info",
  Removed: "secondary",
  Failed: "inactive",
  Unknown: "outline",
};

const DESIRED_STATE_VARIANTS: Record<CarrierIntelDesiredState, BadgeVariant> = {
  Enrolled: "teal",
  NotEnrolled: "secondary",
};

export function enrollmentInSync(row: CarrierMonitoringEnrollmentRow): boolean {
  if (row.desiredState === "Enrolled") {
    return row.vendorState === "Active";
  }
  return row.vendorState === "Removed" || row.vendorState === "Unknown";
}

function EnrollmentSubjectCell({
  row,
  t,
}: {
  row: CarrierMonitoringEnrollmentRow;
  t: TranslateFn;
}) {
  const name = row.subjectName || row.dotNumber;
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
      <span className="text-muted-foreground text-2xs">
        {row.docketNumber
          ? t("USDOT {0} · MC {1}", row.dotNumber, row.docketNumber)
          : t("USDOT {0}", row.dotNumber)}
      </span>
    </div>
  );
}

function EnrollmentStateCell({
  row,
  t,
  labels,
}: {
  row: CarrierMonitoringEnrollmentRow;
  t: TranslateFn;
  labels: CarrierIntelLabels;
}) {
  const inSync = enrollmentInSync(row);
  return (
    <div className="flex flex-wrap items-center gap-1">
      <Badge variant={DESIRED_STATE_VARIANTS[row.desiredState]} className="max-h-5">
        {labels.desiredState[row.desiredState]}
      </Badge>
      <Badge
        variant={VENDOR_STATE_VARIANTS[row.vendorState]}
        className="max-h-5"
        title={
          inSync
            ? t("The provider matches what Trenova wants.")
            : t("The provider has not caught up with what Trenova wants yet.")
        }
      >
        {labels.vendorState[row.vendorState]}
      </Badge>
    </div>
  );
}

function EnrollmentFailureCell({
  row,
  t,
}: {
  row: CarrierMonitoringEnrollmentRow;
  t: TranslateFn;
}) {
  if (row.failureCount === 0 && !row.lastError) {
    return <span className="text-muted-foreground">-</span>;
  }
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="text-destructive text-xs font-medium tabular-nums">
        {t("{0, plural, one {# failure} other {# failures}}", row.failureCount)}
      </span>
      {row.lastError ? (
        <span className="text-muted-foreground truncate text-2xs" title={row.lastError}>
          {row.lastError}
        </span>
      ) : null}
    </div>
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
      cell: ({ row }) => <EnrollmentSubjectCell row={row.original} t={t} />,
      size: 240,
      minSize: 160,
      maxSize: 340,
      meta: {
        apiField: "subjectName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "vendorState",
      header: t("Desired / Provider"),
      cell: ({ row }) => <EnrollmentStateCell row={row.original} t={t} labels={labels} />,
      size: 220,
      minSize: 180,
      maxSize: 280,
      meta: {
        apiField: "vendorState",
        filterable: false,
        sortable: true,
        exportValue: (row: CarrierMonitoringEnrollmentRow) =>
          `${labels.desiredState[row.desiredState]} / ${labels.vendorState[row.vendorState]}`,
      },
    },
    {
      accessorKey: "reason",
      header: t("Reason"),
      cell: ({ row }) => <span>{labels.enrollmentReason[row.original.reason]}</span>,
      size: 170,
      minSize: 130,
      maxSize: 220,
      meta: {
        apiField: "reason",
        filterable: false,
        sortable: true,
        exportValue: (row: CarrierMonitoringEnrollmentRow) => labels.enrollmentReason[row.reason],
      },
    },
    {
      accessorKey: "mode",
      header: t("Mode"),
      cell: ({ row }) => (
        <span className="text-muted-foreground">{labels.enrollmentMode[row.original.mode]}</span>
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        apiField: "mode",
        filterable: false,
        sortable: true,
        exportValue: (row: CarrierMonitoringEnrollmentRow) => labels.enrollmentMode[row.mode],
      },
    },
    {
      accessorKey: "failureCount",
      header: t("Failures"),
      cell: ({ row }) => <EnrollmentFailureCell row={row.original} t={t} />,
      size: 220,
      minSize: 120,
      maxSize: 360,
      meta: {
        apiField: "failureCount",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "lastSyncedAt",
      header: t("Last synced"),
      cell: ({ row }) =>
        row.original.lastSyncedAt ? (
          <HoverCardTimestamp timestamp={row.original.lastSyncedAt} />
        ) : (
          <span className="text-muted-foreground">{t("Never")}</span>
        ),
      size: 190,
      minSize: 160,
      maxSize: 240,
      meta: {
        apiField: "lastSyncedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "enrolledAt",
      header: t("Enrolled"),
      cell: ({ row }) =>
        row.original.enrolledAt ? (
          <HoverCardTimestamp timestamp={row.original.enrolledAt} />
        ) : (
          <span className="text-muted-foreground">-</span>
        ),
      size: 190,
      minSize: 160,
      maxSize: 240,
      meta: {
        apiField: "enrolledAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
