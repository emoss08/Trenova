import { useTheme } from "@/components/theme-provider";
import { http, type HttpClientResponse } from "@/lib/http-client";
import { resourceEditorSearchParamsParser } from "@/lib/search-params/resource-editor";
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
import ace from "ace-builds";
import "ace-builds/src-noconflict/ext-language_tools";
import "ace-builds/src-noconflict/mode-sql";
import "ace-builds/src-noconflict/theme-dawn";
import "ace-builds/src-noconflict/theme-tomorrow_night_bright";
import { useQueryStates } from "nuqs";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
// import AceEditor from "react-ace";
import { ResultsSection } from "./results/result-section";
import { SQLEditorHeader } from "./sql-editor-header";

const AceEditor = lazy(() => import("react-ace"));

// Configure Ace basePath for Vite environment
ace.config.set("basePath", "/ace-builds/src-noconflict");

export default function SQLEditor({
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
        getCompletions: async (session: any, prefix: string, callback: any) => {
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

  return (
    <SQLEditorOuter>
      <SQLEditorInner>
        <SQLEditorHeader
          handleExecuteQuery={handleExecuteQuery}
          isExecutingQuery={isExecutingQuery}
          sqlQuery={sqlQuery}
        />
        <Suspense fallback={<div>Loading...</div>}>
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
        </Suspense>
      </SQLEditorInner>
      <ResultsSection
        queryResult={queryResult}
        isExecutingQuery={isExecutingQuery}
      />
    </SQLEditorOuter>
  );
}

function SQLEditorOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="w-3/4 flex flex-col px-4 space-y-4 overflow-hidden h-full">
      {children}
    </div>
  );
}

function SQLEditorInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col border border-border rounded-md bg-sidebar overflow-hidden flex-[60%]">
      {children}
    </div>
  );
}
