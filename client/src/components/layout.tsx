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

import { RainbowTopBar } from "@/components/topbar";
import { getUserDetails } from "@/services/UserRequestService";
import { useUserStore } from "@/stores/AuthStore";
import React from "react";
import { useQuery, useQueryClient } from "react-query";
import { Breadcrumb } from "./common/BreadCrumbs";
import { Footer } from "./footer";
import { NavMenu } from "./navbar";
import { SiteSearch } from "./site-search";
import { Skeleton } from "./ui/skeleton";
import { Toaster } from "./ui/toaster";
import { UserAvatarMenu } from "./user-avatar-menu";

// Type Definitions

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
  const queryClient = useQueryClient();

  // Fetch user data based on userId
  const { data: userData, isLoading: isUserDataLoading } = useQuery({
    queryKey: ["user", userId],
    queryFn: () => (userId ? getUserDetails(userId) : Promise.resolve(null)),
    initialData: () => queryClient.getQueryData(["user", userId]),
    staleTime: Infinity,
  });

  return (
    <div className="relative flex flex-col h-screen">
      <header className="bg-background sticky top-0 z-50 w-full border-b">
        <RainbowTopBar />
        <div className="container flex h-14 items-center">
          <NavMenu />
          <SiteSearch />
          {isUserDataLoading ? (
            <div className="flex flex-1 items-center justify-between space-x-2 md:justify-end">
              <Skeleton className="h-10 w-10 rounded-full" />
            </div>
          ) : (
            userData && <UserAvatarMenu user={userData} />
          )}
        </div>
      </header>

      {/* Main Content Area */}
      <div className="flex-1">
        <div className="container relative">
          <Breadcrumb />
          {children}
          <Toaster />
        </div>
      </div>

      {/* Footer */}
      <Footer />
    </div>
  );
}

/**
 * UnprotectedLayout component for pages that don't require authentication.
 */
export function UnprotectedLayout({ children }: LayoutProps): JSX.Element {
  return (
    <div className="h-screen flex flex-col overflow-hidden">
      <RainbowTopBar />
      <div className="h-screen">{children}</div>
      <Toaster />
    </div>
  );
}
