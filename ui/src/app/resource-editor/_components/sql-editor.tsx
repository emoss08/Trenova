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
import { format as formatSQL } from "sql-formatter";
import { ResultsSection } from "./results/result-section";
import { SQLEditorHeader } from "./sql-editor-header";

const AceEditor = lazy(() => import("react-ace"));

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

  // Keep a ref to the Ace editor instance so we can access it outside
  // of the onChange handler (e.g., for formatting).
  const editorRef = useRef<any>(null);

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

  const handleExecuteQuery = useCallback(
    async (queryOverride?: string) => {
      const queryToExecute = queryOverride ?? sqlQuery;

      if (isExecutingQuery) {
        return;
      }
      if (!queryToExecute.trim()) {
        setQueryResult({
          columns: [],
          rows: [] as any[][],
          error: "Query cannot be empty.",
        });
        return;
      }
      setIsExecutingQuery(true);
      setQueryResult(undefined);

      if (queryOverride !== undefined) {
        setSqlQuery(queryOverride);
      }

      try {
        const response = await http.post<ExecuteQueryResponse>(
          "/resource-editor/execute-query/",
          {
            schemaName: results?.data?.schemaName || "public",
            query: queryToExecute,
          } as ExecuteQueryRequest,
        );
        setQueryResult(response.data.result);
      } catch (error: any) {
        console.error("Error executing query:", error);
        setQueryResult({
          columns: [],
          rows: [] as any[][],
          error:
            error.response?.data?.message ||
            error.message ||
            "Failed to execute query.",
        });
      }
      setIsExecutingQuery(false);
    },
    [isExecutingQuery, sqlQuery, results, setQueryResult, setIsExecutingQuery],
  );

  const handleFormatQuery = useCallback(() => {
    if (!editorRef.current) return;

    const raw = editorRef.current.getValue();
    if (!raw.trim()) return;

    const prettified = formatSQL(raw, {
      language: "postgresql",
      tabWidth: 2,
      keywordCase: "upper",
      linesBetweenQueries: 2,
    });

    editorRef.current.setValue(prettified, -1); // -1 => keep cursor
    setSqlQuery(prettified);
  }, [setSqlQuery]);

  useEffect(() => {
    if (
      results?.data &&
      !ace.require("ace/ext/language_tools").customCompleter
    ) {
      const langTools = ace.require("ace/ext/language_tools");

      const customSQLCompleter = {
        getCompletions: async (
          _editor: any,
          session: any,
          _pos: any,
          prefix: string,
          callback: any,
        ) => {
          const currentSchemaName = results?.data?.schemaName || "public";
          const currentTableName = searchParams.selectedTable || "";
          const fullQuery = session.getValue();

          try {
            const response = await http.post<AutocompleteResponse>(
              "/resource-editor/autocomplete/",
              {
                schemaName: currentSchemaName,
                tableName: currentTableName,
                currentQuery: fullQuery,
                prefix,
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
        ace.require(
          "ace/autocomplete",
        ).Autocomplete.prototype.defaultCompleters = [
          langTools.keyWordCompleter,
          langTools.textCompleter,
          langTools.snippetCompleter,
          customSQLCompleter,
        ];
      } else {
        console.warn(
          "Could not set custom Ace completers using known methods.",
        );
        if (langTools.addCompleter) {
          langTools.addCompleter(customSQLCompleter);
        }
      }
      ace.require("ace/ext/language_tools").customCompleter =
        customSQLCompleter;
    }
  }, [results, searchParams.selectedTable]);

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
            height="100%"
            readOnly={isExecutingQuery}
            placeholder="Type your SQL query here, then press Ctrl+Shift+Enter or click Execute."
            onLoad={(editor: any) => {
              editorRef.current = editor;
            }}
            commands={[
              {
                name: "executeQuery",
                bindKey: { win: "Ctrl-Shift-Enter", mac: "Cmd-Shift-Enter" },
                exec: (editor: any) => {
                  const currentQuery = editor.getValue();
                  console.info("executing query", currentQuery);
                  return handleExecuteQuery(currentQuery);
                },
              },
              {
                name: "formatSQL",
                bindKey: { win: "Ctrl-Shift-F", mac: "Cmd-Shift-F" },
                exec: () => {
                  handleFormatQuery();
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
    <div className="size-full flex flex-col px-4 space-y-4 overflow-hidden">
      {children}
    </div>
  );
}

function SQLEditorInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col border border-border rounded-md bg-sidebar overflow-hidden h-[calc(50vh-100px)]">
      {children}
    </div>
  );
}
