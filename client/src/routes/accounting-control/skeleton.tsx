import { InputFieldSkeleton } from "@/components/fields/input-field";
import { SwitchFieldSkeleton } from "@/components/fields/switch-field";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function PageSkeleton() {
  return (
    <div className="flex flex-col gap-y-4">
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-6 w-[150px]" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-10 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-6 w-[150px]" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-10 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-6 w-[150px]" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-10 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
          <SwitchFieldSkeleton />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-6 w-[150px]" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-10 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <SwitchFieldSkeleton />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-6 w-[150px]" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-10 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <InputFieldSkeleton />
          <SwitchFieldSkeleton />
          <InputFieldSkeleton />
          <SwitchFieldSkeleton />
        </CardContent>
      </Card>
    </div>
  );
}
