/**
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

import { useNotifications } from "@/hooks/useQueries";
import { useUserStore } from "@/stores/AuthStore";
import { faEllipsis } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "../common/icons";
import { Button } from "../ui/button";
import { DropdownMenu, DropdownMenuTrigger } from "../ui/dropdown-menu";
import { Skeleton } from "../ui/skeleton";
import { UserAvatar, UserAvatarMenuContent } from "./user-avatar-menu";
import { cn } from "@/lib/utils";
import React from "react";
import { User } from "@/types/accounts";

function UserAsideAvatar({ user }: { user: User }) {
  return (
    <span className="relative inline-block">
      <UserAvatar user={user} />
      <span className="absolute bottom-1 right-1 block size-2 rounded-full bg-green-600 ring-2 ring-background" />
    </span>
  );
}

export function UserAsideMenu() {
  const user = useUserStore.get("user");
  const { notificationsData, notificationsLoading } = useNotifications(user.id);
  const userHasNotifications =
    (notificationsData && notificationsData?.unreadCount > 0) || false;

  if (notificationsLoading) {
    return <Skeleton className="m-2 h-14" />;
  }

  return (
    <div className="m-2 flex flex-col space-y-1 rounded-lg border border-border p-2">
      <div className="flex items-center space-x-2">
        <UserAsideAvatar user={user} />
        <div className="grow">
          <p className="truncate text-sm font-medium leading-none">
            {user.name || user.username}
          </p>
          <p className="text-xs leading-none text-muted-foreground">
            {user.email}
          </p>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              className={cn(
                "ml-auto size-6",
                userHasNotifications && "bg-green-600/10 hover:bg-green-600/20",
              )}
              size="icon"
              variant="ghost"
            >
              {userHasNotifications && (
                <span className="absolute bottom-11 right-4 flex size-1.5 rounded-full bg-green-600 ring-2 ring-background motion-safe:animate-pulse"></span>
              )}
              <Icon icon={faEllipsis} />
            </Button>
          </DropdownMenuTrigger>
          <UserAvatarMenuContent
            user={user}
            hasNotifications={userHasNotifications}
          />
        </DropdownMenu>
      </div>
    </div>
  );
}
