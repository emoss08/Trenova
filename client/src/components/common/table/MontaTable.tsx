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

import { DeleteRecordModal } from "@/components/common/table/DeleteRecordModal";
import { montaTableIcons } from "@/components/common/table/Icons";
import { TableTopToolbar } from "@/components/common/table/TableTopToolbar";
import { API_URL } from "@/lib/constants";
import { StoreType } from "@/lib/useGlobalStore";
import { QueryKeys } from "@/types";
import { ApiResponse } from "@/types/server";
import { TableStoreProps } from "@/types/tables";
import { useMantineTheme } from "@mantine/core";
import {
  MantineReactTable,
  MRT_ColumnDef,
  MRT_PaginationState,
  MRT_RowSelectionState,
  MRT_TableInstance,
  MRT_TableOptions,
  useMantineReactTable,
} from "mantine-react-table";
import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { TableExportModal } from "./TableExportModal";
import axios from "@/lib/axiosConfig";

MontaTable.defaultProps = {
  displayDeleteModal: true,
  TableCreateDrawer: undefined,
  TableDeleteModal: undefined,
  TableDrawer: undefined,
  deleteKey: "id",
};

type Props<T extends Record<string, any>> = MRT_TableInstance<T> &
  MRT_TableOptions<T>;

type MontaTableProps<T extends Record<string, any>, K> = {
  store: StoreType<K>;
  link: string;
  displayDeleteModal?: boolean;
  TableCreateDrawer?: React.ComponentType;
  TableDeleteModal?: React.ComponentType;
  TableDrawer?: React.ComponentType;
  exportModelName: string;
  name: string;
  tableQueryKey: QueryKeys;
  columns: MRT_ColumnDef<T>[];
  deleteKey?: string;
} & Pick<Props<T>, "mantineBottomToolbarProps" | "mantineTableBodyRowProps">;

export function MontaTable<
  T extends Record<string, any>,
  K extends TableStoreProps<T>,
>({
  store,
  link,
  TableCreateDrawer,
  TableDrawer,
  TableDeleteModal,
  tableQueryKey,
  exportModelName,
  displayDeleteModal,
  name,
  columns,
  deleteKey,
}: MontaTableProps<T, K>) {
  const theme = useMantineTheme();
  const [globalFilter, setGlobalFilter] = useState("");
  const [pagination, setPagination] = useState<MRT_PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  });
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({});

  const useGetTableData = () => {
    const fetchURL = new URL(`${API_URL}${link}/`);

    fetchURL.searchParams.set("limit", pagination.pageSize.toString());
    fetchURL.searchParams.set(
      "offset",
      (pagination.pageIndex * pagination.pageSize).toString(),
    );

    if (globalFilter) {
      fetchURL.searchParams.set("search", globalFilter);
    }

    return useQuery<ApiResponse<T>>({
      queryKey: [tableQueryKey, fetchURL.href],
      queryFn: async () => {
        const response = await axios.get(fetchURL.href);
        return response.data;
      },
      keepPreviousData: true,
      staleTime: 30_000,
    });
  };

  const { data, isError, isFetching, isLoading } = useGetTableData();
  const mtTable = useMantineReactTable<T>({
    columns,
    data: data?.results ?? [],
    manualPagination: true,
    mantinePaginationProps: {
      variant: "filled",
    },
    paginationDisplayMode: "pages",
    onPaginationChange: setPagination,
    onRowSelectionChange: setRowSelection,
    rowCount: data?.count ?? 0,
    getRowId: (row) => row.id,
    enableRowSelection: true,
    icons: montaTableIcons,
    positionActionsColumn: "last",
    mantinePaperProps: {
      sx: { overflow: "visible" },
      shadow: "none",
      withBorder: false,
    },
    mantineBottomToolbarProps: {
      sx: { overflow: "visible" },
    },
    mantineTableHeadRowProps: {
      sx: { overflow: "visible", boxShadow: "none" },
    },
    mantineTableContainerProps: {
      sx: { overflow: "visible" },
    },
    state: {
      isLoading,
      pagination,
      showAlertBanner: isError,
      showSkeletons: isFetching,
      rowSelection,
    },
    initialState: {
      showGlobalFilter: true,
      density: "xs",
      pagination: { pageIndex: 0, pageSize: 10 },
    },
    positionGlobalFilter: "left",
    mantineSearchTextInputProps: {
      placeholder: data?.count
        ? `Search ${data.count} records...`
        : "Search...",
      sx: { minWidth: "300px" },
      variant: "filled",
    },
    enableGlobalFilterModes: false,
    onGlobalFilterChange: setGlobalFilter,
    mantineFilterTextInputProps: {
      sx: { borderBottom: "unset", marginTop: "8px" },
      variant: "filled",
    },
    mantineFilterSelectProps: {
      sx: { borderBottom: "unset", marginTop: "8px" },
      variant: "filled",
    },
    mantineTableBodyCellProps: ({ row }) => ({
      sx: {
        backgroundColor:
          theme.colorScheme === "dark" ? theme.colors.dark[7] : "white",
        cursor: "pointer",
        userSelect: "none",
      },
      onDoubleClick: () => {
        store.set("selectedRecord", row.original);
        store.set("drawerOpen", true);
      },
    }),
    renderTopToolbar: ({ table }) => (
      <TableTopToolbar table={table} store={store} name={name} />
    ),
  });

  return (
    <>
      <MantineReactTable table={mtTable} />
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
      {TableDrawer && <TableDrawer />}
      {TableDeleteModal && <TableDeleteModal />}
    </>
  );
}
