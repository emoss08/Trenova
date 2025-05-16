import { Kbd } from "@/components/kbd";
import { useTheme } from "@/components/theme-provider";
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { http, type HttpClientResponse } from "@/lib/http-client";
import { resourceEditorSearchParamsParser } from "@/lib/search-params/resource-editor";
import { cn } from "@/lib/utils";
import type {
  ExecuteQueryRequest,
  ExecuteQueryResponse,
  QueryResult,
  SchemaInformation,
} from "@/types/resource-editor";
import {
  AutocompleteRequest,
  AutocompleteResponse,
} from "@/types/resource-editor";
import { faEllipsisVertical } from "@fortawesome/pro-solid-svg-icons";
import ace from "ace-builds";
import "ace-builds/src-noconflict/ext-language_tools";
import "ace-builds/src-noconflict/mode-sql";
import "ace-builds/src-noconflict/theme-dawn";
import "ace-builds/src-noconflict/theme-tomorrow_night_bright";
import { AlertTriangleIcon, PlayIcon, TerminalIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useRef, useState } from "react";
import AceEditor from "react-ace";
import { ResultsTableVirtualizer } from "./query-results";

// Configure Ace basePath for Vite environment
ace.config.set("basePath", "/ace-builds/src-noconflict");

// Helper function to generate a filename
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

