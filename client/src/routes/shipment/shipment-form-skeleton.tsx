import { FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

export function ShipmentFormSkeleton() {
  return (
    <div className="flex flex-col gap-6">
      <SectionSkeleton />
      <FormGroupSkeleton amount={2} />
      <FormGroupSkeleton amount={2} />
      <Separator />
      <SectionSkeleton />
      <FormGroupSkeleton amount={2} cols={1} />
      <FormGroupSkeleton amount={2} />
      <Separator />
      <SectionSkeleton />
      <FormGroupSkeleton amount={2} />
      <FormGroupSkeleton amount={2} />
    </div>
  );
}

function SectionSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <Skeleton className="h-5 w-[150px]" />
          <Skeleton className="h-3 w-[330px]" />
        </div>
      </div>
    </div>
  );
}

function FormGroupSkeleton({ cols = 2, amount }: { cols?: 1 | 2 | 3 | 4; amount: number }) {
  return (
    <FormGroup cols={cols}>
      {Array.from({ length: amount }).map((_, i) => (
        <div key={i} className="flex flex-col gap-1">
          <Skeleton className="h-4 w-[100px]" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-2 w-[200px]" />
        </div>
      ))}
    </FormGroup>
  );
}
