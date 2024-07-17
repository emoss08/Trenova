/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { useNotifications } from "@/hooks/useQueries";
import { useUserStore } from "@/stores/AuthStore";
import { faEllipsis } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button } from "../ui/button";
import { DropdownMenu, DropdownMenuTrigger } from "../ui/dropdown-menu";
import { Skeleton } from "../ui/skeleton";
import { UserAvatar, UserAvatarMenuContent } from "./user-avatar-menu";

export function UserAsideMenu() {
  const user = useUserStore.get("user");
  const { notificationsData, notificationsLoading } = useNotifications(user.id);
  const userHasNotifications =
    (notificationsData && notificationsData?.unreadCount > 0) || false;

  if (notificationsLoading) {
    return <Skeleton className="m-2 h-14" />;
  }

  return (
    <div className="border-border mt-2 flex flex-col space-y-1 border-t px-4 py-2">
      <div className="flex items-center space-x-2 pt-2">
        <UserAvatar user={user} />
        <div className="grow">
          <p className="truncate text-sm font-medium leading-none">
            {user.name || user.username}
          </p>
          <p className="text-muted-foreground text-xs leading-none">
            {user.email}
          </p>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button className="ml-auto size-6" size="icon" variant="ghost">
              {userHasNotifications && (
                <span className="ring-background absolute bottom-9 right-4 flex size-1.5 rounded-full bg-green-600 ring-2 motion-safe:animate-pulse"></span>
              )}
              <FontAwesomeIcon icon={faEllipsis} />
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
