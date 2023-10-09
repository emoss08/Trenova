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

import { NavMenu } from "@/components/navbar";
import RainbowTopBar from "@/components/topbar";
import { getUserDetails } from "@/services/UserRequestService";
import { useUserStore } from "@/stores/AuthStore";
import React from "react";
import { useQuery, useQueryClient } from "react-query";
import { UserAvatarMenu } from "./user-avatar-menu";

type LayoutProps = {
  children: React.ReactNode;
};

export function Layout({ children }: LayoutProps): React.ReactElement {
  const { userId } = useUserStore.get("user");
  const queryClient = useQueryClient();

  const { data: userData, isLoading: isUserDataLoading } = useQuery({
    queryKey: ["user", userId],
    queryFn: () => {
      if (!userId) {
        return Promise.resolve(null);
      }
      return getUserDetails(userId);
    },
    initialData: () => queryClient.getQueryData(["user", userId]),
    staleTime: Infinity,
  });

  return (
    <div className="relative flex min-h-screen flex-col">
      <header className="supports-backdrop-blur:bg-background/60 sticky top-0 z-50 w-full border-b background/95 backdrop-blur">
        <RainbowTopBar />
        <div className="container flex h-14 items-center">
          <NavMenu />
          {isUserDataLoading ? (
            <div className="flex flex-1 items-center justify-between space-x-2 md:justify-end">
              <div className="animate-pulse flex space-x-4">
                <div className="rounded-full bg-black dark:bg-white opacity-10 h-10 w-10"></div>
              </div>
            </div>
          ) : (
            userData && <UserAvatarMenu user={userData} />
          )}
        </div>
      </header>
      <div className="flex-1 overflow-auto">
        <div className="flex flex-col sm:flex-row sm:justify-center sm:items-center p-8">
          {children}
        </div>
      </div>
    </div>
  );
}

export function UnprotectedLayout({
  children,
}: LayoutProps): React.ReactElement {
  return (
    <div className="h-screen flex flex-col overflow-hidden">
      <RainbowTopBar />
      <div className="h-screen">{children}</div>
    </div>
  );
}
