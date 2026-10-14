import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Spinner } from "./ui/spinner";

export type ComponentLoaderProps = {
  className?: string;
  message?: string;
  description?: string;
};

export function ComponentLoader({ className, message, description }: ComponentLoaderProps) {
  const t = useT();

  return (
    <div className={cn("flex flex-col items-center justify-center p-2", className)}>
      <Spinner className="size-4" />
      <p className="mt-2 text-sm text-foreground">{message ?? t("Loading data...")}</p>
      <p className="mt-2 text-sm text-muted-foreground">
        {description ?? t("If this takes too long, please refresh the page.")}
      </p>
    </div>
  );
}
