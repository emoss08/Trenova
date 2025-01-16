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
  onDataChange: () => {},
  setInitialPageSize: () => {},
  setDefaultSort: () => {},
});
