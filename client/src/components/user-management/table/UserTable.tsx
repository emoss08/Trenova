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

import React, { useMemo } from "react";
import { MantineReactTable } from "mantine-react-table";
import { useQuery } from "react-query";
import { API_URL } from "@/utils/utils";
import "@fortawesome/fontawesome-svg-core/styles.css";
import { config } from "@fortawesome/fontawesome-svg-core";
import { ExportUserModal } from "@/components/user-management/table/ExportUserModal";
import { UserApiResponse } from "@/types/apps/accounts";
import { CreateUserDrawer } from "@/components/user-management/table/CreateUserDrawer";
import { UserTableColumns } from "@/components/user-management/table/UserTableColumns";
import axios from "@/lib/AxiosConfig";
import { montaTableIcons } from "@/components/ui/table/Icons";
import { UserTableTopToolbar } from "./UserTableTopToolbar";
import { ViewUserModal } from "./ViewUserModal";
import { DeleteUserModal } from "@/components/user-management/table/DeleteUserModal";
import { userTableStore } from "@/stores/UserTableStore";

config.autoAddCss = false;

export const UsersAdminTable = () => {
  const [pagination] = userTableStore.use("pagination");
  const [globalFilter, setGlobalFilter] = userTableStore.use("globalFilter");

  // Function to handle pagination
  const { data, isError, isFetching, isLoading } = useQuery<UserApiResponse>(
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
      staleTime: 1000 * 60 * 5, // 5 minutes
    }
  );

  // Function to handle column filters
  const columns = useMemo(() => UserTableColumns(), []);

  return (
    <>
      <MantineReactTable
        columns={columns}
        data={data?.results ?? []}
        manualPagination
        onPaginationChange={(newPagination) => {
          userTableStore.set("pagination", newPagination);
        }}
        rowCount={data?.count ?? 0}
        getRowId={(row) => row.id}
        enableRowSelection
        icons={montaTableIcons}
        state={{
          isLoading,
          pagination: pagination,
          showAlertBanner: isError,
          showSkeletons: isFetching,
        }}
        initialState={{
          showGlobalFilter: true,
          density: "xs",
        }}
        positionGlobalFilter="left"
        mantineSearchTextInputProps={{
          placeholder: `Search ${data?.count} users...`,
          sx: { minWidth: "300px" },
          variant: "filled",
        }}
        enableGlobalFilterModes={false}
        onGlobalFilterChange={(filter: string) => {
          setGlobalFilter(filter);
        }}
        mantineFilterTextInputProps={{
          sx: { borderBottom: "unset", marginTop: "8px" },
          variant: "filled",
        }}
        mantineFilterSelectProps={{
          sx: { borderBottom: "unset", marginTop: "8px" },
          variant: "filled",
        }}
        renderTopToolbar={({ table }) => <UserTableTopToolbar table={table} />}
      />
      <ExportUserModal />
      <CreateUserDrawer />
      <DeleteUserModal />
      <ViewUserModal />
    </>
  );
};
