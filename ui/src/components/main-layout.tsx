import { ChangePasswordDialog } from "@/app/auth/_components/change-password-dialog";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useAuth } from "@/hooks/use-auth";
import { useQueryInvalidationListener } from "@/hooks/use-invalidate-query";
import { useUser } from "@/stores/user-store";
import { useEffect, useState } from "react";
import { Outlet } from "react-router";
import { AppSidebar } from "./app-sidebar";
import { Header } from "./header";
import { AuthorVerification } from "./ui/author-verification";
import { SidebarInset, SidebarProvider } from "./ui/sidebar";

// function BottomRightPopup() {
//   return (
//     <div className="fixed bottom-6 right-10 z-50">
//       <AIAssistant />
//     </div>
//   );
// }

export function MainLayout() {
  const { isPopout } = usePopoutWindow();
  const user = useUser();
  const [changePasswordDialogOpen, setChangePasswordDialogOpen] =
    useState(false);

  useAuth();
  useQueryInvalidationListener();

  useEffect(() => {
    if (user?.mustChangePassword) {
      setChangePasswordDialogOpen(true);
    }
  }, [user?.mustChangePassword]);

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
          {/* <BottomRightPopup /> */}
        </div>
      </div>
      {changePasswordDialogOpen && (
        <ChangePasswordDialog
          mustChangePassword={user?.mustChangePassword ?? false}
          open={changePasswordDialogOpen}
          onOpenChange={setChangePasswordDialogOpen}
        />
      )}
    </>
  );
}
