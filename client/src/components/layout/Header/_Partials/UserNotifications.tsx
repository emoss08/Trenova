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

import React, { useEffect } from "react";
import {
  Badge,
  Button,
  Divider,
  Indicator,
  Popover,
  ScrollArea,
  UnstyledButton,
} from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faBell, faCheck } from "@fortawesome/pro-duotone-svg-icons";
import { useQuery, useQueryClient } from "react-query";
import axios from "axios";
import { notifications } from "@mantine/notifications";
import { Howl, Howler } from "howler";
import { Notifications } from "@/components/layout/Header/_Partials/Notifications";
import { getUserNotifications } from "@/services/UserRequestService";
import {
  ENABLE_WEBSOCKETS,
  getUserId,
  WEB_SOCKET_URL,
} from "@/helpers/constants";
import { useAuthStore } from "@/stores/AuthStore";
import { createWebsocketManager } from "@/helpers/websockets";
import { useNavbarStore } from "@/stores/HeaderStore";

import NotificationSound from "@/assets/audio/notification.webm";
import NotificationSoundMp3 from "@/assets/audio/notification.mp3";
import { useAsideStyles } from "@/assets/styles/AsideStyles";

const sound = new Howl({
  src: [NotificationSound, NotificationSoundMp3],
});

const webSocketManager = createWebsocketManager();

let intervalId: number;

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
          notifications.update({
            id: "connection-closed",
            title: "Connection re-established",
            message: "Websocket connection re-established.",
            color: "green",
            icon: <FontAwesomeIcon icon={faCheck} />,
          });
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

export function UserNotifications(): React.ReactElement | null {
  const [notificationMenuOpen] = useNavbarStore.use("notificationsMenuOpen");
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const userId = getUserId() || "";
  const queryClient = useQueryClient();
  const { classes } = useAsideStyles();

  useEffect(() => {
    if (ENABLE_WEBSOCKETS && isAuthenticated && userId) {
      // Connecting the websocket

      webSocketManager.connect(
        "notifications",
        `${WEB_SOCKET_URL}/notifications/`,
        {
          onOpen: () => console.info("Connected to notifications websocket"),

          onMessage: (event: MessageEvent) => {
            const data = JSON.parse(event.data);

            queryClient
              .invalidateQueries(["userNotifications", userId])
              .then(() => {
                notifications.show({
                  title: "New notification",
                  message: data.description,
                  color: "blue",
                  icon: <FontAwesomeIcon icon={faCheck} />,
                });

                if (data.attr === "report") {
                  queryClient
                    .invalidateQueries(["userReport", userId])
                    .then(() => {
                      console.info("Report query invalidated");
                    });
                }

                sound.play();
                Howler.volume(0.5);
              });
          },
          onClose: (event: CloseEvent) => {
            if (event.wasClean) {
              console.log("Connection closed cleanly");
            } else {
              notifications.show({
                id: "connection-closed",
                title: "Connection closed",
                message:
                  "Websocket connection died. Reconnect will be attempted in 5 second.",
                color: "blue",
                loading: true,
                autoClose: false,
                withCloseButton: false,
              });
              reconnect();
            }
          },
          onError: (error: Event) => {
            console.error(`[error] ${error}`);
          },
        },
      );
    } else if (isAuthenticated && !userId) {
      webSocketManager.disconnect("notifications");
    }
    // On component unmount, disconnect the websocket
    return () => {
      if (isAuthenticated) {
        webSocketManager.disconnect("notifications");
      }
    };
  }, [isAuthenticated, userId, queryClient]);

  const { data: notificationData, isLoading: isNotificationDataLoading } =
    useQuery({
      queryKey: ["userNotifications", userId],
      queryFn: () => {
        if (!userId) {
          return Promise.resolve(null);
        }
        return getUserNotifications();
      },
      initialData: () =>
        queryClient.getQueryData(["userNotifications", userId]),
    });

  const readAllNotifications = async () => {
    await axios.get("/user/notifications/?max=10&mark_as_read=true");
    notifications.show({
      title: "Notifications marked as read",
      message: "All notifications have been marked as read",
      color: "blue",
      icon: <FontAwesomeIcon icon={faCheck} />,
    });
    await queryClient.invalidateQueries(["userNotifications", userId]);
    useNavbarStore.set("notificationsMenuOpen", false);
  };

  if (!userId) {
    return null;
  }

  return (
    <Popover
      width={300}
      position="right-start"
      withArrow
      shadow="md"
      opened={notificationMenuOpen}
      trapFocus
      onClose={() => {
        useNavbarStore.set("notificationsMenuOpen", false);
      }}
    >
      <Popover.Target>
        {notificationData && notificationData?.unreadCount > 0 ? (
          <div className={classes.mainLinks}>
            <UnstyledButton
              className={classes.mainLink}
              onClick={() => useNavbarStore.set("notificationsMenuOpen", true)}
            >
              <div className={classes.mainLinkInner}>
                <FontAwesomeIcon
                  size="lg"
                  icon={faBell}
                  className={classes.mainLinkIcon}
                />
                <span>Notifications</span>
              </div>
              <Indicator processing color="violet">
                <Badge
                  size="sm"
                  variant="filled"
                  className={classes.mainLinkBadge}
                >
                  {notificationData?.unreadCount || 0}
                </Badge>
              </Indicator>
            </UnstyledButton>
          </div>
        ) : (
          <div className={classes.mainLinks}>
            <UnstyledButton
              className={classes.mainLink}
              onClick={() => {
                useNavbarStore.set(
                  "notificationsMenuOpen",
                  !notificationMenuOpen,
                );
              }}
            >
              <div className={classes.mainLinkInner}>
                <FontAwesomeIcon
                  size="lg"
                  icon={faBell}
                  className={classes.mainLinkIcon}
                />
                <span>Notifications</span>
              </div>
              <Badge
                size="sm"
                variant="filled"
                className={classes.mainLinkBadge}
              >
                0
              </Badge>
            </UnstyledButton>
          </div>
        )}
      </Popover.Target>
      <Popover.Dropdown>
        <ScrollArea h={250} scrollbarSize={4}>
          {notificationData && (
            <Notifications
              notification={notificationData}
              notificationLoading={isNotificationDataLoading}
            />
          )}
        </ScrollArea>
        {notificationData && notificationData?.unreadCount > 0 ? (
          <>
            <Divider mb={2} mt={10} />
            <div
              key={Math.random()}
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                marginTop: "5px",
              }}
            >
              <Button
                leftIcon={<FontAwesomeIcon icon={faCheck} />}
                variant="subtle"
                color="dark"
                size="sm"
                className={classes.button}
                onClick={readAllNotifications}
              >
                Mark all as read
              </Button>
            </div>
          </>
        ) : null}
      </Popover.Dropdown>
    </Popover>
  );
}
