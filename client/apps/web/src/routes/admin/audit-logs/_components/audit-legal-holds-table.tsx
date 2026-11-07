import { DataTable } from "@/components/data-table/data-table";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { usePermission } from "@/hooks/use-permission";
import {
  AUDIT_LEGAL_HOLD_LIST_KEY,
  auditLegalHoldTableGraphQLConfig,
  type AuditLegalHoldRow,
} from "@/lib/graphql/audit-log-admin";
import { LockUnlocked01Icon, PlusIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { resourceLabel } from "./audit-log-formatters";
import { PlaceLegalHoldDialog, ReleaseLegalHoldDialog } from "./legal-hold-dialogs";

function holdColumns(t: TranslateFn): ColumnDef<AuditLegalHoldRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      meta: {
        label: t("Name"),
        apiField: "name",
        sortable: true,
        filterable: true,
        filterType: "text",
      },
    },
    {
      id: "scope",
      header: t("Holds"),
      cell: ({ row }) => {
        const { resource, resourceId } = row.original;
        if (!resource) {
          return t("Every entry");
        }
        return resourceId
          ? t("{0} {1}", resourceLabel(resource), resourceId)
          : t("Every {0} entry", resourceLabel(resource));
      },
      meta: { label: t("Holds"), sortable: false, filterable: false },
    },
    {
      accessorKey: "reason",
      header: t("Reason"),
      meta: {
        label: t("Reason"),
        apiField: "reason",
        sortable: false,
        filterable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "placedAt",
      header: t("Placed"),
      cell: ({ row }) => (
        <div className="flex flex-col">
          <HoverCardTimestamp timestamp={row.original.placedAt} />
          <span className="text-muted-foreground text-xs">{row.original.placedBy?.name}</span>
        </div>
      ),
      meta: { label: t("Placed"), apiField: "placedAt", sortable: true, filterable: false },
    },
    {
      id: "status",
      header: t("Status"),
      cell: ({ row }) =>
        row.original.active ? (
          <Badge variant="warning">{t("Active")}</Badge>
        ) : (
          <div className="flex flex-col">
            <Badge variant="neutral">{t("Released")}</Badge>
            <span className="text-muted-foreground text-xs">{row.original.releaseReason}</span>
          </div>
        ),
      meta: { label: t("Status"), sortable: false, filterable: false },
    },
  ];
}

export default function AuditLegalHoldsTable() {
  const t = useT();
  const { allowed: canPlace } = usePermission(Resource.AuditLegalHold, Operation.Create);
  const { allowed: canRelease } = usePermission(Resource.AuditLegalHold, Operation.Update);
  const [placeOpen, setPlaceOpen] = useState(false);
  const [releasing, setReleasing] = useState<{ id: string; name: string } | null>(null);

  const contextMenuActions: RowAction<AuditLegalHoldRow>[] = [
    {
      id: "release",
      label: t("Release"),
      icon: LockUnlocked01Icon,
      onClick: (row) => setReleasing({ id: row.original.id, name: row.original.name }),
      hidden: (row) => !canRelease || !row.original.active,
    },
  ];

  return (
    <>
      <DataTable<AuditLegalHoldRow>
        name="Legal Hold"
        emptyTitle={t("No legal holds")}
        queryKey={AUDIT_LEGAL_HOLD_LIST_KEY}
        graphql={auditLegalHoldTableGraphQLConfig}
        resource={Resource.AuditLegalHold}
        columns={holdColumns(t)}
        contextMenuActions={contextMenuActions}
        enableExport={false}
        enableCreateAction={false}
        toolbar={
          canPlace
            ? {
                trailing: (
                  <Button type="button" onClick={() => setPlaceOpen(true)}>
                    <PlusIcon aria-hidden />
                    {t("New legal hold")}
                  </Button>
                ),
              }
            : undefined
        }
      />
      <PlaceLegalHoldDialog open={placeOpen} onOpenChange={setPlaceOpen} />
      <ReleaseLegalHoldDialog
        hold={releasing}
        onOpenChange={(open) => !open && setReleasing(null)}
      />
    </>
  );
}
