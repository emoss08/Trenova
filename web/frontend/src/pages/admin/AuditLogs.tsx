/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import AdminLayout from "@/components/admin-page/layout";
import { AuditLogView } from "@/components/audit-log/audit-log-table";
import { Input } from "@/components/common/fields/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/common/fields/select";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import {
  Credenza,
  CredenzaBody,
  CredenzaContent,
} from "@/components/ui/credenza";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatToUserTimezone } from "@/lib/date";
import { upperFirst } from "@/lib/utils";
import { getAuditLogs } from "@/services/OrganizationRequestService";
import { AuditLog, AuditLogAction, AuditLogStatus } from "@/types/organization";
import { useQuery } from "@tanstack/react-query";
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";

function mapActionToBadge(action: AuditLogAction) {
  switch (action) {
    case "CREATE":
      return <Badge variant="purple">Create</Badge>;
    case "UPDATE":
      return <Badge variant="info">Update</Badge>;
    case "DELETE":
      return <Badge variant="inactive">Delete</Badge>;
    default:
      return <Badge variant="default">Unknown</Badge>;
  }
}

function mapStatusToBadge(status: AuditLogStatus) {
  switch (status) {
    case "SUCCEEDED":
      return <Badge variant="active">Success</Badge>;
    case "FAILED":
      return <Badge variant="inactive">Failure</Badge>;
    case "ATTEMPTED":
      return <Badge variant="info">Attempted</Badge>;
    default:
      return <Badge variant="default">Unknown</Badge>;
  }
}

const columns: ColumnDef<AuditLog>[] = [
  {
    accessorKey: "status",
    header: "Log Status",
    cell: ({ row }) => mapStatusToBadge(row.original.status),
  },
  {
    accessorKey: "username",
    header: "Username",
    cell: ({ row }) => upperFirst(row.original.username),
  },
  {
    accessorKey: "tableName",
    header: "Table Name",
  },
  {
    accessorKey: "action",
    header: "Action",
    cell: ({ row }) => mapActionToBadge(row.original.action),
  },
  {
    accessorKey: "timestamp",
    header: "Timestamp",
    cell: ({ getValue }) => formatToUserTimezone(getValue() as string),
  },
];

function AuditLogDataDialog({
  auditLog,
  open,
  setOpen,
}: {
  auditLog: AuditLog;
  open: boolean;
  setOpen: (open: boolean) => void;
}) {
  return (
    <Credenza open={open} onOpenChange={setOpen}>
      <CredenzaContent className="max-w-[600px] bg-background">
        <CredenzaBody>
          <AuditLogView auditLog={auditLog} />
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}

type AuditLogTableFilters = {
  tableName: string;
  userId: string;
  entityId: string;
  action?: AuditLogAction;
  status?: AuditLogStatus;
};

function AuditLogTable() {
  const [filters, setFilters] = useState<AuditLogTableFilters>({
    tableName: "",
    userId: "",
    entityId: "",
    action: undefined,
    status: undefined,
  });

  const { data, isLoading, isError } = useQuery({
    queryKey: ["auditLogs", filters],
    queryFn: () =>
      getAuditLogs(
        filters.tableName,
        filters.userId,
        filters.entityId,
        filters.action,
        filters.status,
      ),
  });
  const [viewDialogOpen, setViewDialogOpen] = useState(false);
  const [currentRecord, setCurrentRecord] = useState<AuditLog | null>(null);

  const table = useReactTable({
    data: data?.results || [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const handleFilterChange = (key: keyof typeof filters, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  if (isLoading) return <div>Loading...</div>;
  if (isError) return <div>Error loading audit logs</div>;
  return (
    <>
      <Card className="rounded-md border border-border bg-card">
        <div className="m-4 space-y-4">
          <div className="grid grid-cols-4 gap-4">
            <div>
              <Input
                id="tableName"
                value={filters.tableName}
                onChange={(e) =>
                  handleFilterChange("tableName", e.target.value)
                }
                placeholder="Filter by table name"
              />
            </div>
            <div>
              <Input
                id="userId"
                value={filters.userId}
                onChange={(e) => handleFilterChange("userId", e.target.value)}
                placeholder="Filter by user ID"
              />
            </div>
            <div>
              <Select
                value={filters.action}
                onValueChange={(value) => handleFilterChange("action", value)}
              >
                <SelectTrigger id="action">
                  <SelectValue placeholder="Select an action" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="NONE">All Actions</SelectItem>
                  <SelectItem value="CREATE">Create</SelectItem>
                  <SelectItem value="UPDATE">Update</SelectItem>
                  <SelectItem value="DELETE">Delete</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Select
                value={filters.status}
                onValueChange={(value) => handleFilterChange("status", value)}
              >
                <SelectTrigger id="status">
                  <SelectValue placeholder="Select an status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="NONE">All Actions</SelectItem>
                  <SelectItem value="ATTEMPTED">Attempted</SelectItem>
                  <SelectItem value="SUCCEEDED">Succeeded</SelectItem>
                  <SelectItem value="FAILED">Failed</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext(),
                        )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.map((row) => (
              <TableRow
                key={row.id}
                className="cursor-pointer select-none"
                onDoubleClick={() => {
                  setCurrentRecord(row.original);
                  setViewDialogOpen(true);
                }}
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
      {viewDialogOpen && currentRecord && (
        <AuditLogDataDialog
          auditLog={currentRecord}
          open={viewDialogOpen}
          setOpen={setViewDialogOpen}
        />
      )}
    </>
  );
}

export default function AuditLogs() {
  return (
    <AdminLayout>
      <AuditLogTable />
    </AdminLayout>
  );
}
