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

import NotificationSoundMp3 from "@/assets/audio/notification.mp3";
import NotificationSound from "@/assets/audio/notification.webm";
import { Notifications } from "@/components/layout/notification_menu/notification";
import { Button } from "@/components/ui/button";
import { InternalLink } from "@/components/ui/link";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useNotifications } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import {
  ENABLE_WEBSOCKETS,
  TOAST_STYLE,
  WEB_SOCKET_URL,
} from "@/lib/constants";
import { createWebsocketManager } from "@/lib/websockets";
import { useUserStore } from "@/stores/AuthStore";
import { useHeaderStore } from "@/stores/HeaderStore";
import { UserNotification } from "@/types/accounts";
import { faBell, faCheck } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useQueryClient } from "@tanstack/react-query";
import { Howl } from "howler";
import React, { useState } from "react";
import toast from "react-hot-toast";

const sound = new Howl({
  src: [NotificationSound, NotificationSoundMp3],
  volume: 0.2,
  format: ["webm", "mp3"],
  mute: false,
});

const webSocketManager = createWebsocketManager();

let intervalId: number | undefined;

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
              style: TOAST_STYLE,
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
  }, 1000);

  return () => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  };
};

function NotificationButton({
  userHasNotifications,
  open,
}: {
  userHasNotifications: boolean;
  open: boolean;
}) {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon"
            variant="outline"
            role="button"
            aria-label="Open Application Grid"
            aria-expanded={open}
            className="border-muted-foreground/40 hover:border-muted-foreground/80 relative size-8"
          >
            <FontAwesomeIcon icon={faBell} className="size-5" />
            <span className="sr-only">Notifications</span>
            {userHasNotifications && (
              <span className="absolute -right-1 -top-1 flex size-2.5">
                <span className="absolute inline-flex size-full animate-ping rounded-full bg-lime-400 opacity-100"></span>
                <span className="ring-background relative inline-flex size-2.5 rounded-full bg-lime-600 ring-1"></span>
              </span>
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={5}>
          <span>Notifications</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function NotificationContent({
  notificationsData,
  notificationsLoading,
  userHasNotifications,
  readAllNotifications,
}: {
  notificationsData: UserNotification | undefined;
  notificationsLoading: boolean;
  userHasNotifications: boolean;
  readAllNotifications: () => void;
}) {
  return (
    <>
      {notificationsLoading ? (
        <div className="border-accent flex flex-col space-y-2 border-b px-4 py-2">
          <div className="flex items-center justify-between">
            <h4 className="font-medium leading-none">
              <Skeleton className="h-4 w-20" />
            </h4>
            <span className="text-muted-foreground text-xs">
              <Skeleton className="h-4 w-20" />
            </span>
          </div>
          <p className="text-muted-foreground text-sm">
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
      {!userHasNotifications && (
        <div className="select-none items-center justify-center border-t pt-2 text-center text-xs">
          Know when you have new notifications by enabling text notifications in
          your{" "}
          <InternalLink to="/account/settings/">Account Settings</InternalLink>
        </div>
      )}
      {userHasNotifications && (
        <div className="flex items-center justify-center border-t pt-2 text-center">
          <Button onClick={readAllNotifications} className="w-full">
            <FontAwesomeIcon icon={faCheck} className="mr-2 size-4" /> Mark all
            as read
          </Button>
        </div>
      )}
    </>
  );
}

export function NotificationMenu() {
  const [notificationsMenuOpen, setNotificationMenuOpen] = useHeaderStore.use(
    "notificationMenuOpen",
  );
  const [userHasNotifications, setUserHasNotifications] =
    useState<boolean>(false);
  const { id: userId } = useUserStore.get("user");
  const { notificationsData, notificationsLoading } = useNotifications(userId);
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
        style: TOAST_STYLE,
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
                    style: TOAST_STYLE,
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
                      Websocket Connection died. We will attempt to reconnect
                      shortly.
                    </span>
                  </div>
                ),
                {
                  id: "connection-closed",
                  style: TOAST_STYLE,
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
      onOpenChange={setNotificationMenuOpen}
    >
      <PopoverTrigger>
        <NotificationButton
          userHasNotifications={userHasNotifications}
          open={notificationsMenuOpen}
        />
      </PopoverTrigger>
      <PopoverContent
        className="bg-popover w-80 p-3"
        sideOffset={10}
        alignOffset={-40}
        align="end"
      >
        <NotificationContent
          notificationsData={notificationsData as UserNotification}
          notificationsLoading={notificationsLoading}
          userHasNotifications={userHasNotifications}
          readAllNotifications={readAllNotifications}
        />
      </PopoverContent>
    </Popover>
  );
}
