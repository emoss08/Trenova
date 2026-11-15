import type { TableArrangement, TableConfig } from "@/types/table-configuration";
import { createContext, use, type ReactNode } from "react";
import { useStore } from "zustand";
import { useShallow } from "zustand/react/shallow";
import {
  selectViewActions,
  type DataTableViewActions,
  type DataTableViewState,
  type DataTableViewStore,
} from "./data-table-view-store";

const DataTableViewContext = createContext<DataTableViewStore | null>(null);

/** Hands the table's view store to everything drawn inside it. */
export function DataTableViewProvider({
  store,
  children,
}: {
  store: DataTableViewStore;
  children: ReactNode;
}) {
  return <DataTableViewContext value={store}>{children}</DataTableViewContext>;
}

function useViewStore(): DataTableViewStore {
  const store = use(DataTableViewContext);
  if (!store) {
    throw new Error("A data table's view is read inside the DataTable that owns it.");
  }
  return store;
}

/**
 * What a part of the table draws from its view, and it re-renders only when that changes:
 * select a single field, a value derived from fields (a row's own "is it unseen"), or several
 * fields wrapped in `useShallow`.
 */
export function useDataTableView<T>(selector: (state: DataTableViewState) => T): T {
  return useStore(useViewStore(), selector);
}

/** The view's actions. They never change, so reading them never re-renders. */
export function useDataTableViewActions(): DataTableViewActions {
  return useStore(useViewStore(), selectViewActions);
}

const selectStyle = (state: DataTableViewState) => ({
  density: state.density,
  formatRules: state.formatRules,
});

/** The table as a saved view would keep it: what it shows, drawn the way the person has it. */
export function useTableConfig(arrangement: TableArrangement): TableConfig {
  const style = useDataTableView(useShallow(selectStyle));
  return { ...arrangement, ...style };
}
