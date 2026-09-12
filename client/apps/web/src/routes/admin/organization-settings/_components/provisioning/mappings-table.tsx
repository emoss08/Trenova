import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { DataTable } from "@/components/data-table/data-table";
import {
  createSCIMGroupRoleMappingTableGraphQLConfig,
  type SCIMGroupRoleMappingRow,
} from "@/lib/graphql/scim-group-role-mapping-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useMemo } from "react";
import { scimGroupMappingPanelQueryKey } from "./constants";
import { SCIMGroupMappingPanel } from "./mapping-panel";

function getColumns(t: TranslateFn): ColumnDef<SCIMGroupRoleMappingRow>[] {
  return [
    {
      accessorKey: "externalGroupId",
      header: t("External Group ID"),
      meta: {
        label: t("External Group ID"),
        apiField: "externalGroupId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "displayName",
      header: t("Display Name"),
      meta: {
        label: t("Display Name"),
        apiField: "displayName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "role",
      header: t("Role"),
      cell: ({ row }) => {
        const { role } = row.original;

        if (!role) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<NonNullable<SCIMGroupRoleMappingRow["role"]>, SCIMGroupRoleMappingRow>
            entity={role}
            config={{
              basePath: "/roles",
              getId: (role) => role.id,
              getDisplayText: (role) => role.name,
              getHeaderText: "Role",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        apiField: "role.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
        label: t("Role"),
      },
    },
  ];
}

export default function SCIMGroupRoleMappingsTable({
  organizationId,
  directoryId,
}: {
  organizationId: string;
  directoryId: string;
}) {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);
  const queryKey = scimGroupMappingPanelQueryKey(organizationId, directoryId);
  const graphql = useMemo(
    () => createSCIMGroupRoleMappingTableGraphQLConfig(directoryId),
    [directoryId],
  );

  return (
    <DataTable<SCIMGroupRoleMappingRow>
      name="SCIM Group Role Mapping"
      queryKey={queryKey}
      graphql={graphql}
      columns={columns}
      includeHeader={false}
      enableCreateAction={false}
      TablePanel={(props) => (
        <SCIMGroupMappingPanel
          {...props}
          directoryId={directoryId}
          organizationId={organizationId}
        />
      )}
    />
  );
}
