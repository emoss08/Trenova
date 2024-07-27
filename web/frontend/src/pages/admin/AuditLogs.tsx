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
import { Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/common/fields/select";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import {
  Credenza,
  CredenzaBody,
  CredenzaContent,
  CredenzaDescription,
  CredenzaHeader,
  CredenzaTitle,
} from "@/components/ui/credenza";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatToUserTimezone } from "@/lib/date";
import { getAuditLogs } from "@/services/OrganizationRequestService";
import { AuditLog, AuditLogAction, AuditLogStatus } from "@/types/organization";
import { ScrollArea } from "@radix-ui/react-scroll-area";
import { useQuery } from "@tanstack/react-query";
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";

function ActionBadge({ action }: { action: AuditLogAction }) {
  return (
    <Badge variant={action === "CREATE" ? "purple" : "info"}>{action}</Badge>
  );
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
    accessorKey: "user",
    header: "User",
    cell: ({ row }) => {
      const user = row.original.user;
      return (
        <div className="flex items-center">
          <Avatar className="size-9">
            <AvatarImage src={user.profilePicUrl || ""} alt={user.name} />
            <AvatarFallback>{user.name[0]}</AvatarFallback>
          </Avatar>
          <div className="ml-4">
            <div className="font-medium">{user.name}</div>
            <div className="text-sm text-muted-foreground">{user.email}</div>
          </div>
        </div>
      );
    },
  },
  {
    accessorKey: "tableName",
    header: "Table Name",
  },
  {
    accessorKey: "action",
    header: "Action",
    cell: ({ row }) => <ActionBadge action={row.original.action} />,
  },
  {
    accessorKey: "timestamp",
    header: "Timestamp",
    cell: ({ getValue }) => formatToUserTimezone(getValue() as string),
  },
];

function AuditLogDetailsTable({ data }: { data: { [key: string]: any } }) {
  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-1/5">Field</TableHead>
            <TableHead>Value</TableHead>
          </TableRow>
        </TableHeader>
      </Table>
      <Table>
        <ScrollArea className="h-[500px]">
          <TableBody>
            {Object.entries(data).map(([key, value]) => (
              <TableRow key={key}>
                <TableCell className="w-1/5 font-medium">{key}</TableCell>
                <TableCell className="break-all">
                  {typeof value === "object"
                    ? JSON.stringify(value, null, 2)
                    : String(value)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </ScrollArea>
      </Table>
    </>
  );
}

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
      <CredenzaContent className="max-w-[800px] bg-background">
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>Viewing Audit Log Entry</span>
            <span className="ml-3">
              <ActionBadge action={auditLog.action} />
            </span>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Action performed by <b>{auditLog.user.username}</b> on{" "}
          {formatToUserTimezone(auditLog.timestamp)}
        </CredenzaDescription>
        <CredenzaBody>
          <AuditLogDetailsTable data={auditLog.data} />
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
      <div className="mb-4 space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <Label htmlFor="tableName">Table Name</Label>
            <Input
              id="tableName"
              value={filters.tableName}
              onChange={(e) => handleFilterChange("tableName", e.target.value)}
              placeholder="Filter by table name"
            />
          </div>
          <div>
            <Label htmlFor="userId">User ID</Label>
            <Input
              id="userId"
              value={filters.userId}
              onChange={(e) => handleFilterChange("userId", e.target.value)}
              placeholder="Filter by user ID"
            />
          </div>
          <div>
            <Label htmlFor="entityId">Entity ID</Label>
            <Input
              id="entityId"
              value={filters.entityId}
              onChange={(e) => handleFilterChange("entityId", e.target.value)}
              placeholder="Filter by entity ID"
            />
          </div>
          <div>
            <Label htmlFor="action">Action</Label>
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
            <Label htmlFor="status">Status</Label>
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
        <TableCaption>A list of audit logs for the organization.</TableCaption>
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
              onClick={() => {
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
