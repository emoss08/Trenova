import { DataTable } from "@/components/data-table/data-table";
import { roleTableGraphQLConfig, type RoleRow } from "@/lib/graphql/role-table";
import { Resource } from "@trenova/shared/types/permission";
import type { Row } from "@trenova/shared/types/data-table";
import { useCallback, useMemo } from "react";
import { useNavigate } from "react-router";
import { getColumns } from "./role-columns";

export default function RoleTable() {
  const navigate = useNavigate();
  const columns = useMemo(() => getColumns(), []);

  const handleAddRecord = () => {
    void navigate("/admin/roles/new");
  };

  const handleRowClick = useCallback(
    (row: Row<RoleRow>) => void navigate(`/admin/roles/${row.original.id}/edit`),
    [navigate],
  );

  return (
    <DataTable<RoleRow>
      name="Role"
      queryKey="role-list"
      graphql={roleTableGraphQLConfig}
      resource={Resource.Role}
      columns={columns}
      onAddRecord={handleAddRecord}
      onRowClick={handleRowClick}
    />
  );
}
