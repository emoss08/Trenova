import { Badge } from "@/components/ui/badge";
import { CommandItem } from "@/components/ui/command";
import Highlight from "@/components/ui/highlight";
import { cn } from "@/lib/utils";
import { SearchResultItemProps } from "@/types/search";

type ResultItemOuterContainerProps = React.ComponentProps<typeof CommandItem>;

export function ResultItemOuterContainer({
  className,
  ...props
}: ResultItemOuterContainerProps) {
  return (
    <CommandItem
      className={cn(
        "flex items-center gap-2 px-2 py-1.5 text-sm outline-hidden select-none",
        "data-[selected=true]:border-input data-[selected=true]:bg-input/50",
        "h-full rounded-md border border-transparent !px-3 font-medium",
        className,
      )}
      {...props}
    />
  );
}

export function ShipmentResultItem({
  result,
  searchQuery,
}: SearchResultItemProps) {
  return (
    <div className="flex w-full flex-col min-w-0">
      <p className="text-sm font-medium truncate">
        <Highlight highlight={searchQuery} text={result.metadata?.proNumber} />
      </p>

      <div className="flex items-center gap-2 text-2xs text-muted-foreground overflow-hidden">
        {(result.metadata?.customerName || result.metadata?.customerCode) && (
          <p className="truncate max-w-[220px]">
            {result.metadata?.customerName ? (
              <>
                <Highlight
                  highlight={searchQuery}
                  text={result.metadata.customerName}
                />
                {result.metadata?.customerCode && (
                  <span className="opacity-70">
                    {" ("}
                    {result.metadata.customerCode}
                    {")"}
                  </span>
                )}
              </>
            ) : (
              result.metadata?.customerCode
            )}
          </p>
        )}

        {result.metadata?.bol &&
          (result.metadata?.customerName || result.metadata?.customerCode) && (
            <span className="text-border">•</span>
          )}
        {result.metadata?.bol && (
          <p className="truncate max-w-[140px]">
            <span className="opacity-70">BOL</span>
            <span className="ml-1">
              <Highlight highlight={searchQuery} text={result.metadata.bol} />
            </span>
          </p>
        )}

        {result.metadata?.serviceTypeCode &&
          (result.metadata?.customerName ||
            result.metadata?.customerCode ||
            result.metadata?.bol) && <span className="text-border">•</span>}
        {result.metadata?.serviceTypeCode && (
          <Badge
            withDot={false}
            variant="secondary"
            className="max-h-5 px-1.5 py-0 text-[10px]"
          >
            {result.metadata.serviceTypeCode}
          </Badge>
        )}
      </div>
    </div>
  );
}
