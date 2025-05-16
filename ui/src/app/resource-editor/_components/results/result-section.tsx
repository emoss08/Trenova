import type { QueryResult } from "@/types/resource-editor";
import { AlertTriangleIcon } from "lucide-react";
import { useRef } from "react";
import { ResultSectionHeader } from "./result-section-header";
import { ResultsTable } from "./result-table";

export function ResultsSection({
  queryResult,
  isExecutingQuery,
}: {
  queryResult?: QueryResult;
  isExecutingQuery: boolean;
}) {
  const parentRef = useRef<HTMLDivElement>(null);

  return (
    <ResultsSectionOuter>
      <ResultSectionHeader
        isExecutingQuery={isExecutingQuery}
        queryResult={queryResult}
      />
      <ResultsSectionInner ref={parentRef}>
        {isExecutingQuery && (
          <p className="text-center p-4 text-muted-foreground">
            Executing query...
          </p>
        )}

        {!isExecutingQuery && queryResult?.error && (
          <div className="p-3 m-2 text-red-400 bg-red-900/40 border border-red-700/50 rounded-md">
            <div className="flex items-center font-semibold mb-1">
              <AlertTriangleIcon className="w-5 h-5 mr-2 flex-shrink-0" /> Error
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
              <ResultsTable queryResult={queryResult} parentRef={parentRef} />
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
      </ResultsSectionInner>
    </ResultsSectionOuter>
  );
}

function ResultsSectionOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col border border-border rounded-md bg-sidebar flex-[40%] max-h-[45vh]">
      {children}
    </div>
  );
}

function ResultsSectionInner({
  children,
  ref,
}: {
  children: React.ReactNode;
  ref: React.RefObject<HTMLDivElement | null>;
}) {
  return (
    <div ref={ref} className="flex-grow overflow-auto min-h-0">
      {children}
    </div>
  );
}
