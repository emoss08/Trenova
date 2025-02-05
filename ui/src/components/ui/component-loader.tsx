import { cn } from "@/lib/utils";
import { faSpinnerThird } from "@fortawesome/pro-regular-svg-icons";
import { Suspense } from "react";
import { Icon } from "./icons";

export function ComponentLoader({
  className,
  message,
  description,
}: {
  className?: string;
  message?: string;
  description?: string;
}) {
  return (
    <div
      className={cn("flex flex-col items-center justify-center p-2", className)}
    >
      <Icon
        icon={faSpinnerThird}
        size="1x"
        className="text-primary motion-safe:animate-spin"
      />
      <p className="text-foreground mt-2 text-sm">
        {message || "Loading data..."}
      </p>
      <p className="text-muted-foreground mt-2 text-sm">
        {description || "If this takes too long, please refresh the page."}
      </p>
    </div>
  );
}

export function SuspenseLoader({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<ComponentLoader />}>{children}</Suspense>;
}
