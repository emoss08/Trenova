import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { PageHeaderProps } from "@/components/page-header";
import { cn } from "@trenova/shared/lib/utils";

type BillingWorkspaceLayoutProps = {
  pageHeaderProps: PageHeaderProps;
  toolbar?: React.ReactNode;
  sidebar: React.ReactNode;
  detail: React.ReactNode;
  preview?: React.ReactNode;
  className?: string;
};

export function BillingWorkspaceLayout({
  pageHeaderProps,
  toolbar,
  sidebar,
  detail,
  preview,
  className,
}: BillingWorkspaceLayoutProps) {
  return (
    <PageLayout fill pageHeaderProps={pageHeaderProps} className={className}>
      {toolbar}
      <div
        className={cn(
          "border-border bg-card grid min-h-0 flex-1 gap-0 overflow-hidden rounded-lg border",
          preview ? "grid-cols-[300px_1fr_1fr]" : "grid-cols-[320px_1fr]",
        )}
      >
        <div className="border-border overflow-hidden border-r">{sidebar}</div>
        <div className={cn("overflow-hidden", preview && "border-border border-r")}>{detail}</div>
        {preview ? <div className="overflow-hidden">{preview}</div> : null}
      </div>
    </PageLayout>
  );
}
