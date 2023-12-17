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

import { useNotificaitons } from "@/hooks/useQueries";
import { formatTimestamp } from "@/lib/date";
import { useUserStore } from "@/stores/AuthStore";
import { useHeaderStore } from "@/stores/HeaderStore";
import { Notification, UserNotification } from "@/types/accounts";
import { BellIcon } from "lucide-react";
import nothingFound from "../../assets/images/there-is-nothing-here.png";
import { Badge } from "../ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { ScrollArea } from "../ui/scroll-area";
import { Skeleton } from "../ui/skeleton";

function Notifications({
  notification,
  notificationLoading,
}: {
  notification: UserNotification;
  notificationLoading: boolean;
}) {
  if (notificationLoading) {
    return <Skeleton className="h-80" />;
  }

  if (!notification || notification.unreadList.length === 0) {
    return (
      <div className="flex flex-col justify-content-center items-center h-full w-full mt-10">
        <img src={nothingFound} alt="Nothing Found" className="h-40 w-40" />
        <h3 className="text-2xl font-medium">All Caught up!</h3>
        <p className="text-sm text-muted-foreground">
          You have no unread notifications
        </p>
      </div>
    );
  }

  const notificaitonItems = notification?.unreadList.map(
    (notification: Notification) => {
      const humanReadableTime = formatTimestamp(notification.timestamp);

      return (
        <div
          key={notification.id}
          className="flex flex-col space-y-2 px-4 py-2 border-b border-gray-200"
        >
          <div className="flex items-center justify-between">
            <h4 className="font-medium leading-none">{notification.verb}</h4>
            <Badge className="text-xs">{humanReadableTime}</Badge>
          </div>
          <p className="text-xs text-muted-foreground">
            {notification.description}
          </p>
        </div>
      );
    },
  );

  return <>{notificaitonItems}</>;
}

export function NotificationMenu() {
  const [notificationsMenuOpen, setNotificationMenuOpen] = useHeaderStore.use(
    "notificaitonMenuOpen",
  );
  const { userId } = useUserStore.get("user");
  const { notificationsData, notificationsLoading } = useNotificaitons(
    notificationsMenuOpen,
    userId,
  );

  return (
    <Popover
      open={notificationsMenuOpen}
      onOpenChange={(open) => setNotificationMenuOpen(open)}
    >
      <PopoverTrigger asChild>
        <nav className="relative inline-flex mx-4 cursor-pointer">
          <BellIcon className="h-5 w-5" />
          <span className="sr-only">Notifications</span>
          <span className="flex absolute h-2 w-2 top-0 right-0 -mt-1 -mr-1">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-25"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-red-500"></span>
          </span>
        </nav>
      </PopoverTrigger>
      <PopoverContent
        className="w-80"
        sideOffset={10}
        alignOffset={-40}
        align="end"
      >
        {notificationsLoading ? (
          <div className="flex flex-col space-y-2 px-4 py-2 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <h4 className="font-medium leading-none">
                <Skeleton className="h-4 w-20" />
              </h4>
              <span className="text-xs text-muted-foreground">
                <Skeleton className="h-4 w-20" />
              </span>
            </div>
            <p className="text-sm text-muted-foreground">
              <Skeleton className="h-4 w-20" />
            </p>
          </div>
        ) : (
          <ScrollArea className="h-80 w-full">
            <Notifications
              notification={null}
              notificationLoading={notificationsLoading}
            />
          </ScrollArea>
        )}
      </PopoverContent>
    </Popover>
  );
}
