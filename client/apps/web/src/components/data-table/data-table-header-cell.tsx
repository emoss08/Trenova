import {
  columnHeaderLabel,
  columnSizeVar,
  pinnedCellClass,
  pinnedSideStyle,
} from "@/lib/data-table";
import { useTableAtom } from "@trenova/shared/hooks/use-table-atom";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { flexRender, type RowData } from "@tanstack/react-table";
import { TableHead } from "@trenova/shared/components/ui/table";
import { cn } from "@trenova/shared/lib/utils";
import type { Header, SortDirection, SortField } from "@trenova/shared/types/data-table";
import { DataTableColumnHeader } from "./data-table-column-header";
import { DataTableColumnResizeHandle } from "./data-table-column-resize-handle";
import {
  DataTableHeaderFacetFilter,
  type DataTableHeaderFacets,
} from "./data-table-header-facet-filter";
import type { FilterableField } from "@/lib/data-table";

type DataTableHeaderCellProps<TData extends RowData> = {
  header: Header<TData, unknown>;
  /** The field this column's header filters by value, when the table can count it. */
  facetField?: FilterableField | null;
  facets?: DataTableHeaderFacets;
  sort: SortField[];
  onSort: (field: string, direction: SortDirection | null) => void;
};

function ariaSortOf(sort: readonly SortField[], field: string) {
  const direction = sort.find((entry) => entry.field === field)?.direction;
  if (direction === "asc") return "ascending";
  if (direction === "desc") return "descending";
  return "none";
}

export function DataTableHeaderCell<TData extends RowData>({
  header,
  sort,
  onSort,
  facetField,
  facets,
}: DataTableHeaderCellProps<TData>) {
  const { column } = header;
  const meta = column.columnDef.meta;
  const isSortable = meta?.sortable !== false;
  // The column object outlives a pin or unpin, so the side and the boundary edge
  // are read from the pinning state rather than from the column alone.
  const isPinned = useTableAtom(column.table.atoms.columnPinning, () => column.getIsPinned());
  const pinClass = useTableAtom(column.table.atoms.columnPinning, () => pinnedCellClass(column));
  const canReorder = column.id !== "select" && !isPinned;

  const { listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: column.id,
    disabled: !canReorder,
  });

  return (
    <TableHead
      ref={setNodeRef}
      className={cn(
        // design-tokens-ignore: column heads are uppercase across every data table by product decision
        "group/head border-border relative border-b uppercase",
        pinClass,
        // A pinned head needs to be opaque over the scrolling content, not a
        // different colour from the heads beside it, and above the resize
        // handles of the heads that scroll under it.
        isPinned && "bg-sunken z-20",
        isDragging && "z-20 opacity-80",
      )}
      style={{
        width: `var(${columnSizeVar(column.id)})`,
        ...pinnedSideStyle(column.id, isPinned),
        transform: canReorder ? CSS.Translate.toString(transform) : undefined,
        transition: canReorder ? transition : undefined,
      }}
      {...(canReorder ? listeners : {})}
      role="columnheader"
      aria-colindex={header.index + 1}
      aria-sort={isSortable ? ariaSortOf(sort, meta?.apiField ?? column.id) : undefined}
      data-column-id={column.id}
    >
      {header.isPlaceholder ? null : isSortable ? (
        <DataTableColumnHeader
          column={column}
          title={columnHeaderLabel(column)}
          currentSort={sort}
          onSort={onSort}
        />
      ) : (
        flexRender(column.columnDef.header, header.getContext())
      )}
      {facetField && facets ? (
        <span className="absolute top-1/2 right-2.5 z-10 -translate-y-1/2">
          <DataTableHeaderFacetFilter field={facetField} facets={facets} />
        </span>
      ) : null}
      <DataTableColumnResizeHandle header={header} />
    </TableHead>
  );
}
