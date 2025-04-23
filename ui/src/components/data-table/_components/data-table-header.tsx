import { TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  flexRender,
  type Header,
  type HeaderGroup,
  type Table,
} from "@tanstack/react-table";

function HeaderCell<K extends Record<string, any>>({
  header,
}: {
  header: Header<K, unknown>;
}) {
  return (
    <TableHead key={header.id}>
      {header.isPlaceholder
        ? null
        : flexRender(header.column.columnDef.header, header.getContext())}
    </TableHead>
  );
}
function HeaderRow<K extends Record<string, any>>({
  headerGroup,
}: {
  headerGroup: HeaderGroup<K>;
}) {
  return (
    <TableRow>
      {headerGroup.headers.map((header) => (
        <HeaderCell key={header.id} header={header} />
      ))}
    </TableRow>
  );
}

// Main header component
export function DataTableHeader<TData extends Record<string, any>>({
  table,
}: {
  table: Table<TData>;
}) {
  return (
    <TableHeader>
      {table.getHeaderGroups().map((headerGroup) => (
        <HeaderRow key={headerGroup.id} headerGroup={headerGroup} />
      ))}
    </TableHeader>
  );
}
