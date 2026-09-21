import { cn } from "@trenova/shared/lib/utils";

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("ui-shimmer rounded-md", className)}
      {...props}
    />
  );
}

export { Skeleton };
