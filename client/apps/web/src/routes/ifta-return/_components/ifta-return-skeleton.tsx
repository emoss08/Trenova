import { Card, CardContent, CardHeader } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The worksheet's own shape while the quarter is read: five figures across
 * the top, then the lines table with a group heading and its rows, so the
 * figures land where their outlines were.
 */

const TILE_LABEL_WIDTHS = ["w-16", "w-20", "w-14", "w-12", "w-16"] as const;
const LINE_ROW_WIDTHS = ["w-8", "w-10", "w-8", "w-10", "w-8", "w-12"] as const;
const LINE_COLUMN_COUNT = 7;

export function IftaReturnSkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy aria-label="Loading the return">
      <div className="contents" aria-hidden>
        <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-5">
          {TILE_LABEL_WIDTHS.map((width, index) => (
            <Card key={index} className="gap-0 overflow-hidden rounded-md">
              <CardHeader className="pb-1">
                <Skeleton className={cn("h-3", width)} />
              </CardHeader>
              <CardContent className="flex flex-col gap-1.5">
                <Skeleton className="h-5 w-20" />
                <Skeleton className="h-3 w-24" />
              </CardContent>
            </Card>
          ))}
        </div>

        <Card className="rounded-md">
          <CardHeader>
            <Skeleton className="h-3.5 w-32" />
          </CardHeader>
          <CardContent>
            <div className="overflow-hidden rounded-md border">
              <table aria-label="Return lines" className="w-full">
                <thead className="bg-muted/50">
                  <tr>
                    {Array.from({ length: LINE_COLUMN_COUNT }, (_, index) => (
                      <th key={index} className="px-3 py-2">
                        <Skeleton className="h-3 w-12" />
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {LINE_ROW_WIDTHS.map((width, rowIndex) => (
                    <tr key={rowIndex} className="border-t">
                      <td className="px-3 py-2">
                        <Skeleton className={cn("h-3", width)} />
                      </td>
                      {Array.from({ length: LINE_COLUMN_COUNT - 1 }, (_, index) => (
                        <td key={index} className="px-3 py-2">
                          <Skeleton className="ml-auto h-3 w-10" />
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
