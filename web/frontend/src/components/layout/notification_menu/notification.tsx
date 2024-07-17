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

import { Badge } from "@/components/ui/badge";
import { ComponentLoader } from "@/components/ui/component-loader";
import { formatTimestamp } from "@/lib/date";
import { truncateText } from "@/lib/utils";
import { Notification, UserNotification } from "@/types/accounts";
import { InboxIcon } from "lucide-react";
import { Link } from "react-router-dom";

export function Notifications({
  notification,
  notificationLoading,
}: {
  notification: UserNotification;
  notificationLoading: boolean;
}) {
  if (notificationLoading) {
    return <ComponentLoader className="h-80" />;
  }

  if (!notification || notification.unreadList === null) {
    return (
      <div className="flex h-80 w-full items-center justify-center p-4">
        <div className="flex flex-col items-center justify-center gap-y-3">
          <div className="bg-accent flex size-10 items-center justify-center rounded-full">
            <InboxIcon className="text-muted-foreground" />
          </div>
          <p className="text-muted-foreground select-none text-center text-sm">
            No new notifications
          </p>
        </div>
      </div>
    );
  }

  const notificationItems = notification?.unreadList?.map(
    (notification: Notification) => {
      const humanReadableTime = formatTimestamp(notification.createdAt);

      return (
        <Link to={notification.actionUrl} key={notification.id}>
          <div
            key={notification.id}
            className="border-accent hover:bg-accent/80 group flex cursor-pointer flex-col space-y-2 rounded-md border-b px-4 py-2"
          >
            <div className="flex items-center justify-between">
              <p className="text-sm font-semibold leading-none">
                {truncateText(notification.title, 25)}
              </p>
              <Badge
                withDot={false}
                className="bg-accent text-accent-foreground group-hover:bg-accent-foreground group-hover:text-accent select-none p-0.5 text-xs"
              >
                {humanReadableTime}
              </Badge>
            </div>
            <p className="text-muted-foreground text-xs">
              {notification.description}
            </p>
          </div>
        </Link>
      );
    },
  );

  return <>{notificationItems}</>;
}
