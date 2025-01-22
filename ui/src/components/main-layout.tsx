import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useAuth } from "@/hooks/use-auth";
import { useQueryInvalidationListener } from "@/hooks/use-invalidate-query";
import { Outlet } from "react-router";
import { AppSidebar } from "./app-sidebar";
import { Header } from "./header";
import { SidebarInset, SidebarProvider } from "./ui/sidebar";

export function MainLayout() {
  const { isPopout } = usePopoutWindow();

  useAuth();
  useQueryInvalidationListener();

  return (
    <div className="flex min-h-screen flex-col">
      <div className="flex flex-1 flex-col">
        <SidebarProvider>
          {!isPopout && <AppSidebar />}
          <SidebarInset className="pt-1">
            {!isPopout && <Header />}
            <main className="flex flex-1 flex-col px-4">
              <Outlet />
            </main>
          </SidebarInset>
        </SidebarProvider>
      </div>
    </div>
  );
}
