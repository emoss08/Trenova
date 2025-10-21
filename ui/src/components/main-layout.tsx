import { ChangePasswordDialog } from "@/app/auth/_components/change-password-dialog";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useAuth } from "@/hooks/use-auth";
import { useQueryInvalidationListener } from "@/hooks/use-invalidate-query";
import { useUser } from "@/stores/user-store";
import React, { useState } from "react";
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
  const [changePasswordDialogOpen, setChangePasswordDialogOpen] = useState(
    user?.mustChangePassword ?? false,
  );

  useAuth();
  useQueryInvalidationListener();

  return (
    <>
      <AuthorVerification />
      <MainLayoutOuter>
        <MainLayoutInner>
          <SidebarProvider>
            {!isPopout && <AppSidebar />}
            <SidebarInset>
              {!isPopout && <Header />}
              <Outlet />
            </SidebarInset>
          </SidebarProvider>
        </MainLayoutInner>
      </MainLayoutOuter>
      {changePasswordDialogOpen && (
        <ChangePasswordDialog
          mustChangePassword={user?.mustChangePassword ?? false}
          open={changePasswordDialogOpen}
          onOpenChange={(open) => {
            // Only allow closing if user doesn't have to change password
            if (!user?.mustChangePassword || !open) {
              setChangePasswordDialogOpen(open);
            }
          }}
        />
      )}
    </>
  );
}

function MainLayoutOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex min-h-screen flex-col">{children}</div>;
}

function MainLayoutInner({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-1 flex-col">{children}</div>;
}
