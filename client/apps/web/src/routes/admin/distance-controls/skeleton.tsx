import { InputFieldSkeleton } from "@/components/fields/input-field";
import { SwitchFieldSkeleton } from "@/components/fields/switch-field";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";

export function PageSkeleton() {
  return (
    <div className="flex flex-col gap-y-4 p-4">
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-6 w-50" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-10 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
          <InputFieldSkeleton />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-6 w-62.5" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-10 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4">
          {Array.from({ length: 8 }).map((_, index) => (
            <InputFieldSkeleton key={index} />
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
