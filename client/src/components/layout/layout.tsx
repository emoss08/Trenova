/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { NavMenu } from "@/components/layout/navbar";
import { NotificationMenu } from "@/components/layout/notification_menu/notification-menu";
import { SiteSearch } from "@/components/layout/site-search";
import { RainbowTopBar } from "@/components/layout/topbar";
import { UserAvatarMenu } from "@/components/layout/user-avatar-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { Toaster } from "react-hot-toast";

import { useQueryInvalidationListener } from "@/hooks/useBroadcast";
import { useUser } from "@/hooks/useQueries";
import { ENVIRONMENT } from "@/lib/constants";
import { useUserStore } from "@/stores/AuthStore";
import { User } from "@/types/accounts";
import React from "react";
import { useLocation } from "react-router-dom";
import { AppGridMenu } from "./app-grid";
import { Breadcrumb } from "./breadcrumb";
import { Footer } from "./footer";
import { Logo } from "./logo";

function DevHeader() {
  // Simple header that puts div in the middle on a red background
  return (
    <header className="flex items-center justify-center h-5 w-full bg-indigo-700">
      <div className="text-white">
        You're currently running Monta in development mode.
      </div>
    </header>
  );
}

/**
 * LayoutProps defines the props for the Layout components.
 */
type LayoutProps = {
  children: React.ReactNode;
};

/**
 * Layout component that provides a common structure for protected pages.
 * Contains navigation, header, and footer.
 */
export function Layout({ children }: LayoutProps) {
  const { userId } = useUserStore.get("user");
  const { data: userData, isLoading: isUserDataLoading } = useUser(userId);
  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);
  const hideHeader = queryParams.get("hideHeader") === "true";

  useQueryInvalidationListener();

  return (
    // Use min-h-screen instead of h-screen to prevent overflow from causing double scrollbars
    <div className="flex flex-col min-h-screen" id="app">
      <Toaster position="bottom-right" />
      {!hideHeader && (
        <header className="shrink-0 sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <RainbowTopBar />
          {ENVIRONMENT === "development" && <DevHeader />}
          <div className="flex items-center justify-between h-14 w-full px-4">
            {/* Logo on the left */}
            <Logo />
            {/* Centered Navigation Menu (visible on md screens and up) */}
            <div className="hidden md:flex flex-1 justify-center">
              <NavMenu />
            </div>
            {/* User Avatar and Notification Bell on the right */}
            <div className="flex items-center">
              <AppGridMenu />
              <NotificationMenu />
              <div className="border-l border-muted-foreground/40 h-7 mr-2 pl-2" />
              {isUserDataLoading ? (
                <div className="flex items-center space-x-2">
                  <Skeleton className="h-10 w-10 rounded-full" />
                </div>
              ) : (
                (userData as User) && <UserAvatarMenu user={userData as User} />
              )}
            </div>
          </div>
        </header>
      )}

      {/* Main content should allow for y-axis overflow only */}
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto px-6 sm:px-6 md:px-12 xl:px-20">
          <Breadcrumb />
          <SiteSearch />
          {children}
        </div>
      </main>

      {/* Footer */}
      <footer>
        <Footer />
      </footer>
    </div>
  );
}

/**
 * UnprotectedLayout component for pages that don't require authentication.
 */
export function UnprotectedLayout({ children }: LayoutProps) {
  return (
    <div className="h-screen flex flex-col overflow-hidden">
      <Toaster position="bottom-right" />
      <header className="shrink-0 sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <RainbowTopBar />
        {ENVIRONMENT === "development" && <DevHeader />}
      </header>
      <div className="h-screen">{children}</div>
    </div>
  );
}
