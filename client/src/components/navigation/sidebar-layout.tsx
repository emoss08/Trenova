import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";
import { useNavigationStore } from "@/stores/navigation-store";
import { useUpdateStore } from "@/stores/update-store";
import { useEffect } from "react";
import { RouteCommandPalette } from "../command-palette/route-command-palette";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";
import { Header } from "../header";
import { PageHeader, type PageHeaderProps } from "../page-header";
import { AppSidebar } from "./app-sidebar";

interface SidebarLayoutProps {
  children: React.ReactNode;
}

export function SidebarLayout({ children }: SidebarLayoutProps) {
  const { sidebarOpen, setSidebarOpen } = useNavigationStore();
  const fetchStatus = useUpdateStore((state) => state.fetchStatus);

  useEffect(() => {
    void fetchStatus();
  }, [fetchStatus]);

  return (
    <SidebarProvider defaultOpen={sidebarOpen} open={sidebarOpen} onOpenChange={setSidebarOpen}>
      <RouteCommandPalette />
      <KeyboardShortcutsDialog />
      <AppSidebar variant="sidebar" />
      <SidebarInset>
        <Header />
        <main className="flex-1 overflow-auto">{children}</main>
      </SidebarInset>
    </SidebarProvider>
  );
}

export function PageLayout({
  pageHeaderProps,
  children,
  className,
}: {
  pageHeaderProps: PageHeaderProps;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <>
      <PageHeader {...pageHeaderProps} />
      <div className={cn("flex flex-col gap-y-4 p-4", className)}>{children}</div>
    </>
  );
}

export function AdminPageLayout({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("flex flex-col gap-y-4 px-6", className)}>{children}</div>;
}
