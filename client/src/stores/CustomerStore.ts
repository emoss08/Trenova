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

import { MRT_RowSelectionState } from "mantine-react-table";
import { createGlobalStore } from "@/utils/zustand";
import { TableStoreProps } from "@/types/tables";
import { Customer } from "@/types/apps/customer";
import { BillingHistory } from "@/types/apps/billing";

type customerStoreProps = {
  editModalOpen: boolean;
  activeTab: string | null;
  createRuleProfileModalOpen: boolean;
};

type paymentRecordsTableStoreProps<T extends Record<string, unknown>> = {
  pagination: {
    pageIndex: number;
    pageSize: number;
  };
  selectedRecord: T | null;
  globalFilter: string;
  columnFilters: boolean;
  rowSelection: MRT_RowSelectionState;
};

export const customerTableStore = createGlobalStore<
  Omit<TableStoreProps<Customer>, "drawerOpen">
>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  viewModalOpen: false,
  editModalOpen: false,
  selectedRecord: null,
  globalFilter: "",
  exportModalOpen: false,
  deleteModalOpen: false,
  createModalOpen: false,
  columnFilters: false,
  rowSelection: {},
  errorCount: 0,
});

export const paymentRecordsTableStore = createGlobalStore<
  paymentRecordsTableStoreProps<BillingHistory>
>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  selectedRecord: null,
  globalFilter: "",
  columnFilters: false,
  rowSelection: {},
});

export const customerStore = createGlobalStore<customerStoreProps>({
  editModalOpen: false,
  createRuleProfileModalOpen: false,
  activeTab: "overview",
});
