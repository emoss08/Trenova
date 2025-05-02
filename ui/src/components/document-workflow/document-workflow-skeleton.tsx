import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Skeleton } from "@/components/ui/skeleton";
import { getCustomerById } from "@/services/customer";
import { faCheck, faFile, faPlus } from "@fortawesome/pro-solid-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";

export function NoDocumentRequirements({ customerId }: { customerId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ["customer", customerId],
    queryFn: async () => {
      return await getCustomerById(customerId);
    },
    enabled: !!customerId,
  });

  return isLoading ? (
    <div className="flex items-center justify-center">
      <Skeleton className="h-10 w-10 rounded-full" />
    </div>
  ) : (
    <div className="flex items-center justify-center bg-background border border-border rounded-md p-4">
      <div className="flex items-center gap-x-2 text-center justify-center flex-col gap-y-4 group transition duration-500 hover:duration-200">
        <div className="isolate flex justify-center">
          <div className="relative left-2.5 top-1.5 grid size-10 -rotate-6 place-items-center rounded-xl bg-background shadow-lg ring-1 ring-border transition duration-500 group-hover:-translate-x-5 group-hover:-translate-y-0.5 group-hover:-rotate-12 group-hover:duration-200">
            <Icon icon={faFile} className="size-5 text-muted-foreground" />
          </div>
          <div className="relative z-10 grid size-10 place-items-center rounded-xl bg-background shadow-lg ring-1 ring-border transition duration-500 group-hover:-translate-y-0.5 group-hover:duration-200">
            <Icon icon={faPlus} className="size-5 text-muted-foreground" />
          </div>
          <div className="relative right-2.5 top-1.5 grid size-10 rotate-6 place-items-center rounded-xl bg-background shadow-lg ring-1 ring-border transition duration-500 group-hover:-translate-y-0.5 group-hover:translate-x-5 group-hover:rotate-12 group-hover:duration-200">
            <Icon icon={faCheck} className="size-5 text-muted-foreground" />
          </div>
        </div>
        <div className="flex flex-col">
          <p className="text-sm font-medium">No Billing Requirements</p>
          <p className="text-xs text-muted-foreground">
            Customer {data?.name} has no billing requirements
          </p>
        </div>
        <Link
          to={`/billing/configurations/customers?entityId=${customerId}&modalType=edit`}
          state={{
            isNavigatingToModal: true,
          }}
          target="_blank"
          replace
          preventScrollReset
        >
          <Button variant="outline" size="sm">
            Add Billing Requirements
          </Button>
        </Link>
      </div>
    </div>
  );
}

export function CategoryListSkeleton() {
  return (
    <div className="space-y-2">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="p-3 border border-border rounded-md">
          <div className="flex justify-between mb-1">
            <Skeleton className="h-5 w-24" />
            <Skeleton className="h-5 w-16" />
          </div>
          <Skeleton className="h-4 w-full mt-2" />
          <Skeleton className="h-1 w-full mt-2" />
        </div>
      ))}
    </div>
  );
}

export function DocumentListSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {[1, 2, 3, 4, 5, 6].map((i) => (
        <div
          key={i}
          className="border border-border rounded-md overflow-hidden"
        >
          <div className="p-3 border-b border-border">
            <Skeleton className="h-5 w-full" />
          </div>
          <div className="p-3">
            <div className="space-y-2">
              <div className="flex justify-between">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-24" />
              </div>
              <div className="flex justify-between">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-16" />
              </div>
              <div className="flex justify-between">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-24" />
              </div>
            </div>
          </div>
          <div className="p-3 bg-muted flex justify-between">
            <Skeleton className="h-8 w-20" />
            <Skeleton className="h-8 w-20" />
          </div>
        </div>
      ))}
    </div>
  );
}
