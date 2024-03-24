/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { CookieConsent } from "@/components/layout/cookie-consent";
import { SiteSearch, SiteSearchInput } from "@/components/layout/site-search";
import OrganizationSwitcher from "@/components/layout/team-switcher";
import { RainbowTopBar } from "@/components/layout/topbar";
import { UserAvatarMenu } from "@/components/layout/user-avatar-menu";
import { useQueryInvalidationListener } from "@/hooks/useBroadcast";
import { useUserStore } from "@/stores/AuthStore";
import React from "react";
import { Toaster } from "react-hot-toast";
import { useLocation } from "react-router-dom";
import { AsideMenuSheet } from "./aside-menu";
import { Breadcrumb } from "./breadcrumb";
import { OrganizationLogo } from "./logo";

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
      <Toaster position="bottom-right" />
      {!hideHeader && (
        <header className="border-border/40 bg-background/95 supports-[backdrop-filter]:bg-background/60 sticky top-0 z-50 w-full border-b backdrop-blur">
          <RainbowTopBar />
          <div className="flex h-14 w-full items-center justify-between px-4">
            <div className="flex items-center gap-x-4">
              <OrganizationLogo />
              <AsideMenuSheet />
              <div className="border-muted-foreground/40 h-7 border-l" />
              <OrganizationSwitcher />
            </div>
            {/* <NavMenu /> */}
            <div className="flex items-center gap-x-4">
              <SiteSearchInput />
              {/* <NotificationMenu /> */}
              <div className="border-muted-foreground/40 h-7 border-l" />
              {user && <UserAvatarMenu user={user} />}
            </div>
          </div>
        </header>
      )}
      <main className="flex-1 px-6 sm:px-6 md:px-12 xl:px-20">
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
      <Toaster position="bottom-right" />
      <header className="sticky top-0 z-50 w-full shrink-0 border-b">
        <RainbowTopBar />
      </header>
      <div className="h-screen">{children}</div>
      <CookieConsent />
    </div>
  );
}
