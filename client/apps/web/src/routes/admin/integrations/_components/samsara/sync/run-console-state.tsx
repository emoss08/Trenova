import { Skeleton } from "@/components/ui/skeleton";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { cn } from "@/lib/utils";

export function RunConsoleLoadingState({
  description,
  className,
}: {
  description: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex h-76.75 w-full flex-col items-center justify-center rounded-md border border-border",
        className,
      )}
    >
      <div className="relative size-full">
        <Skeleton className="size-full" />
        <span className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
          <TextShimmer duration={1.5}>{description}</TextShimmer>
        </span>
      </div>
    </div>
  );
}
