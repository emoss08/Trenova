import { DataTable } from "@/components/data-table/data-table";
import { Resource } from "@/types/permission";
import type { Role } from "@/types/role";
import type { Row } from "@tanstack/react-table";
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
    (row: Row<Role>) => void navigate(`/admin/roles/${row.original.id}/edit`),
    [navigate],
  );

  return (
    <DataTable<Role>
      exportModelName="Role"
      name="Role"
      link="/roles/"
      queryKey="role-list"
      resource={Resource.Role}
      columns={columns}
      onAddRecord={handleAddRecord}
      onRowClick={handleRowClick}
    />
  );
}
