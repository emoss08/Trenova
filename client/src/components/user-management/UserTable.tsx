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

import React, { useMemo, useState } from "react";
import {
  MantineReactTable,
  MRT_ColumnDef,
  MRT_GlobalFilterTextInput,
  MRT_PaginationState,
} from "mantine-react-table";
import { useQuery } from "react-query";
import axios from "@/lib/axiosConfig";
import { API_URL } from "@/utils/utils";
import { User } from "@/types/user";
import {
  ActionIcon,
  Avatar,
  Badge,
  Box,
  Button,
  Flex,
  Menu,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import "@fortawesome/fontawesome-svg-core/styles.css";
import { config } from "@fortawesome/fontawesome-svg-core";
import { DatePickerInput } from "@mantine/dates";
import { formatDate, formatDateToHumanReadable } from "@/utils/date";
import {
  faBars,
  faFilter,
  faFileExport,
  faUser,
  faUserGear,
  faUserMinus,
  faUserPlus,
} from "@fortawesome/pro-duotone-svg-icons";
import { montaTableIcons } from "@/components/ui/table/Icons";
import { CreateUserDrawer } from "./CreateUserDrawer";
import { useDisclosure } from "@mantine/hooks";
import { modals } from "@mantine/modals";
import { ExportUsersModal } from "@/components/user-management/ExportUsersModal";

config.autoAddCss = false;

type ApiResponse = {
  results: User[];
  count: number;
};

const UsersAdminTable = () => {
  const [pagination, setPagination] = useState<MRT_PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  });
  const [globalFilter, setGlobalFilter] = useState<string>("");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [isExportModalOpen, setExportModalOpen] = useState(false);

  const openExportUsersModal = () => {
    setExportModalOpen(true);
  };

  const closeExportUsersModal = () => {
    setExportModalOpen(false);
  };
  const closeDrawer = () => setDrawerOpen(false);
  const openDrawer = () => setDrawerOpen(true);

  const { data, isError, isFetching, isLoading } = useQuery<ApiResponse>(
    [
      "user-table-data",
      pagination.pageIndex,
      pagination.pageSize,
      globalFilter,
    ],
    async () => {
      const offset = pagination.pageIndex * pagination.pageSize;
      const url = `${API_URL}/users/?limit=${
        pagination.pageSize
      }&offset=${offset}${globalFilter ? `&search=${globalFilter}` : ""}`;
      const response = await axios.get(url);
      return response.data;
    },
    {
      refetchOnWindowFocus: false,
      keepPreviousData: true,
    }
  );

  const handlePaginationChange = (state: any) => {
    setPagination(state);
  };

  const columns = useMemo(
    () =>
      [
        {
          id: "status",
          accessorFn: (originalRow) =>
            originalRow.is_active ? "true" : "false",
          header: "Status",
          filterFn: "equals",
          Cell: ({ cell }) => (
            <Badge
              color={cell.getValue() === "true" ? "green" : "red"}
              variant="dot"
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
            `${row.profile?.first_name} ${row.profile?.last_name}`,
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
              {row.original.profile?.profile_picture ? (
                <Avatar
                  src={row.original.profile?.profile_picture}
                  alt={"Test"}
                  radius="xl"
                  size={30}
                />
              ) : (
                <Avatar color="blue" radius="xl" size={30}>
                  {row.original.profile?.first_name.charAt(0)}
                  {row.original.profile?.last_name.charAt(0)}
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
          accessorFn: (row) => new Date(row.date_joined),
          id: "date_joined",
          header: "Date Joined",
          filterFn: "lessThanOrEqualTo",
          sortingFn: "datetime",
          Cell: ({ cell }) => cell.getValue<Date>()?.toLocaleString(),
          Header: ({ column }) => <em>{column.columnDef.header}</em>,
          Filter: ({ column }) => (
            <DatePickerInput
              placeholder="Filter by Date Joined"
              onChange={(newValue: Date) => {
                column.setFilterValue(newValue);
              }}
              variant="filled"
              mt={9}
              value={column.getFilterValue() as Date}
              modalProps={{ withinPortal: true }}
            />
          ),
        },
        {
          id: "last_login",
          header: "Last Login",
          accessorFn: (row) => {
            if (row.last_login) {
              return formatDateToHumanReadable(row.last_login);
            } else {
              return null;
            }
          },
          Cell: ({ renderedCellValue, row }) => {
            if (!row.original.last_login) {
              return <Text>Never</Text>;
            }
            const tooltipDate = formatDate(row.original.last_login);

            return (
              <Tooltip withArrow position="left" label={tooltipDate}>
                <Text>{renderedCellValue}</Text>
              </Tooltip>
            );
          },
        },
        {
          id: "actions",
          header: "Actions",
          Cell: ({ row }) => (
            <Menu width={200} shadow="md">
              <Menu.Target>
                <ActionIcon variant="transparent">
                  <FontAwesomeIcon icon={faBars} />
                </ActionIcon>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>User Actions</Menu.Label>
                <Menu.Item icon={<FontAwesomeIcon icon={faUser} />}>
                  View User Profile
                </Menu.Item>
                <Menu.Item icon={<FontAwesomeIcon icon={faUserGear} />}>
                  Edit User Profile
                </Menu.Item>
                <Menu.Item
                  color="red"
                  icon={<FontAwesomeIcon icon={faUserMinus} />}
                >
                  Delete User Profile
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          ),
        },
      ] as MRT_ColumnDef<User>[],
    []
  );

  return (
    <MantineReactTable
      columns={columns}
      data={data?.results ?? []}
      manualPagination
      onPaginationChange={handlePaginationChange}
      rowCount={data?.count ?? 0}
      getRowId={(row) => row.id}
      enableRowSelection
      icons={montaTableIcons}
      state={{
        isLoading,
        pagination,
        showAlertBanner: isError,
        showSkeletons: isFetching,
      }}
      initialState={{
        showGlobalFilter: true,
      }}
      positionGlobalFilter="left"
      mantineSearchTextInputProps={{
        placeholder: `Search ${data?.count} users...`,
        sx: { minWidth: "300px" },
        variant: "filled",
      }}
      enableGlobalFilterModes={false}
      onGlobalFilterChange={setGlobalFilter}
      mantineFilterTextInputProps={{
        sx: { borderBottom: "unset", marginTop: "8px" },
        variant: "filled",
      }}
      mantineFilterSelectProps={{
        sx: { borderBottom: "unset", marginTop: "8px" },
        variant: "filled",
      }}
      renderTopToolbar={({ table }) => (
        <>
          <Flex
            sx={() => ({
              borderRadius: "4px",
              flexDirection: "row",
              justifyContent: "space-between",
              padding: "24px 16px",
              "@media max-width: 768px": {
                flexDirection: "column",
              },
            })}
          >
            <Box
              sx={{
                display: "flex",
                gap: "16px",
                flexWrap: "wrap",
                flex: 1,
                justifyContent: "flex-start",
              }}
            >
              <MRT_GlobalFilterTextInput table={table} />
            </Box>

            <Flex
              gap="xs"
              align="center"
              sx={{
                flex: 1,
                justifyContent: "flex-end",
              }}
            >
              <Button
                color="blue"
                leftIcon={<FontAwesomeIcon icon={faFilter} />}
              >
                Filter
              </Button>
              <Button
                color="blue"
                leftIcon={<FontAwesomeIcon icon={faFileExport} />}
                onClick={openExportUsersModal}
              >
                Export
              </Button>
              <Button
                color="blue"
                onClick={openDrawer}
                leftIcon={<FontAwesomeIcon icon={faUserPlus} />}
              >
                Create New User
              </Button>
              <ExportUsersModal
                onClose={closeExportUsersModal}
                opened={isExportModalOpen}
              />
              <CreateUserDrawer onClose={closeDrawer} opened={drawerOpen} />
            </Flex>
          </Flex>
        </>
      )}
    />
  );
};

export default UsersAdminTable;
