import { CookieConsent } from "@/components/layout/cookie-consent";
import {
  SearchButton,
  SiteSearch,
  SiteSearchInput,
} from "@/components/layout/site-search";
import OrganizationSwitcher from "@/components/layout/team-switcher";
import { RainbowTopBar } from "@/components/layout/topbar";
import { UserAvatarMenu } from "@/components/layout/user-avatar-menu";
import { useQueryInvalidationListener } from "@/hooks/useBroadcast";
import { useUserStore } from "@/stores/AuthStore";
import React from "react";
import { useLocation } from "react-router-dom";
import { Breadcrumb } from "./breadcrumb";
import { OrganizationLogo } from "./logo";
import { NavMenu } from "./navbar";
import { NotificationMenu } from "./notification_menu/notification-menu";

/**
 * Layout component that provides a common structure for protected pages.
 * Contains navigation, header, and footer.
 */
export function Layout({ children }: { children: React.ReactNode }) {
  const [user] = useUserStore.use("user");
  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);
  const hideHeader = queryParams.get("hideHeader") === "true";
  useQueryInvalidationListener();

  return (
    <div className="bg-background relative flex min-h-screen flex-col" id="app">
      {!hideHeader && (
        <header className="border-border/40 bg-background/95 supports-[backdrop-filter]:bg-background/60 sticky top-0 z-50 w-full border-b backdrop-blur">
          <RainbowTopBar />
          <div className="flex h-14 w-full items-center justify-between px-4">
            <div className="flex items-center gap-x-4">
              <OrganizationLogo />
              <div className="border-muted-foreground/40 h-7 border-l" />
              <OrganizationSwitcher />
            </div>
            <NavMenu />
            <div className="flex items-center gap-x-4">
              <SiteSearchInput />
              <SearchButton />
              <NotificationMenu />
              <div className="border-muted-foreground/40 h-7 border-l" />
              {user && <UserAvatarMenu user={user} />}
            </div>
          </div>
        </header>
      )}
      <main className="max-w-screen-3xl mx-auto w-full flex-1 px-6 sm:px-6 md:px-12 xl:px-20">
        <Breadcrumb />
        <SiteSearch />
        {children}
      </main>
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
