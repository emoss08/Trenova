import { Skeleton } from "@/components/ui/skeleton";

export function IntegrationSkeleton() {
  return (
    <div className="overflow-hidden border border-input rounded-md transition-all p-4">
      <div className="flex flex-row items-center justify-between gap-4 pb-2">
        <div className="flex flex-col gap-1">
          <Skeleton className="h-5 w-28" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex-shrink-0 rounded-full flex items-center justify-center p-2 border border-input">
          <Skeleton className="size-4" />
        </div>
      </div>
      <div className="mb-4">
        <Skeleton className="h-[60px] w-full" />
      </div>
      <div className="flex items-center justify-between py-2">
        <Skeleton className="h-8 w-[80px]" />
        <Skeleton className="h-4 w-[100px]" />
      </div>
    </div>
  );
}
