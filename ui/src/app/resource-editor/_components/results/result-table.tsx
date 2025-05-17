"use no memo";
import { cn } from "@/lib/utils";
import type { QueryResult } from "@/types/resource-editor";
import { useVirtualizer } from "@tanstack/react-virtual";
import "ace-builds/src-noconflict/ext-language_tools";
import "ace-builds/src-noconflict/mode-sql";
import "ace-builds/src-noconflict/theme-dawn";
import "ace-builds/src-noconflict/theme-tomorrow_night_bright";

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
            className="flex border-b border-border bg-background hover:bg-muted/50 items-stretch" // Row styling, min-w-max for highlight
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

export function ResultsTable({
  queryResult,
  parentRef,
}: {
  queryResult: QueryResult;
  parentRef: React.RefObject<HTMLDivElement | null>;
}) {
  return (
    <ResultTableOuter>
      <ResultTableInner>
        {queryResult.columns.map((colName, index) => (
          <div
            key={index}
            className={cn(
              "py-2.5 px-3 flex-shrink-0 text-foreground basis-[180px] min-w-[180px] truncate border-r border-border",
              index === queryResult.columns.length - 1
                ? "border-r-0"
                : "border-r",
            )}
            title={colName}
          >
            {colName}
          </div>
        ))}
      </ResultTableInner>
      {/* Virtualized Rows Container */}
      <ResultsTableVirtualizer
        queryResult={queryResult}
        parentRef={parentRef}
      />
    </ResultTableOuter>
  );
}

function ResultTableOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="font-mono text-sm select-text relative min-w-max">
      {children}
    </div>
  );
}

function ResultTableInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex bg-muted sticky top-0 z-10 border-b border-border select-none min-w-max">
      {children}
    </div>
  );
}
