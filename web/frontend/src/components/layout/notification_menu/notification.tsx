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
