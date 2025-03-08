import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "./tooltip";

export function SensitiveBadge() {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="ml-2 text-xs px-1.5 py-0.5 bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400 rounded-sm font-medium select-none">
            Sensitive
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>Field is sensitive and has been masked.</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
