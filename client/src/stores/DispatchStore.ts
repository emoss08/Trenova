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
import { createGlobalStore } from "@/lib/useGlobalStore";
import { TableStoreProps } from "@/types/tables";
import { CommentType, DelayCode, FleetCode, Rate } from "@/types/dispatch";

export const useDelayCodeStore = createGlobalStore<TableStoreProps<DelayCode>>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  drawerOpen: false,
  selectedRecord: null,
  globalFilter: "",
  exportModalOpen: false,
  deleteModalOpen: false,
  createModalOpen: false,
  rowSelection: {},
  columnFilters: false,
  errorCount: 0,
});

export const useFleetCodeStore = createGlobalStore<TableStoreProps<FleetCode>>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  drawerOpen: false,
  selectedRecord: null,
  globalFilter: "",
  exportModalOpen: false,
  deleteModalOpen: false,
  createModalOpen: false,
  columnFilters: false,
  rowSelection: {},
  errorCount: 0,
});

export const useCommentTypeStore = createGlobalStore<
  TableStoreProps<CommentType>
>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  drawerOpen: false,
  selectedRecord: null,
  globalFilter: "",
  exportModalOpen: false,
  deleteModalOpen: false,
  createModalOpen: false,
  columnFilters: false,
  rowSelection: {},
  errorCount: 0,
});

export const useRateStore = createGlobalStore<TableStoreProps<Rate>>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  drawerOpen: false,
  selectedRecord: null,
  globalFilter: "",
  exportModalOpen: false,
  deleteModalOpen: false,
  createModalOpen: false,
  columnFilters: false,
  rowSelection: {},
  errorCount: 0,
});
