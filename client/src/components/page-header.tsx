import { cn } from "@/lib/utils";
import { Metadata } from "./metadata";
import { Skeleton } from "./ui/skeleton";

export type PageHeaderProps = {
  title: string;
  description: string;
  context?: React.ReactNode;
  actions?: React.ReactNode;
  includeMetadata?: boolean;
  className?: string;
  // Simple flag to include inner padding for the title and description (Mainly used for admin pages)
  includeInnerPadding?: boolean;
};

export function PageHeader({
  title,
  description,
  context,
  actions,
  includeMetadata = true,
  includeInnerPadding = false,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn("border-b border-border p-4", className)}>
      <div
        className={cn(
          "flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between",
          includeInnerPadding ? "px-4" : "",
        )}
      >
        <div className="flex min-w-0 flex-col items-start leading-none">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
            {context}
          </div>
          <p className="text-muted-foreground">{description}</p>
        </div>
        {actions ? (
          <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>
        ) : null}
      </div>
      {includeMetadata && <Metadata title={title} description={description} />}
    </div>
  );
}

export function PageHeaderSkeleton() {
  return (
    <div className="flex flex-col items-start gap-2 leading-none">
      <Skeleton className="h-10 w-62.5" />
      <Skeleton className="h-4 w-md" />
    </div>
  );
}
