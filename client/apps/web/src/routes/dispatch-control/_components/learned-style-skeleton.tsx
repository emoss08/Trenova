import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = 3;

export function LearnedStyleSkeleton() {
  const t = useT();

  return (
    <Card aria-busy aria-label={t("Loading the learned dispatch style")}>
      <div aria-hidden className="contents">
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-5 w-56" />
          </CardTitle>
          <CardDescription className="flex flex-col gap-1.5">
            <Skeleton className="h-3.5 w-full" />
            <Skeleton className="h-3.5 w-3/4" />
          </CardDescription>
        </CardHeader>
        <CardContent className="flex max-w-prose flex-col gap-3">
          <Skeleton className="h-4 w-2/3" />
          {Array.from({ length: ROWS }, (_, index) => (
            <div key={index} className="flex items-center justify-between gap-4">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-24" />
            </div>
          ))}
          <Skeleton className="h-8 w-28" />
        </CardContent>
      </div>
    </Card>
  );
}
