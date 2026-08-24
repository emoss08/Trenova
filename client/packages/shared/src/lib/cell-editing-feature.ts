import {
  assignPrototypeAPIs,
  assignTableAPIs,
  functionalUpdate,
  makeStateUpdater,
  type CellData,
  type OnChangeFn,
  type RowData,
  type TableFeature,
  type TableFeatures,
  type Updater,
} from "@tanstack/react-table";

export type EditingCellPosition = {
  rowId: string;
  columnId: string;
};

export type CellEditingState = EditingCellPosition | null;

export type CellEditCommitParams<TData extends RowData = RowData> = {
  rowId: string;
  columnId: string;
  value: unknown;
  previousValue: unknown;
  row: TData;
};

export type CellEditCommitFn<TData extends RowData = RowData> = (
  params: CellEditCommitParams<TData>,
) => void | Promise<void>;

interface CellEditingTableState {
  cellEditing: CellEditingState;
}

interface CellEditingOptions<TData extends RowData> {
  enableCellEditing?: boolean;
  onCellEditingChange?: OnChangeFn<CellEditingState>;
  onCellEditCommit?: CellEditCommitFn<TData>;
}

interface CellEditingColumnDef {
  enableCellEditing?: boolean;
}

interface CellEditingTable {
  getEditingCell: () => CellEditingState;
  setEditingCell: (updater: Updater<CellEditingState>) => void;
  stopEditing: () => void;
}

interface CellEditingCell {
  getCanEdit: () => boolean;
  getIsEditing: () => boolean;
  startEditing: () => void;
  stopEditing: () => void;
  commitEdit: (value: unknown) => Promise<void>;
}

declare module "@tanstack/table-core" {
  interface Plugins {
    cellEditingFeature: TableFeature;
  }
  interface TableState_FeatureMap {
    cellEditingFeature: CellEditingTableState;
  }
  interface TableOptions_FeatureMap<TFeatures extends TableFeatures, TData extends RowData> {
    cellEditingFeature: CellEditingOptions<TData>;
  }
  interface Table_FeatureMap<TFeatures extends TableFeatures, TData extends RowData> {
    cellEditingFeature: CellEditingTable;
  }
  interface ColumnDef_FeatureMap<
    TFeatures extends TableFeatures,
    TData extends RowData,
    TValue extends CellData,
  > {
    cellEditingFeature: CellEditingColumnDef;
  }
  interface Cell_FeatureMap {
    cellEditingFeature: CellEditingCell;
  }
}

type CellEditingAtomHost = {
  atoms: { cellEditing: { get: () => CellEditingState } };
};

type CellEditingOptionsHost = {
  options: CellEditingOptions<RowData>;
};

type CellEditingTableHost = CellEditingTable & CellEditingOptionsHost;

type EditableCellHost = {
  table: CellEditingTableHost & CellEditingAtomHost & CellEditingOptionsHost;
  row: { id: string; original: RowData };
  column: { id: string; columnDef: CellEditingColumnDef };
  getValue: () => unknown;
};

const readEditingCell = (table: unknown): CellEditingState =>
  (table as CellEditingAtomHost).atoms.cellEditing.get();

export const cellEditingFeature: TableFeature = {
  getInitialState: (state) => ({ cellEditing: null, ...state }),
  getDefaultTableOptions: (table) => ({
    onCellEditingChange: makeStateUpdater("cellEditing", table),
  }),
  constructTableAPIs: (table) => {
    assignTableAPIs("cellEditingFeature", table, {
      table_getEditingCell: {
        fn: () => readEditingCell(table),
      },
      table_setEditingCell: {
        fn: (updater: Updater<CellEditingState>) =>
          (table.options as CellEditingOptions<RowData>).onCellEditingChange?.((old) =>
            functionalUpdate(updater, old),
          ),
      },
      table_stopEditing: {
        fn: () => (table.options as CellEditingOptions<RowData>).onCellEditingChange?.(() => null),
      },
    });
  },
  assignCellPrototype: (prototype, table) => {
    assignPrototypeAPIs("cellEditingFeature", prototype, table, {
      cell_getCanEdit: {
        fn: (cell) => {
          const host = cell as unknown as EditableCellHost;
          return (
            host.table.options.enableCellEditing === true &&
            host.column.columnDef.enableCellEditing === true
          );
        },
      },
      cell_getIsEditing: {
        fn: (cell) => {
          const host = cell as unknown as EditableCellHost;
          const editing = readEditingCell(host.table);
          return (
            editing !== null && editing.rowId === host.row.id && editing.columnId === host.column.id
          );
        },
      },
      cell_startEditing: {
        fn: (cell) => {
          const host = cell as unknown as EditableCellHost;
          if (!(cell as unknown as CellEditingCell).getCanEdit()) return;
          host.table.setEditingCell({ rowId: host.row.id, columnId: host.column.id });
        },
      },
      cell_stopEditing: {
        fn: (cell) => {
          const host = cell as unknown as EditableCellHost;
          if (!(cell as unknown as CellEditingCell).getIsEditing()) return;
          host.table.stopEditing();
        },
      },
      cell_commitEdit: {
        fn: async (cell, value: unknown) => {
          const host = cell as unknown as EditableCellHost;
          if (!(cell as unknown as CellEditingCell).getIsEditing()) return;
          const commit = host.table.options.onCellEditCommit;
          if (commit) {
            await commit({
              rowId: host.row.id,
              columnId: host.column.id,
              value,
              previousValue: host.getValue(),
              row: host.row.original,
            });
          }
          host.table.stopEditing();
        },
      },
    });
  },
};
