import { cn } from "@/lib/utils";
import { ArrowLeftIcon } from "lucide-react";
import { Link } from "react-router";

type UserBuilderLayoutProps = {
  title: string;
  subtitle?: string;
  leftPanel: React.ReactNode;
  rightPanel: React.ReactNode;
  footer: React.ReactNode;
};

export function UserBuilderLayout({
  title,
  subtitle,
  leftPanel,
  rightPanel,
  footer,
}: UserBuilderLayoutProps) {
  return (
    <div className="flex h-screen flex-col bg-background">
      <header className="shrink-0 border-b bg-card px-6 py-4">
        <div className="flex items-center gap-4">
          <Link
            to="/admin/users"
            className="flex size-8 items-center justify-center rounded-lg border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <ArrowLeftIcon className="size-4" />
          </Link>
          <div>
            <h1 className="text-lg font-semibold">{title}</h1>
            {subtitle && (
              <p className="text-sm text-muted-foreground">{subtitle}</p>
            )}
          </div>
        </div>
      </header>

      <div className="flex min-h-0 flex-1">
        <aside className="w-[420px] shrink-0 overflow-y-auto border-r bg-card p-6">
          {leftPanel}
        </aside>

        <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <div className="flex-1 overflow-y-auto p-6">{rightPanel}</div>
        </main>
      </div>

      <footer className="shrink-0 border-t bg-card">{footer}</footer>
    </div>
  );
}

type UserBuilderSectionProps = {
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
};

export function UserBuilderSection({
  title,
  description,
  children,
  className,
}: UserBuilderSectionProps) {
  return (
    <section className={cn("flex flex-col", className)}>
      <div className="mb-4">
        <h2 className="text-sm font-medium">{title}</h2>
        {description && (
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="flex-1">{children}</div>
    </section>
  );
}
