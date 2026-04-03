import { cn } from "@/lib/utils";
import { Metadata } from "./metadata";
import { Skeleton } from "./ui/skeleton";

export type PageHeaderProps = {
  title: string;
  description: string;
  includeMetadata?: boolean;
  className?: string;
  // Simple flag to include inner padding for the title and description (Mainly used for admin pages)
  includeInnerPadding?: boolean;
};

export function PageHeader({
  title,
  description,
  includeMetadata = true,
  includeInnerPadding = false,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn("border-b border-border p-4", className)}>
      <div
        className={cn("flex flex-col items-start leading-none", includeInnerPadding ? "px-4" : "")}
      >
        <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
        <p className="text-muted-foreground">{description}</p>
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
