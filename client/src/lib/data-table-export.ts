import { fetchGraphQLData } from "@/hooks/data-table/use-data-table-query";
import type { DataTableGraphQLConfig, DataTableQueryOptions } from "@/types/data-table";
import type { Column } from "@tanstack/react-table";

export type ExportColumn<TData> = {
  id: string;
  header: string;
  getValue: (row: TData) => unknown;
};

export type ExportScope = "page" | "all";

export const EXPORT_MAX_ROWS = 10_000;
const EXPORT_PAGE_SIZE = 200;

function getNestedValue(row: unknown, path: string): unknown {
  if (row === null || typeof row !== "object") return undefined;
  if (!path.includes(".")) {
    return (row as Record<string, unknown>)[path];
  }
  let current: unknown = row;
  for (const segment of path.split(".")) {
    if (current === null || typeof current !== "object") return undefined;
    current = (current as Record<string, unknown>)[segment];
  }
  return current;
}

export function buildExportColumns<TData>(
  leafColumns: Column<TData, unknown>[],
  visibleOnly: boolean,
): ExportColumn<TData>[] {
  const exportColumns: ExportColumn<TData>[] = [];

  for (const column of leafColumns) {
    const def = column.columnDef;
    const meta = def.meta;
    if (meta?.exportable === false) continue;
    if (visibleOnly && !column.getIsVisible()) continue;

    const accessorKey = "accessorKey" in def ? String(def.accessorKey) : null;
    const exportValue = meta?.exportValue;
    if (!accessorKey && !exportValue) continue;

    const header =
      typeof def.header === "string" ? def.header : (meta?.label ?? column.id);

    exportColumns.push({
      id: column.id,
      header,
      getValue: exportValue ?? ((row: TData) => getNestedValue(row, accessorKey as string)),
    });
  }

  return exportColumns;
}

function formatCsvValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number" || typeof value === "bigint") return value.toString();
  return JSON.stringify(value) ?? "";
}

function escapeCsvValue(value: string): string {
  if (/[",\n\r]/.test(value)) {
    return `"${value.replaceAll('"', '""')}"`;
  }
  return value;
}

export function buildCsv<TData>(rows: TData[], columns: ExportColumn<TData>[]): string {
  const lines: string[] = Array.from({ length: rows.length + 1 }, () => "");
  lines[0] = columns.map((c) => escapeCsvValue(c.header)).join(",");

  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    lines[i + 1] = columns
      .map((c) => escapeCsvValue(formatCsvValue(c.getValue(row))))
      .join(",");
  }

  return lines.join("\r\n");
}

export type ExportProgress = {
  fetched: number;
  total: number | null;
};

export type FetchAllRowsParams<TData extends Record<string, unknown>> = {
  graphql: DataTableGraphQLConfig<TData>;
  options: Omit<DataTableQueryOptions, "cursor">;
  maxRows?: number;
  onProgress?: (progress: ExportProgress) => void;
  isCancelled?: () => boolean;
};

export async function fetchAllRows<TData extends Record<string, unknown>>({
  graphql,
  options,
  maxRows = EXPORT_MAX_ROWS,
  onProgress,
  isCancelled,
}: FetchAllRowsParams<TData>): Promise<TData[]> {
  const rows: TData[] = [];
  let cursor: string | undefined;

  for (;;) {
    if (isCancelled?.()) break;

    const pageSize = Math.min(EXPORT_PAGE_SIZE, maxRows - rows.length);
    if (pageSize <= 0) break;

    const page = await fetchGraphQLData<TData>(pageSize, graphql, { ...options, cursor });
    rows.push(...page.results);
    onProgress?.({
      fetched: rows.length,
      total: page.pageInfo?.totalCount != null ? Math.min(page.pageInfo.totalCount, maxRows) : null,
    });

    if (!page.pageInfo?.hasNextPage || !page.pageInfo.endCursor) break;
    cursor = page.pageInfo.endCursor;
  }

  return rows;
}

export function downloadCsv(csv: string, filename: string): void {
  const blob = new Blob(["\uFEFF", csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function exportFilename(resource: string): string {
  const stamp = new Date().toISOString().slice(0, 19).replaceAll(":", "-");
  return `${resource.toLowerCase().replaceAll(/\s+/g, "-")}-${stamp}.csv`;
}