export function SQLEditor({
  results,
}: {
  results?: HttpClientResponse<SchemaInformation>;
}) {
  const { theme } = useTheme();
  const [searchParams, setSearchParams] = useQueryStates(
    resourceEditorSearchParamsParser,
  );
  const [isExecutingQuery, setIsExecutingQuery] = useState(false);
  const [sqlQuery, setSqlQuery] = useState<string>("");
  const [queryResult, setQueryResult] = useState<QueryResult | undefined>(
    undefined,
  );
  const parentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const getEffectiveTheme = () => {
      if (theme === "system") {
        return window.matchMedia("(prefers-color-scheme: dark)").matches
          ? "dark"
          : "light";
      }
      return theme;
    };

    const effectiveTheme = getEffectiveTheme();
    setSearchParams({
      aceTheme: effectiveTheme === "dark" ? "tomorrow_night_bright" : "dawn",
    });

    // Listener for system theme changes
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = () => {
      if (theme === "system") {
        setSearchParams({
          aceTheme: mediaQuery.matches ? "tomorrow_night_bright" : "dawn",
        });
      }
    };

    if (theme === "system") {
      mediaQuery.addEventListener("change", handleChange);
    }

    return () => {
      mediaQuery.removeEventListener("change", handleChange);
    };
  }, [theme, setSearchParams]);

  const handleExecuteQuery = useCallback(async () => {
    if (isExecutingQuery) {
      return;
    }
    if (!sqlQuery.trim()) {
      setQueryResult({
        columns: [],
        rows: [],
        error: "Query cannot be empty.",
      });
      return;
    }
    setIsExecutingQuery(true);
    setQueryResult(undefined); // Clear previous results
    try {
      const response = await http.post<ExecuteQueryResponse>(
        "/resource-editor/execute-query/",
        {
          schemaName: results?.data?.schemaName || "public",
          query: sqlQuery,
        } as ExecuteQueryRequest,
      );
      setQueryResult(response.data.result);
    } catch (error: any) {
      console.error("Error executing query:", error);
      setQueryResult({
        columns: [],
        rows: [],
        error:
          error.response?.data?.message ||
          error.message ||
          "Failed to execute query.",
      });
    }
    setIsExecutingQuery(false);
  }, [
    isExecutingQuery,
    sqlQuery,
    results,
    setQueryResult,
    setIsExecutingQuery,
  ]);

  const handleExecuteQueryRef = useRef(handleExecuteQuery);

  useEffect(() => {
    handleExecuteQueryRef.current = handleExecuteQuery;
  }, [handleExecuteQuery]);

  useEffect(() => {
    if (
      results?.data &&
      !ace.require("ace/ext/language_tools").customCompleter
    ) {
      const langTools = ace.require("ace/ext/language_tools");

      const customSQLCompleter = {
        getCompletions: async (
          editor: any,
          session: any,
          pos: any,
          prefix: string,
          callback: any,
        ) => {
          const currentSchemaName = results.data?.schemaName || "public";
          const currentTableName = searchParams.selectedTable || "";
          const fullQuery = session.getValue();

          try {
            const response = await http.post<AutocompleteResponse>(
              "/resource-editor/autocomplete/",
              {
                schemaName: currentSchemaName,
                tableName: currentTableName,
                currentQuery: fullQuery,
                prefix: prefix,
              } as AutocompleteRequest,
            );

            if (response.data && response.data.suggestions) {
              callback(
                null,
                response.data.suggestions.map((s: any) => ({
                  value: s.value,
                  caption: s.caption,
                  meta: s.meta,
                  score: s.score,
                })),
              );
            }
          } catch (error) {
            console.error("Error fetching autocomplete suggestions:", error);
            callback(null, []);
          }
        },
      };
      // Register the custom completer
      // Check if defaultCompleters exists, it was added in a later version of ace-builds
      if (langTools.setCompleters) {
        // Older API
        langTools.setCompleters([
          langTools.keyWordCompleter,
          langTools.textCompleter,
          langTools.snippetCompleter,
          customSQLCompleter,
        ]);
      } else if (
        ace.require("ace/autocomplete").Autocomplete.prototype.defaultCompleters
      ) {
        // Newer Ace versions might have it here
        ace.require(
          "ace/autocomplete",
        ).Autocomplete.prototype.defaultCompleters = [
          langTools.keyWordCompleter,
          langTools.textCompleter,
          langTools.snippetCompleter,
          customSQLCompleter,
        ];
      } else {
        // Fallback or log if no clear way to set completers, though one of the above should work for most ace-builds versions
        console.warn(
          "Could not set custom Ace completers using known methods.",
        );
        // As a simpler fallback, trying to add it if an addCompleter method exists.
        if (langTools.addCompleter) {
          langTools.addCompleter(customSQLCompleter);
        }
      }
      // Store a flag to avoid re-registering (optional)
      ace.require("ace/ext/language_tools").customCompleter =
        customSQLCompleter;
    }
  }, [results, searchParams.selectedTable]); // Rerun when results or selectedTable changes to update context for completer

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
    <div className="w-3/4 flex flex-col px-4 space-y-4 overflow-hidden h-full">
      <div className="flex flex-col border border-border rounded-md bg-sidebar overflow-hidden flex-[60%]">
        <div className="flex justify-between items-center p-2 border-b border-border min-h-[44px]">
          <h2 className="text-lg font-semibold text-foreground flex items-center">
            <TerminalIcon className="size-5 mr-2" /> SQL Editor
            {searchParams.selectedTable && (
              <span className="text-sm text-muted-foreground ml-2">
                (Context: {searchParams.selectedTable})
              </span>
            )}
          </h2>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  onClick={handleExecuteQuery}
                  size="sm"
                  disabled={!sqlQuery.trim() || isExecutingQuery}
                  isLoading={isExecutingQuery}
                  loadingText="Executing..."
                >
                  <PlayIcon
                    className={`mr-2 h-4 w-4 ${isExecutingQuery ? "animate-spin" : ""}`}
                  />
                  Execute Query
                </Button>
              </TooltipTrigger>
              <TooltipContent className="flex items-center gap-2 text-xs">
                <Kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                  Ctrl
                </Kbd>
                <Kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                  Enter
                </Kbd>
                <p>to execute the query</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
        <AceEditor
          mode="sql"
          theme={searchParams.aceTheme}
          onChange={setSqlQuery}
          fontSize={14}
          lineHeight={19}
          showPrintMargin={true}
          showGutter={true}
          highlightActiveLine={true}
          name="sql-editor"
          editorProps={{ $blockScrolling: true }}
          value={sqlQuery}
          width="100%"
          height="100%" // Fill available height from parent
          readOnly={isExecutingQuery}
          placeholder="Type your SQL query here, then press Ctrl+Shift+Enter or click Execute."
          commands={[
            {
              name: "executeQuery",
              bindKey: { win: "Ctrl-Shift-Enter", mac: "Cmd-Shift-Enter" },
              exec: () => {
                if (handleExecuteQueryRef.current) {
                  handleExecuteQueryRef.current();
                }
              },
            },
          ]}
          setOptions={{
            enableBasicAutocompletion: true,
            enableLiveAutocompletion: true,
            enableSnippets: true,
            showLineNumbers: true,
            enableMobileMenu: true,
            tabSize: 2,
          }}
          className="!h-full" // Override any default height to ensure it fills parent
        />
      </div>

      {/* Results Section */}
      <div className="flex flex-col border border-border rounded-md bg-sidebar flex-[40%] max-h-[45vh]">
        <div className="p-2 border-b border-border flex justify-between items-center min-h-[44px]">
          <h2 className="text-lg font-semibold text-foreground">Results</h2>
          <div className="flex items-center gap-2">
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
          </div>
        </div>
        <div ref={parentRef} className="flex-grow overflow-auto min-h-0">
          {isExecutingQuery && (
            <p className="text-center p-4 text-muted-foreground">
              Executing query...
            </p>
          )}

          {!isExecutingQuery && queryResult?.error && (
            <div className="p-3 m-2 text-red-400 bg-red-900/40 border border-red-700/50 rounded-md">
              <div className="flex items-center font-semibold mb-1">
                <AlertTriangleIcon className="w-5 h-5 mr-2 flex-shrink-0" />{" "}
                Error
              </div>
              <pre className="text-xs whitespace-pre-wrap break-words font-mono">
                {queryResult.error}
              </pre>
            </div>
          )}

          {!isExecutingQuery &&
            queryResult &&
            typeof queryResult.rows === "undefined" &&
            !queryResult.error &&
            queryResult.message && (
              <div className="p-3 m-2 text-green-500 bg-green-900/40 border border-green-700/50 rounded-md">
                {queryResult.message}
              </div>
            )}

          {!isExecutingQuery && queryResult?.rows && !queryResult.error && (
            <>
              {queryResult.rows.length > 0 ? (
                <div className="font-mono text-sm select-text relative min-w-max">
                  <div className="flex bg-background sticky top-0 z-10 border-b border-border select-none min-w-max">
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
                  </div>
                  {/* Virtualized Rows Container */}
                  <ResultsTableVirtualizer
                    queryResult={queryResult}
                    parentRef={parentRef}
                  />
                </div>
              ) : (
                <p className="text-muted-foreground p-4 text-center">
                  Query executed successfully, 0 rows returned.
                </p>
              )}
            </>
          )}

          {!isExecutingQuery && !queryResult && (
            <p className="text-muted-foreground p-4 text-center">
              Execute a query to see results here.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
