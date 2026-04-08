import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { PageHeaderProps } from "@/components/page-header";
import { cn } from "@/lib/utils";

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
    <PageLayout pageHeaderProps={pageHeaderProps} className={className}>
      {toolbar}
      <div
        className={cn(
          "mx-4 mt-3 mb-4 grid h-[calc(100vh-220px)] gap-0 overflow-hidden rounded-lg border",
          preview ? "grid-cols-[300px_1fr_1fr]" : "grid-cols-[320px_1fr]",
        )}
      >
        <div className="overflow-hidden border-r">{sidebar}</div>
        <div className={cn("overflow-hidden", preview && "border-r")}>{detail}</div>
        {preview ? <div className="overflow-hidden">{preview}</div> : null}
      </div>
    </PageLayout>
  );
}
