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

import React from "react";
import {
  MantineReactTable,
  MRT_ColumnDef,
  MRT_TableInstance,
  MRT_TableOptions,
} from "mantine-react-table";
import { useQuery } from "react-query";
import { useMantineTheme } from "@mantine/core";
import { useLocation, useSearchParams } from "react-router-dom";
import axios from "@/helpers/AxiosConfig";
import { montaTableIcons } from "@/components/common/table/Icons";
import { ApiResponse } from "@/types/server";
import { DeleteRecordModal } from "@/components/common/table/DeleteRecordModal";
import { API_URL } from "@/helpers/constants";
import { TableTopToolbar } from "@/components/common/table/TableTopToolbar";
import { TableExportModal } from "./TableExportModal";

MontaTable.defaultProps = {
  displayDeleteModal: true,
  TableCreateDrawer: undefined,
  TableDeleteModal: undefined,
  TableEditModal: undefined,
  TableViewModal: undefined,
  deleteKey: "id",
};

type Props<T extends Record<string, any>> = MRT_TableInstance<T> &
  MRT_TableOptions<T>;

type MontaTableProps<T extends Record<string, any>> = {
  store: any;
  link: string;
  displayDeleteModal?: boolean;
  TableCreateDrawer?: React.ComponentType;
  TableDeleteModal?: React.ComponentType;
  TableEditModal?: React.ComponentType;
  TableViewModal?: React.ComponentType;
  exportModelName: string;
  name: string;
  tableQueryKey: string;
  columns: MRT_ColumnDef<T>[];
  deleteKey?: string;
} & Pick<Props<T>, "mantineBottomToolbarProps" | "mantineTableBodyRowProps">;

export function MontaTable<T extends Record<string, any>>({
  store,
  link,
  TableCreateDrawer,
  TableEditModal,
  TableDeleteModal,
  TableViewModal,
  tableQueryKey,
  exportModelName,
  displayDeleteModal,
  name,
  columns,
  deleteKey,
}: MontaTableProps<T>) {
  const theme = useMantineTheme();
  const [pagination] = store.use("pagination");
  const [rowSelection, setRowSelection] = store.use("rowSelection");
  const [searchParams, setSearchParams] = useSearchParams();
  const location = useLocation();

  const searchQuery = searchParams.get("search") ?? "";

  React.useEffect(() => {
    setSearchParams(new URLSearchParams(location.search));
  }, [location.search]);

  const { data, isError, isFetching, isLoading } = useQuery<ApiResponse<T>>(
    [tableQueryKey, pagination.pageIndex, pagination.pageSize, searchQuery],
    async () => {
      const offset = pagination.pageIndex * pagination.pageSize;
      const fullUrl = `${API_URL}${link}?limit=${
        pagination.pageSize
      }&offset=${offset}${searchQuery ? `&search=${searchQuery}` : ""}`;
      const response = await axios.get(fullUrl);
      return response.data;
    },
    {
      refetchOnWindowFocus: false,
      keepPreviousData: true,
      staleTime: 1000 * 60 * 5, // 5 minutes
    },
  );

  return (
    <>
      <MantineReactTable
        columns={columns}
        data={data?.results ?? []}
        manualPagination
        onPaginationChange={(newPagination) => {
          store.set("pagination", newPagination);
        }}
        rowCount={data?.count ?? 0}
        getRowId={(row) => row.id}
        enableRowSelection
        icons={montaTableIcons}
        onRowSelectionChange={(row) => setRowSelection(row)}
        mantinePaperProps={{
          sx: { overflow: "visible" },
          shadow: "none",
          withBorder: false,
        }}
        mantineTableHeadRowProps={{
          sx: { overflow: "visible", boxShadow: "none" },
        }}
        mantineTableContainerProps={{
          sx: { overflow: "visible" },
        }}
        state={{
          isLoading,
          pagination,
          showAlertBanner: isError,
          showSkeletons: isFetching,
          rowSelection,
        }}
        initialState={{
          showGlobalFilter: true,
          density: "xs",
        }}
        positionGlobalFilter="left"
        mantineSearchTextInputProps={{
          placeholder: data?.count
            ? `Search ${data.count} records...`
            : "Search...",
          sx: { minWidth: "300px" },
          variant: "filled",
        }}
        enableGlobalFilterModes={false}
        onGlobalFilterChange={(filter: string) => {
          if (filter) {
            setSearchParams({ search: filter });
          } else {
            const newSearchParams = new URLSearchParams(
              searchParams.toString(),
            );
            newSearchParams.delete("search");
            setSearchParams(newSearchParams);
          }
        }}
        mantineFilterTextInputProps={{
          sx: { borderBottom: "unset", marginTop: "8px" },
          variant: "filled",
        }}
        mantineFilterSelectProps={{
          sx: { borderBottom: "unset", marginTop: "8px" },
          variant: "filled",
        }}
        mantineTableBodyCellProps={() => ({
          sx: {
            backgroundColor:
              theme.colorScheme === "dark" ? theme.colors.dark[7] : "white",
          },
        })}
        renderTopToolbar={({ table }) => (
          <TableTopToolbar table={table} store={store} name={name} />
        )}
      />
      <TableExportModal store={store} name={name} modelName={exportModelName} />
      {TableCreateDrawer && <TableCreateDrawer />}
      {displayDeleteModal && !TableDeleteModal && (
        <DeleteRecordModal
          link={link}
          queryKey={tableQueryKey}
          store={store}
          deleteKey={deleteKey}
        />
      )}
      {TableEditModal && <TableEditModal />}
      {TableDeleteModal && <TableDeleteModal />}
      {TableViewModal && <TableViewModal />}
    </>
  );
}
