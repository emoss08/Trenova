import { TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { flexRender, Table } from "@tanstack/react-table";
import { memo } from "react";

// Memoized header cell component to prevent unnecessary re-renders
const HeaderCell = memo(
  ({ header }: { header: any }) => (
    <TableHead colSpan={header.colSpan}>
      {header.isPlaceholder
        ? null
        : flexRender(header.column.columnDef.header, header.getContext())}
    </TableHead>
  ),
  // Custom compare function to prevent unnecessary re-renders
  (prev, next) => prev.header.id === next.header.id,
);

HeaderCell.displayName = "HeaderCell";

// Memoized header row component
const HeaderRow = memo(
  ({ headerGroup }: { headerGroup: any }) => (
    <TableRow>
      {headerGroup.headers.map((header: any) => (
        <HeaderCell key={header.id} header={header} />
      ))}
    </TableRow>
  ),
  // Only re-render if the header group ID changes
  (prev, next) => prev.headerGroup.id === next.headerGroup.id,
);

HeaderRow.displayName = "HeaderRow";

// Memoize the entire header component
export const DataTableHeader = memo(function DataTableHeaderInner<
  K extends Record<string, any>,
>({ table }: { table: Table<K> }) {
  return (
    <TableHeader>
      {table.getHeaderGroups().map((headerGroup) => (
        <HeaderRow key={headerGroup.id} headerGroup={headerGroup} />
      ))}
    </TableHeader>
  );
});

DataTableHeader.displayName = "DataTableHeader";
