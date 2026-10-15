import { Skeleton } from "@trenova/shared/components/ui/skeleton";

export function RoleBundlePickerSkeleton() {
  return (
    <div className="space-y-5" aria-hidden>
      {Array.from({ length: 3 }, (_, section) => (
        <div key={section} className="space-y-2">
          <Skeleton className="h-3 w-20" />
          <div className="grid gap-2 md:grid-cols-2">
            {Array.from({ length: 4 }, (__, option) => (
              <div key={option} className="flex items-start gap-3 rounded-md border p-3">
                <Skeleton className="mt-0.5 size-4 rounded-sm" />
                <div className="flex-1 space-y-1.5">
                  <Skeleton className="h-3.5 w-32" />
                  <Skeleton className="h-3 w-full" />
                  <Skeleton className="h-3 w-2/3" />
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
