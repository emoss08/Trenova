import { useAuth } from "@/hooks/use-auth";
import { Outlet } from "react-router";
import { AppSidebar } from "./app-sidebar";
import { Header } from "./header";
import { SidebarInset, SidebarProvider } from "./ui/sidebar";

export function MainLayout() {
  useAuth();

  return (
    <div className="flex min-h-screen flex-col">
      <div className="flex flex-1 flex-col">
        <SidebarProvider>
          <AppSidebar />
          <SidebarInset className="pt-1">
            <Header />
            <main className="flex flex-1 flex-col px-4">
              <Outlet />
            </main>
          </SidebarInset>
        </SidebarProvider>
      </div>
    </div>
  );
}
