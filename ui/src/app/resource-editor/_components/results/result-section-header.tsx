import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import type { QueryResult } from "@/types/resource-editor";
import { faEllipsisVertical } from "@fortawesome/pro-solid-svg-icons";

const generateFilename = (baseName: string, extension: string) => {
  const date = new Date();
  const timestamp = `_${date.getFullYear()}${(date.getMonth() + 1)
    .toString()
    .padStart(2, "0")}${date.getDate().toString().padStart(2, "0")}_${date
    .getHours()
    .toString()
    .padStart(2, "0")}${date.getMinutes().toString().padStart(2, "0")}${date
    .getSeconds()
    .toString()
    .padStart(2, "0")}`;
  return `${baseName}${timestamp}.${extension}`;
};

// Helper function to convert data to CSV
const convertToCsv = (data: { columns: string[]; rows: any[][] }): string => {
  if (!data || !data.columns || !data.rows) return "";
  const header = data.columns.join(",") + "\n";
  const rows = data.rows
    .map((row) =>
      row
        .map((cell) => {
          const cellStr = String(
            cell === null || cell === undefined ? "" : cell,
          );
          // Escape quotes and commas
          return `"${cellStr.replace(/"/g, '""')}"`;
        })
        .join(","),
    )
    .join("\n");
  return header + rows;
};

// Helper function to trigger file download
const downloadFile = (filename: string, content: string, mimeType: string) => {
  const blob = new Blob([content], { type: mimeType });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(link.href);
};

export function ResultSectionHeader({
  isExecutingQuery,
  queryResult,
}: {
  isExecutingQuery: boolean;
  queryResult?: QueryResult;
}) {
  const handleExportJson = () => {
    if (queryResult && queryResult.rows && queryResult.rows.length > 0) {
      const jsonData = JSON.stringify(
        { columns: queryResult.columns, rows: queryResult.rows },
        null,
        2,
      );
      downloadFile(
        generateFilename("query_export", "json"),
        jsonData,
        "application/json",
      );
    }
  };

  const handleExportCsv = (isExcel = false) => {
    if (queryResult && queryResult.rows && queryResult.rows.length > 0) {
      const csvData = convertToCsv(queryResult);
      downloadFile(
        generateFilename("query_export", isExcel ? "csv" : "csv"), // Excel opens CSVs fine
        csvData,
        isExcel ? "text/csv;charset=utf-8;" : "text/csv",
      );
    }
  };

  return (
    <ResultSectionHeaderOuter>
      <h2 className="text-lg font-semibold text-foreground">Results</h2>
      <ResultSectionHeaderContent>
        {!isExecutingQuery &&
          queryResult &&
          (queryResult.rows?.length >= 0 || queryResult.message) &&
          !queryResult.error && (
            <span className="text-xs text-muted-foreground">
              {queryResult.rows?.length > 0
                ? `${queryResult.rows.length} row(s) returned`
                : queryResult.message
                  ? queryResult.message
                  : queryResult.rows?.length === 0
                    ? `0 rows returned`
                    : ``}
            </span>
          )}
        <div className="h-6 w-px bg-border" />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              title="Result Actions"
              size="icon"
              variant="outline"
              disabled={
                !queryResult ||
                !queryResult.rows ||
                queryResult.rows.length === 0
              }
            >
              <Icon icon={faEllipsisVertical} />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuGroup>
              <DropdownMenuLabel className="text-xs text-muted-foreground">
                Result Actions
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title="Export to CSV"
                onClick={() => handleExportCsv()}
                disabled={
                  !queryResult ||
                  !queryResult.rows ||
                  queryResult.rows.length === 0
                }
              />
              <DropdownMenuItem
                title="Export to Excel (CSV)"
                onClick={() => handleExportCsv(true)}
                disabled={
                  !queryResult ||
                  !queryResult.rows ||
                  queryResult.rows.length === 0
                }
              />
              <DropdownMenuItem
                title="Export to JSON"
                onClick={handleExportJson}
                disabled={
                  !queryResult ||
                  !queryResult.rows ||
                  queryResult.rows.length === 0
                }
              />
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </ResultSectionHeaderContent>
    </ResultSectionHeaderOuter>
  );
}

function ResultSectionHeaderOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="p-2 border-b border-border flex justify-between items-center min-h-[44px]">
      {children}
    </div>
  );
}

function ResultSectionHeaderContent({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex items-center gap-2">{children}</div>;
}
