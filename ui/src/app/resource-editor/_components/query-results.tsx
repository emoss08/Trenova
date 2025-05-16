"use no memo";
import type { QueryResult } from "@/types/resource-editor";
import { useVirtualizer } from "@tanstack/react-virtual";

type ResultsTableVirtualizerProps = {
  queryResult: QueryResult;
  parentRef: React.RefObject<HTMLDivElement | null>;
};

export function ResultsTableVirtualizer({
  queryResult,
  parentRef,
}: ResultsTableVirtualizerProps) {
  const rowVirtualizer = useVirtualizer({
    count: queryResult.rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 38,
    overscan: 5,
  });

  return (
    <div
      style={{
        height: `${rowVirtualizer.getTotalSize()}px`,
        width: "100%",
        position: "relative",
      }}
    >
      {rowVirtualizer.getVirtualItems().map((virtualRow) => {
        const row = queryResult.rows[virtualRow.index];
        if (!row) return null; // Should not happen if count is correct

        return (
          <div
            key={virtualRow.key}
            style={{
              position: "absolute",
              top: 0,
              left: 0,
              width: "100%",
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`,
            }}
            className="flex border-b border-border hover:bg-muted/50 items-stretch" // Row styling, min-w-max for highlight
          >
            {queryResult.columns.map((_colName, cellIndex) => (
              <Row
                key={`${virtualRow.key}-${cellIndex}`}
                row={row}
                cellIndex={cellIndex}
              />
            ))}
          </div>
        );
      })}
    </div>
  );
}

export function Row({
  row,
  cellIndex,
}: {
  row: QueryResult["rows"][number];
  cellIndex: number;
}) {
  return (
    <div className="py-2.5 px-3 text-muted-foreground truncate min-w-[180px] basis-[180px]">
      {String(
        row[cellIndex] === null || row[cellIndex] === undefined
          ? "NULL"
          : row[cellIndex],
      )}
    </div>
  );
}
