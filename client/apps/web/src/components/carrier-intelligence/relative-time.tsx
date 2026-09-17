import { useNowSeconds } from "@/hooks/use-now-seconds";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { INTEL_EMPTY_VALUE } from "@/lib/carrier-intelligence";

export type RelativeTimeProps = {
  timestamp: number | null | undefined;
  className?: string;
  fallback?: string;
};

export function RelativeTime({
  timestamp,
  className,
  fallback = INTEL_EMPTY_VALUE,
}: RelativeTimeProps) {
  const now = useNowSeconds();

  if (!timestamp) {
    return <span className={className}>{fallback}</span>;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <time
            dateTime={new Date(timestamp * 1000).toISOString()}
            className={cn("tabular-nums whitespace-nowrap", className)}
          />
        }
      >
        {formatRelativeTime(timestamp - now)}
      </TooltipTrigger>
      <TooltipContent>{formatUnixDateTimeMedium(timestamp)}</TooltipContent>
    </Tooltip>
  );
}
