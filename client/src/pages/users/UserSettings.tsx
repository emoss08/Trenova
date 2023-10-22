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

import { cn } from "@/lib/utils";
import { getJobTitleDetails } from "@/services/OrganizationRequestService";
import { getUserDetails } from "@/services/UserRequestService";
import { useUserStore } from "@/stores/AuthStore";
import { UserCircleIcon } from "lucide-react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { ThemeSwitcher } from "./appearance/theme-switcher";

const secondaryNavigation = [
  { name: "Preferences", href: "#", icon: UserCircleIcon, current: true },
  { name: "Security", href: "#", icon: UserCircleIcon, current: false },
  { name: "Notifications", href: "#", icon: UserCircleIcon, current: false },
  { name: "Plan", href: "#", icon: UserCircleIcon, current: false },
  { name: "Billing", href: "#", icon: UserCircleIcon, current: false },
  { name: "Team members", href: "#", icon: UserCircleIcon, current: false },
];

export default function UserSettings() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { userId } = useUserStore.get("user");

  const { data: userDetails, isLoading: isUserDetailsLoading } = useQuery({
    queryKey: ["user", userId],
    queryFn: () => {
      if (!userId) {
        return Promise.resolve(null);
      }
      return getUserDetails(userId);
    },
    onError: () => navigate("/error"),
    initialData: () => queryClient.getQueryData(["user", userId]),
  });

  const { data: jobTitleData, isLoading: isJobTitlesLoading } = useQuery({
    queryKey: ["jobTitle", userDetails?.profile?.jobTitle],
    queryFn: () => {
      if (!userDetails || !userDetails.profile) {
        return Promise.resolve(null);
      }
      return getJobTitleDetails(userDetails?.profile?.jobTitle);
    },
    enabled: !!userDetails,
    initialData: () =>
      queryClient.getQueryData(["jobTitle", userDetails?.profile?.jobTitle]),
  });

  const isLoading = isUserDetailsLoading || isJobTitlesLoading;

  return (
    <div className="max-w-7xl lg:flex lg:gap-x-16">
      <aside className="flex overflow-x-auto border-b border-gray-900/5 py-4 lg:block lg:w-64 lg:flex-none lg:border-0">
        <nav className="flex-none px-4 sm:px-6 lg:px-0">
          <ul
            role="list"
            className="flex gap-x-3 gap-y-1 whitespace-nowrap lg:flex-col"
          >
            {secondaryNavigation.map((item) => (
              <li key={item.name}>
                <a
                  href={item.href}
                  className={cn(
                    item.current
                      ? "bg-foreground/5 text-foreground"
                      : "text-accent-foreground hover:text-foreground hover:bg-accent-foreground/5",
                    "group flex gap-x-3 rounded-md py-2 pl-2 pr-3 text-sm leading-6 font-semibold",
                  )}
                >
                  <item.icon
                    className={cn(
                      item.current
                        ? "text-foreground"
                        : "text-muted-foreground group-hover:text-foreground",
                      "h-6 w-6 shrink-0",
                    )}
                    aria-hidden="true"
                  />
                  {item.name}
                </a>
              </li>
            ))}
          </ul>
        </nav>
      </aside>
      <main className="px-4 sm:px-6 lg:flex-auto lg:px-0">
        <div className="mx-auto max-w-2xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
          <div>
            <h2 className="font-extrabold text-3xl md:text-4xl tracking-tight">
              Preferences
            </h2>
            <p className="text-ld text-muted-foreground">
              This information will be displayed publicly so be careful what you
              share.
            </p>
          </div>
          <div className="flex-1 lg:max-w-2xl">
            <div className="space-y-8">
              <ThemeSwitcher />
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
