import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ArrowLeftIcon, Loader2Icon } from "lucide-react";
import { Link } from "react-router";

type RolePageLayoutProps = {
  title: string;
  isSubmitting: boolean;
  submitLabel: string;
  onSubmit: () => void;
  onCancel: () => void;
  permissionCount?: number;
  children: React.ReactNode;
  banner?: React.ReactNode;
};

export function RolePageLayout({
  title,
  isSubmitting,
  submitLabel,
  onSubmit,
  onCancel,
  permissionCount,
  children,
  banner,
}: RolePageLayoutProps) {
  return (
    <div className="flex flex-col overflow-hidden rounded-md border border-border bg-background">
      <header className="sticky top-0 z-10 shrink-0 border-b bg-card/95 px-6 py-3 backdrop-blur-sm">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link
              to="/admin/roles"
              className="flex size-8 items-center justify-center rounded-lg border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <ArrowLeftIcon className="size-4" />
            </Link>
            <h1 className="text-lg font-semibold">{title}</h1>
          </div>

          <div className="flex items-center gap-3">
            {permissionCount !== undefined && permissionCount > 0 && (
              <Badge variant="outline" className="text-xs font-normal">
                {permissionCount} resource{permissionCount !== 1 ? "s" : ""} configured
              </Badge>
            )}
            <Button type="button" variant="outline" size="sm" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="button" size="sm" onClick={onSubmit} disabled={isSubmitting}>
              {isSubmitting && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              {submitLabel}
            </Button>
          </div>
        </div>
      </header>

      {banner}

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-5xl space-y-6 px-6 py-8">{children}</div>
      </div>
    </div>
  );
}

type RoleBuilderSectionProps = {
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
};

export function RoleBuilderSection({
  title,
  description,
  children,
  className,
}: RoleBuilderSectionProps) {
  return (
    <section className={cn("flex flex-col", className)}>
      <div className="mb-4">
        <h2 className="text-sm font-medium">{title}</h2>
        {description && <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>}
      </div>
      <div className="flex-1">{children}</div>
    </section>
  );
}
