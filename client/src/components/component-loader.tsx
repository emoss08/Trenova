import { cn } from "@/lib/utils";
import { Suspense } from "react";
import { Spinner } from "./ui/spinner";

export type ComponentLoaderProps = {
  className?: string;
  message?: string;
  description?: string;
};

export function ComponentLoader({
  className,
  message,
  description,
}: ComponentLoaderProps) {
  return (
    <div
      className={cn("flex flex-col items-center justify-center p-2", className)}
    >
      <Spinner className="size-4" />
      <p className="mt-2 text-sm text-foreground">
        {message ?? "Loading data..."}
      </p>
      <p className="mt-2 text-sm text-muted-foreground">
        {description ?? "If this takes too long, please refresh the page."}
      </p>
    </div>
  );
}

export function SuspenseLoader({
  children,
  componentLoaderProps,
}: {
  children: React.ReactNode;
  componentLoaderProps?: ComponentLoaderProps;
}) {
  return (
    <Suspense fallback={<ComponentLoader {...componentLoaderProps} />}>
      {children}
    </Suspense>
  );
}
