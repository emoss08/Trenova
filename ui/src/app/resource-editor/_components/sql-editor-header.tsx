/* eslint-disable @typescript-eslint/no-unused-vars */
import { Kbd } from "@/components/kbd";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { resourceEditorSearchParamsParser } from "@/lib/search-params/resource-editor";
import { faTerminal } from "@fortawesome/pro-solid-svg-icons";
import { PlayIcon } from "lucide-react";
import { useQueryStates } from "nuqs";

export function SQLEditorHeader({
  handleExecuteQuery,
  isExecutingQuery,
  sqlQuery,
}: {
  handleExecuteQuery: (query?: string) => void;
  isExecutingQuery: boolean;
  sqlQuery: string;
}) {
  const [searchParams, _setSearchParams] = useQueryStates(
    resourceEditorSearchParamsParser,
  );

  return (
    <SQLEditorHeaderOuter>
      <h2 className="flex items-center gap-3">
        <Icon icon={faTerminal} className="size-5" />
        <div className="flex items-center text-center gap-1">
          <h5 className="text-lg font-semibold text-foreground">SQL Editor</h5>
          {searchParams.selectedTable && (
            <h6 className="text-xs text-muted-foreground">
              (Context: {searchParams.selectedTable})
            </h6>
          )}
        </div>
      </h2>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              onClick={() => handleExecuteQuery(sqlQuery)}
              size="sm"
              disabled={!sqlQuery.trim() || isExecutingQuery}
              isLoading={isExecutingQuery}
              loadingText="Executing..."
            >
              <PlayIcon className="size-4" />
              Execute Query
            </Button>
          </TooltipTrigger>
          <TooltipContent className="flex items-center gap-2 text-xs">
            <Kbd>Ctrl + Shift + Enter</Kbd>
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
