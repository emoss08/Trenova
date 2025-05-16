/* eslint-disable @typescript-eslint/no-unused-vars */
import { Kbd } from "@/components/kbd";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { resourceEditorSearchParamsParser } from "@/lib/search-params/resource-editor";
import { PlayIcon, TerminalIcon } from "lucide-react";
import { useQueryStates } from "nuqs";

export function SQLEditorHeader({
  handleExecuteQuery,
  isExecutingQuery,
  sqlQuery,
}: {
  handleExecuteQuery: () => void;
  isExecutingQuery: boolean;
  sqlQuery: string;
}) {
  const [searchParams, _setSearchParams] = useQueryStates(
    resourceEditorSearchParamsParser,
  );

  return (
    <SQLEditorHeaderOuter>
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
    </SQLEditorHeaderOuter>
  );
}

function SQLEditorHeaderOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex justify-between items-center p-2 border-b border-border min-h-[44px]">
      {children}
    </div>
  );
}
