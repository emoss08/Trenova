import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { PageHeaderProps } from "@/components/page-header";
import { cn } from "@trenova/shared/lib/utils";

type BillingWorkspaceLayoutProps = {
  pageHeaderProps: PageHeaderProps;
  toolbar?: React.ReactNode;
  sidebar: React.ReactNode;
  detail: React.ReactNode;
  preview?: React.ReactNode;
  /** A list whose rows carry two lines of detail takes the wider column. */
  sidebarWidth?: "default" | "wide";
  /** Drawn in place of the list and detail when there is nothing to work. */
  empty?: React.ReactNode;
  className?: string;
};

export function BillingWorkspaceLayout({
  pageHeaderProps,
  toolbar,
  sidebar,
  detail,
  preview,
  sidebarWidth = "default",
  empty,
  className,
}: BillingWorkspaceLayoutProps) {
  return (
    <PageLayout fill pageHeaderProps={pageHeaderProps} className={className}>
      {toolbar}
      {empty ? (
        <div className="border-border bg-card flex min-h-0 flex-1 items-center justify-center overflow-auto rounded-lg border px-4 py-12">
          {empty}
        </div>
      ) : (
        <div
          className={cn(
            "border-border bg-card grid min-h-0 flex-1 gap-0 overflow-hidden rounded-lg border",
            preview
              ? "grid-cols-[300px_1fr_1fr]"
              : sidebarWidth === "wide"
                ? "grid-cols-[400px_1fr]"
                : "grid-cols-[320px_1fr]",
          )}
        >
          <div className="border-border overflow-hidden border-r">{sidebar}</div>
          <div className={cn("overflow-hidden", preview && "border-border border-r")}>{detail}</div>
          {preview ? <div className="overflow-hidden">{preview}</div> : null}
        </div>
      )}
    </PageLayout>
  );
}
