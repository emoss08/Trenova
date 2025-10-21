import { Icon } from "@/components/ui/icons";
import { Sortable, SortableItem } from "@/components/ui/sortable";
import { toSentenceCase } from "@/lib/utils";
import { faGhost } from "@fortawesome/pro-regular-svg-icons";
import { Column } from "@tanstack/react-table";
import { useState } from "react";
import { useDataTable } from "../../data-table-provider";
import { ColumnSortItem } from "./sorted-column-item";

type SortedColumnsProps = {
  sortedColumns: Column<any, any>[];
  searchQuery: string;
};

export function ColumnSortContent({
  sortedColumns,
  searchQuery,
}: SortedColumnsProps) {
  const { table } = useDataTable();
  const [drag, setDrag] = useState(false);

  const filteredColumns = sortedColumns.filter((column) =>
    searchQuery
      ? toSentenceCase(column.id)
          .toLowerCase()
          .includes(searchQuery.toLowerCase())
      : true,
  );

  return (
    <ColumnSortOuter>
      {filteredColumns.length > 0 ? (
        <Sortable
          value={filteredColumns.map((c) => ({ id: c.id }))}
          onValueChange={(items) => {
            const newOrder = items.map((c) => c.id);
            table.setColumnOrder(newOrder);
          }}
          overlay={<div className="h-8 w-full rounded-md bg-muted/60" />}
          onDragStart={() => setDrag(true)}
          onDragEnd={() => setDrag(false)}
          onDragCancel={() => setDrag(false)}
        >
          {filteredColumns.map((column) => (
            <SortableItem key={column.id} value={column.id} asChild>
              <ColumnSortItem
                column={column}
                searchQuery={searchQuery}
                drag={drag}
              />
            </SortableItem>
          ))}
        </Sortable>
      ) : (
        <NoColumnsFound searchQuery={searchQuery} />
      )}
    </ColumnSortOuter>
  );
}

function ColumnSortOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col py-2">{children}</div>;
}

function NoColumnsFound({ searchQuery }: { searchQuery: string }) {
  return (
    <span className="flex flex-col justify-center items-center p-2 text-sm">
      <div className="flex items-center justify-center bg-muted rounded-md p-2 size-14 mb-2">
        <Icon icon={faGhost} className="size-8 text-muted-foreground" />
      </div>
      <span className="font-medium">
        {searchQuery ? "No columns match your search" : "No columns found"}
      </span>
      <span className="text-xs text-muted-foreground">
        Try adjusting your search query or create a new column.
      </span>
    </span>
  );
}
