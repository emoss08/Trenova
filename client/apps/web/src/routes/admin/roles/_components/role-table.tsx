import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { roleTableGraphQLConfig, type RoleRow } from "@/lib/graphql/role-table";
import { Resource } from "@trenova/shared/types/permission";
import type { Row } from "@trenova/shared/types/data-table";
import { useCallback, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";
import { getColumns } from "./role-columns";

export default function RoleTable() {
  const t = useT();

  const navigate = useNavigate();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleAddRecord = () => {
    void navigate({ to: "/admin/roles/new" });
  };

  const handleRowClick = useCallback(
    (row: Row<RoleRow>) => void navigate({ to: `/admin/roles/${row.original.id}/edit` }),
    [navigate],
  );

  return (
    <DataTable<RoleRow>
      name="Role"
      emptyTitle={t("No roles yet")}
      queryKey="role-list"
      graphql={roleTableGraphQLConfig}
      resource={Resource.Role}
      columns={columns}
      onAddRecord={handleAddRecord}
      onRowClick={handleRowClick}
    />
  );
}
