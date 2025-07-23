/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { createGlobalStore } from "@/hooks/use-global-store";
import { TableStoreProps } from "@/types/data-table";

export const useTableStore = createGlobalStore<TableStoreProps<any>>({
  pagination: {
    pageIndex: 0,
    pageSize: 10,
  },
  exportModalOpen: false,
  columnVisibility: {},
  currentRecord: undefined,
  rowSelection: {},
  columnFilters: [],
  sorting: [],
  showCreateModal: false,
  showFilterDialog: false,
  editModalOpen: false,
  initialPageSize: 10,
  defaultSort: [],
  showImportModal: false,
  onDataChange: () => {},
  setInitialPageSize: () => {},
  setDefaultSort: () => {},
});
