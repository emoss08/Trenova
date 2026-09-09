import { Card, CardContent, CardHeader } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The page's own shape while the summary is read: four figure cards, the
 * exception aging table beside the work items, and the two links out. Every
 * measurement mirrors the loaded layout so the figures land where their
 * outlines were.
 */

const FIGURE_LABEL_WIDTHS = ["w-14", "w-12", "w-16", "w-16"] as const;
const AGING_ROW_WIDTHS = ["w-12", "w-14", "w-14", "w-12"] as const;
const WORK_ROW_WIDTHS = ["w-12", "w-14", "w-16"] as const;

function FigureCardSkeleton({ labelWidth, rate }: { labelWidth: string; rate?: boolean }) {
  return (
    <Card className="gap-0 overflow-hidden rounded-md">
      <CardHeader className="pb-1">
        <Skeleton className={cn("h-3", labelWidth)} />
      </CardHeader>
      <CardContent className="flex flex-col gap-1.5">
        <Skeleton className={cn(rate ? "h-9 w-16" : "h-8 w-10")} />
        {rate ? null : <Skeleton className="h-3 w-20" />}
      </CardContent>
    </Card>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology.
 */
export function ReconciliationSummarySkeleton() {
  return (
    <div className="space-y-6" aria-busy aria-label="Loading the summary">
      <div className="contents" aria-hidden>
        <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
          {FIGURE_LABEL_WIDTHS.map((width, index) => (
            <FigureCardSkeleton key={index} labelWidth={width} rate={index === 3} />
          ))}
        </div>

        <div className="grid gap-4 xl:grid-cols-2">
          <Card className="rounded-md">
            <CardHeader>
              <Skeleton className="h-3.5 w-28" />
            </CardHeader>
            <CardContent>
              <div className="overflow-hidden rounded-md border">
                <table aria-label="Exception aging" className="w-full text-sm">
                  <thead className="bg-muted/50">
                    <tr>
                      <th className="px-3 py-2 text-left">
                        <Skeleton className="h-3 w-12" />
                      </th>
                      <th className="px-3 py-2">
                        <Skeleton className="ml-auto h-3 w-10" />
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {AGING_ROW_WIDTHS.map((width, index) => (
                      <tr key={index} className="border-t">
                        <td className="px-3 py-2">
                          <Skeleton className={cn("h-3", width)} />
                        </td>
                        <td className="px-3 py-2">
                          <Skeleton className="ml-auto h-3 w-6" />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>

          <Card className="rounded-md">
            <CardHeader>
              <Skeleton className="h-3.5 w-20" />
            </CardHeader>
            <CardContent>
              <ul aria-label="Work items" className="space-y-3">
                {WORK_ROW_WIDTHS.map((width, index) => (
                  <li key={index} className="flex h-5 items-center justify-between">
                    <Skeleton className={cn("h-3", width)} />
                    <Skeleton className="h-3 w-6" />
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>

        <div className="flex items-center gap-3">
          <Skeleton className="h-8 w-32 rounded-md" />
          <Skeleton className="h-8 w-28 rounded-md" />
        </div>
      </div>
    </div>
  );
}
