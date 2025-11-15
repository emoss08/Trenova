import { Skeleton } from "@/components/ui/skeleton";
import { TextShimmer } from "@/components/ui/text-shimmer";

export function AvailableNodesSkeleton() {
  return (
    <div className="flex flex-col gap-2 p-4">
      {Array.from({ length: 10 }, (_, index) => (
        <Skeleton key={index} className="h-10 w-full" />
      ))}
    </div>
  );
}

export function ReactFlowSkeleton() {
  return (
    <div className="flex h-[80vh] w-full flex-col items-center justify-center rounded-md border border-border">
      <div className="relative size-full">
        <Skeleton className="size-full" />
        <span className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
          <TextShimmer duration={1.5}>Loading workflow builder...</TextShimmer>
        </span>
      </div>
    </div>
  );
}

export function WorkflowOptionsSkeleton() {
  return (
    <div className="flex h-[80vh] min-w-[370px] flex-col items-center justify-center rounded-md border border-border">
      <div className="relative size-full">
        <Skeleton className="size-full" />
        <span className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
          <TextShimmer duration={1.5}>Loading workflow options...</TextShimmer>
        </span>
      </div>
    </div>
  );
}
