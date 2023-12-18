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

import NotificationSoundMp3 from "@/assets/audio/notification.mp3";
import NotificationSound from "@/assets/audio/notification.webm";
import { useNotificaitons } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import { ENABLE_WEBSOCKETS, WEB_SOCKET_URL } from "@/lib/constants";
import { createWebsocketManager } from "@/lib/websockets";
import { useUserStore } from "@/stores/AuthStore";
import { useHeaderStore } from "@/stores/HeaderStore";
import { UserNotification } from "@/types/accounts";
import { useQueryClient } from "@tanstack/react-query";
import { Howl } from "howler";
import { BellIcon, ChevronRight } from "lucide-react";
import React, { useState } from "react";
import toast from "react-hot-toast";
import { Notifications } from "@/components/layout/notification_menu/notification";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";

const sound = new Howl({
  src: [NotificationSound, NotificationSoundMp3],
  volume: 0.5,
});

const webSocketManager = createWebsocketManager();

let intervalId: string | number | NodeJS.Timeout | undefined;

const reconnect = () => {
  if (intervalId) {
    clearInterval(intervalId);
  }

  intervalId = setInterval(() => {
    webSocketManager.connect(
      "notifications",
      `${WEB_SOCKET_URL}/notifications/`,
      {
        onOpen: () => {
          toast.success(
            () => (
              <div className="flex flex-col space-y-1">
                <span className="font-semibold">Connection re-established</span>
                <span className="text-xs">
                  Connection to the server has been re-established.
                </span>
              </div>
            ),
            {
              duration: 4000,
              id: "connection-closed",
              style: {
                background: "hsl(var(--background))",
                color: "hsl(var(--foreground))",
                fontSize: "0.875rem",
              },
              ariaProps: {
                role: "status",
                "aria-live": "polite",
              },
            },
          );
          clearInterval(intervalId); // Clear the interval once connected
        },
      },
    );
  }, 5000);

  return () => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  };
};

export function NotificationMenu() {
  const [notificationsMenuOpen, setNotificationMenuOpen] = useHeaderStore.use(
    "notificationMenuOpen",
  );
  const [userHasNotifications, setUserHasNotifications] =
    useState<boolean>(false);
  const { userId } = useUserStore.get("user");
  const { notificationsData, notificationsLoading } = useNotificaitons(userId);
  const queryClient = useQueryClient();

  const markedAndInvalidate = async () => {
    await axios.get("/user/notifications/?max=10&mark_as_read=true");
    await queryClient.invalidateQueries({
      queryKey: ["userNotifications", userId],
    });
  };

  const readAllNotifications = async () => {
    const sendNotificationRequest = markedAndInvalidate();

    // Fire Toast
    await toast.promise(
      sendNotificationRequest,
      {
        loading: "Marking all notifications as read",
        success: "All notifications marked as read",
        error: "Failed to mark all notifications as read",
      },
      {
        id: "notification-toast",
        style: {
          background: "hsl(var(--background))",
          color: "hsl(var(--foreground))",
          fontSize: "0.875rem",
        },
        ariaProps: {
          role: "status",
          "aria-live": "polite",
        },
      },
    );

    setNotificationMenuOpen(false);
  };

  // React useEffect to connect to the websocket
  React.useEffect(() => {
    if (ENABLE_WEBSOCKETS && userId) {
      webSocketManager.connect(
        "notifications",
        `${WEB_SOCKET_URL}/notifications/`,
        {
          onOpen: () => console.info("Notifications Websocket Connected"),
          onMessage: (event: MessageEvent) => {
            const data = JSON.parse(event.data);
            console.log(data);
            queryClient
              .invalidateQueries({
                queryKey: ["userNotifications", userId],
              })
              .then(() => {
                toast.success(
                  () => (
                    <div className="flex flex-col space-y-1">
                      <span className="font-semibold">{data.event}</span>
                      <span className="text-xs">{data.description}</span>
                    </div>
                  ),
                  {
                    duration: 4000,
                    id: "notification-toast",
                    style: {
                      background: "hsl(var(--background))",
                      color: "hsl(var(--foreground))",
                    },
                    ariaProps: {
                      role: "status",
                      "aria-live": "polite",
                    },
                  },
                );
              });

            sound.play();
          },

          onClose: (event: CloseEvent) => {
            if (event.wasClean) {
              console.info(
                `Notifications Websocket Connection Closed Cleanly: ${event.code} ${event.reason}`,
              );
            } else {
              toast.loading(
                () => (
                  <div className="flex flex-col space-y-1">
                    <span className="font-semibold">Connection Closed</span>
                    <span className="text-xs">
                      Websocket Connection died. Reconnect will be attempted in
                      5 seconds.
                    </span>
                  </div>
                ),
                {
                  id: "connection-closed",
                  style: {
                    background: "hsl(var(--background))",
                    color: "hsl(var(--foreground))",
                    fontSize: "0.875rem",
                  },
                  ariaProps: {
                    role: "status",
                    "aria-live": "polite",
                  },
                },
              );

              reconnect();
            }
          },
        },
      );
    } else {
      console.info("Notifications Websocket Disabled");
    }

    return () => {
      // only cleanup if the websocket is enabled
      if (ENABLE_WEBSOCKETS && webSocketManager.has("notifications")) {
        webSocketManager.disconnect("notifications");
      }
    };
  }, [userId, queryClient]);

  // React useEffect to change the userHasNotifications state
  // when the notificationsData changes
  React.useEffect(() => {
    if (
      notificationsData &&
      (notificationsData as UserNotification).unreadList
    ) {
      setUserHasNotifications(
        (notificationsData as UserNotification).unreadList.length > 0,
      );
    }
  }, [notificationsData]);

  return (
    <Popover
      open={notificationsMenuOpen}
      onOpenChange={(open) => setNotificationMenuOpen(open)}
    >
      <PopoverTrigger asChild>
        <nav className="relative inline-flex mx-4 cursor-pointer">
          <BellIcon className="h-5 w-5" />
          <span className="sr-only">Notifications</span>
          {userHasNotifications && (
            <span className="flex absolute h-2 w-2 top-0 right-0 -mt-1 -mr-1">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-25"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
            </span>
          )}
        </nav>
      </PopoverTrigger>
      <PopoverContent
        className="w-80"
        sideOffset={10}
        alignOffset={-40}
        align="end"
      >
        {notificationsLoading ? (
          <div className="flex flex-col space-y-2 px-4 py-2 border-b border-accent">
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
              notification={notificationsData as UserNotification}
              notificationLoading={notificationsLoading}
            />
          </ScrollArea>
        )}
        {userHasNotifications && (
          <div className="border-t pt-2 text-center flex items-center justify-center">
            <button
              className="text-sm flex items-center hover:bg-accent rounded-md p-2 outline-transparent"
              onClick={readAllNotifications}
            >
              Read All Notifications <ChevronRight className="h-4 w-4 ml-1" />
            </button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}
