import { cn } from "@/lib/utils";
import { Card, CardContent } from "./ui/card";
import { Skeleton } from "./ui/skeleton";

export function MetricSkeleton({
  length = 4,
  className,
  cardClassName,
}: {
  length?: number;
  className?: string;
  cardClassName?: string;
}) {
  return (
    <div className={cn("mb-3 grid grid-cols-1 gap-2.5 sm:grid-cols-2 xl:grid-cols-4", className)}>
      {Array.from({ length }).map((_, index) => (
        <Card key={index} className={cn("gap-0 overflow-hidden", cardClassName)}>
          <CardContent className="space-y-4 p-2">
            <Skeleton className="h-3 w-28" />
            <Skeleton className="h-5 w-20" />
            <Skeleton className="h-3 w-24" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
