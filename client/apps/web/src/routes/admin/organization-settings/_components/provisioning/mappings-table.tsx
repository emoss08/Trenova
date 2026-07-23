import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { DataTable } from "@/components/data-table/data-table";
import { createSCIMGroupRoleMappingTableGraphQLConfig } from "@/lib/graphql/scim-group-role-mapping-table";
import type { SCIMGroupRoleMapping } from "@trenova/shared/types/iam";
import type { Role } from "@trenova/shared/types/role";
import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { scimGroupMappingPanelQueryKey } from "./constants";
import { SCIMGroupMappingPanel } from "./mapping-panel";

function getColumns(): ColumnDef<SCIMGroupRoleMapping>[] {
  return [
    {
      accessorKey: "externalGroupId",
      header: "External Group ID",
      meta: {
        label: "External Group ID",
        apiField: "externalGroupId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "displayName",
      header: "Display Name",
      meta: {
        label: "Display Name",
        apiField: "displayName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) => {
        const { role } = row.original;

        if (!role) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<Role, SCIMGroupRoleMapping>
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
        label: "Role",
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
  const columns = useMemo(() => getColumns(), []);
  const queryKey = scimGroupMappingPanelQueryKey(organizationId, directoryId);
  const graphql = useMemo(
    () => createSCIMGroupRoleMappingTableGraphQLConfig(directoryId),
    [directoryId],
  );

  return (
    <DataTable<SCIMGroupRoleMapping>
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
