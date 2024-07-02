import { CookieConsent } from "@/components/layout/cookie-consent";
import { SearchButton, SiteSearch } from "@/components/layout/site-search";
import { RainbowTopBar } from "@/components/layout/topbar";
import { UserAvatarMenu } from "@/components/layout/user-avatar-menu";
import { useQueryInvalidationListener } from "@/hooks/useBroadcast";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { useNotificationListener } from "@/hooks/useNotification";
import { useUserStore } from "@/stores/AuthStore";
import React from "react";
import { useLocation } from "react-router-dom";
import MainAsideMenu, { AsideMenuDialog } from "./aside-menu";
import { Breadcrumb } from "./breadcrumb";
import { NotificationMenu } from "./notification_menu/notification-menu";

/**
 * Layout component that provides a common structure for protected pages.
 * Contains navigation, header, and footer.
 */
export function Layout({ children }: { children: React.ReactNode }) {
  const [user] = useUserStore.use("user");
  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);
  const hideAsideMenu = queryParams.get("hideAside") === "true";
  const isDesktop = useMediaQuery("(min-width: 1024px)");

  // Listen for query invalidation events
  useQueryInvalidationListener();

  // Listen for notifications
  useNotificationListener();

  return (
    <div className="bg-background relative flex min-h-screen flex-col" id="app">
      <RainbowTopBar />
      <div className="flex flex-1 overflow-hidden">
        {hideAsideMenu ? null : <MainAsideMenu />}
        <div className="flex flex-1 flex-col overflow-hidden">
          {!isDesktop ? (
            <header className="border-border/40 bg-background/95 flex flex-none border-b xl:hidden">
              <div className="flex h-14 w-full items-center justify-between px-4">
                <div className="flex items-center gap-x-4">
                  <AsideMenuDialog />
                </div>
                <div className="ml-auto flex items-center gap-x-4">
                  <SearchButton />
                  <NotificationMenu />
                  <div className="border-muted-foreground/40 h-7 border-l" />
                  {user && <UserAvatarMenu user={user} />}
                </div>
              </div>
            </header>
          ) : null}
          <main className="flex-1 overflow-auto px-6">
            <Breadcrumb />
            <SiteSearch />
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}

/**
 * UnprotectedLayout component for pages that don't require authentication.
 */
export function UnprotectedLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <header className="sticky top-0 z-50 w-full shrink-0 border-b">
        <RainbowTopBar />
      </header>
      <div className="h-screen">{children}</div>
      <CookieConsent />
    </div>
  );
}
