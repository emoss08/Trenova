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
import { MantineReactTable } from "mantine-react-table";
import { useQuery } from "react-query";
import axios from "@/lib/AxiosConfig";
import { montaTableIcons } from "@/components/ui/table/Icons";
import { API_URL } from "@/lib/utils";
import { paymentRecordsTableStore } from "@/stores/CustomerStore";
import { CustomerBillingHistoryTableColumns } from "@/components/customer/CustomerBillingHistoryTableColumns";
import { useMantineTheme } from "@mantine/core";

type Props = {
  id: string;
};

export function CustomerBillingHistoryTable({ id }: Props) {
  const theme = useMantineTheme();
  const [pagination] = paymentRecordsTableStore.use("pagination");
  const [globalFilter, setGlobalFilter] =
    paymentRecordsTableStore.use("globalFilter");
  const [rowSelection] = paymentRecordsTableStore.use("rowSelection");

  const { data, isError, isFetching, isLoading } = useQuery(
    [
      "customerPaymentRecords",
      id,
      pagination.pageIndex,
      pagination.pageSize,
      globalFilter,
    ],
    async () => {
      const offset = pagination.pageIndex * pagination.pageSize;
      const fullUrl = `${API_URL}/billing_history/?customer=${id}&limit=${
        pagination.pageSize
      }&offset=${offset}${globalFilter ? `&search=${globalFilter}` : ""}`;
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
        columns={CustomerBillingHistoryTableColumns()}
        data={data?.results ?? []}
        manualPagination
        onPaginationChange={(newPagination) => {
          paymentRecordsTableStore.set("pagination", newPagination);
        }}
        mantinePaperProps={{
          shadow: "none",
          withBorder: false,
        }}
        mantinePaginationProps={{
          rowsPerPageOptions: ["5", "10"],
        }}
        enableTopToolbar={false}
        enableRowSelection={false}
        rowCount={data?.count ?? 0}
        getRowId={(row) => row.id}
        icons={montaTableIcons}
        state={{
          isLoading,
          pagination: pagination,
          showAlertBanner: isError,
          showSkeletons: isFetching,
          rowSelection,
        }}
        initialState={{
          showGlobalFilter: true,
          density: "xs",
        }}
        positionGlobalFilter="left"
        enableGlobalFilterModes={false}
        mantineTableBodyCellProps={() => {
          return {
            sx: {
              backgroundColor:
                theme.colorScheme === "dark" ? theme.colors.dark[7] : "white",
            },
          };
        }}
        onGlobalFilterChange={(filter: string) => {
          setGlobalFilter(filter);
        }}
      />
    </>
  );
}
