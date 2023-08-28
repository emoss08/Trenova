/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import React, { useMemo, useRef } from "react";
import { MRT_ColumnDef } from "mantine-react-table";
import { Avatar, Badge, Box, Text, Tooltip } from "@mantine/core";
import { CreateUserDrawer } from "@/components/user-management/table/CreateUserDrawer";
import { ViewUserModal } from "./ViewUserModal";
import { userTableStore } from "@/stores/UserTableStore";
import { MontaTable } from "@/components/MontaTable";
import { User } from "@/types/apps/accounts";
import { formatDate, formatDateToHumanReadable } from "@/utils/date";
import { MontaTableActionMenu } from "@/components/ui/table/ActionsMenu";

export function UsersAdminTable() {
  const columns = useMemo<MRT_ColumnDef<User>[]>(
    () => [
      {
        id: "status",
        accessorFn: (originalRow) => (originalRow.isActive ? "true" : "false"),
        header: "Status",
        filterFn: "equals",
        Cell: ({ cell }) => (
          <Badge
            color={cell.getValue() === "true" ? "green" : "red"}
            variant="filled"
            radius="xs"
          >
            {cell.getValue() === "true" ? "Active" : "Inactive"}
          </Badge>
        ),
        mantineFilterSelectProps: {
          data: [
            { value: "", label: "All" },
            { value: "true", label: "Active" },
            { value: "false", label: "Inactive" },
          ] as any,
        },
        filterVariant: "select",
      },
      {
        accessorFn: (row) =>
          `${row.profile?.firstName} ${row.profile?.lastName}`,
        id: "name",
        header: "Name",
        size: 250,
        Cell: ({ renderedCellValue, row }) => (
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              gap: "16px",
            }}
          >
            {row.original.profile?.profilePicture ? (
              <Avatar
                src={row.original.profile?.profilePicture}
                alt="Test"
                radius="xl"
                size={30}
              />
            ) : (
              <Avatar color="blue" radius="xl" size={30}>
                {row.original.profile?.firstName.charAt(0)}
                {row.original.profile?.lastName.charAt(0)}
              </Avatar>
            )}
            <span>{renderedCellValue}</span>
          </Box>
        ),
      },
      {
        accessorKey: "email",
        header: "Email",
      },
      {
        id: "date_joined",
        header: "Date Joined",
        accessorFn: (row) => {
          if (row.dateJoined) {
            return formatDateToHumanReadable(row.dateJoined);
          }
          return null;
        },
        Cell: ({ row }) => {
          if (!row.original.dateJoined) {
            return <Text>Never</Text>;
          }

          return formatDate(row.original.dateJoined);
        },
      },
      {
        id: "last_login",
        header: "Last Login",
        accessorFn: (row) => {
          if (row.lastLogin) {
            return formatDateToHumanReadable(row.lastLogin);
          }
          return null;
        },
        Cell: ({ renderedCellValue, row }) => {
          if (!row.original.lastLogin) {
            return <Text>Never</Text>;
          }
          const tooltipDate = formatDate(row.original.lastLogin);
          const ref = useRef<HTMLDivElement>(null);

          return (
            <Tooltip withArrow position="left" label={tooltipDate}>
              <Text ref={ref}>{renderedCellValue}</Text>
            </Tooltip>
          );
        },
      },
      {
        id: "actions",
        header: "Actions",
        Cell: ({ row }) => (
          <MontaTableActionMenu store={userTableStore} data={row.original} />
        ),
      },
    ],
    [],
  );

  return (
    <MontaTable
      store={userTableStore}
      link="/users"
      columns={columns}
      TableCreateDrawer={CreateUserDrawer}
      displayDeleteModal
      TableViewModal={ViewUserModal}
      tableQueryKey="users-table-data"
      name="User"
      exportModelName="User"
    />
  );
}
