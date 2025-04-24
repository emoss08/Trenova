import { useAuth } from "@/hooks/use-auth";
import { useQueryInvalidationListener } from "@/hooks/use-invalidate-query";
import { Outlet } from "react-router";
import { AppSidebar } from "./app-sidebar";
import { Header } from "./header";
import { AuthorVerification } from "./ui/author-verification";
import { SidebarInset, SidebarProvider } from "./ui/sidebar";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";

// function BottomRightPopup() {
//   return (
//     <div className="fixed bottom-6 right-10 z-50">
//       <AIAssistant />
//     </div>
//   );
// }

export function MainLayout() {
  const { isPopout } = usePopoutWindow();

  useAuth();
  useQueryInvalidationListener();

  return (
    <>
      <AuthorVerification />
      <div className="flex min-h-screen flex-col">
        <div className="flex flex-1 flex-col">
          <SidebarProvider>
            {!isPopout && <AppSidebar />}
            <SidebarInset>
              {!isPopout && <Header />}
              <Outlet />
            </SidebarInset>
          </SidebarProvider>
        </div>
      </div>
    </>
  );
}
