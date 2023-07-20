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
import { useHeaderStore } from "@/stores/HeaderStore";
import {
  ActionIcon,
  Button,
  createStyles,
  Divider,
  Indicator,
  Popover,
  ScrollArea,
} from "@mantine/core";
import { faBell } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck } from "@fortawesome/pro-duotone-svg-icons";
import { Notifications } from "@/components/layout/Header/_Partials/Notifications";
import { useQuery, useQueryClient } from "react-query";
import { getUserNotifications } from "@/requests/UserRequestFactory";
import { getUserId, WEB_SOCKET_URL, ENABLE_WEBSOCKETS } from "@/lib/utils";
import axios from "axios";
import { notifications } from "@mantine/notifications";
import { useAuthStore } from "@/stores/AuthStore";
import { createWebsocketManager } from "@/utils/websockets";
import { Howl, Howler } from "howler";

import NotificationSound from "@/assets/audio/notification.webm";
import NotificationSoundMp3 from "@/assets/audio/notification.mp3";

const sound = new Howl({
  src: [NotificationSound, NotificationSoundMp3],
});

const useStyles = createStyles((theme) => ({
  button: {
    "&:hover": {
      backgroundColor: "transparent",
    },
    height: "30px",
    width: "160px",
  },
  hoverEffect: {
    svg: {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.gray[5]
          : theme.colors.gray[9],
    },
    "&:hover svg": {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.gray[0]
          : theme.colors.gray[7],
    },
  },
}));

const webSocketManager = createWebsocketManager();

const reconnect = () => {
  webSocketManager.connect(
    "notificaitons",
    `${WEB_SOCKET_URL}/notifications/`,
    {
      onOpen: () => console.info("Connected to notifications websocket"),
    }
  );
};

export const UserNotifications: React.FC = () => {
  const [notificationMenuOpen] = useHeaderStore.use("notificationsMenuOpen");
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const userId = getUserId() || "";
  const queryClient = useQueryClient();
  const { classes } = useStyles();

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
                sound.play();
                Howler.volume(0.5);
              });
          },
          onClose: (event: CloseEvent) => {
            if (event.wasClean) {
              console.info(
                `[close] Connection closed cleanly, code=${event.code} reason=${event.reason}`
              );
            } else {
              console.info(
                "[close] Connection died. Reconnect will be attempted in 1 second."
              );
              reconnect();
            }
          },
          onError: (error: Event) => {
            console.log(`[error] ${error}`);
          },
        }
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
  }, [isAuthenticated, userId]); // add dependencies here if necessary

  const { data: notificationData, isLoading: isNotificationDataLoading } =
    useQuery({
      queryKey: ["userNotifications", userId],
      queryFn: () => {
        if (!userId) {
          return Promise.resolve(null);
        }
        return getUserNotifications();
      },
      initialData: () => {
        return queryClient.getQueryData(["userNotifications", userId]);
      },
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
    useHeaderStore.set("notificationsMenuOpen", false);
  };

  if (!userId) {
    return null;
  }

  return (
    <>
      <Popover
        width={300}
        position="bottom"
        withArrow
        shadow="md"
        opened={notificationMenuOpen}
        trapFocus
        onClose={() => {
          useHeaderStore.set("notificationsMenuOpen", false);
        }}
      >
        <Popover.Target>
          {notificationData && notificationData?.unread_count > 0 ? (
            <Indicator withBorder processing color="violet">
              <ActionIcon
                className={classes.hoverEffect}
                onClick={() => {
                  useHeaderStore.set(
                    "notificationsMenuOpen",
                    !notificationMenuOpen
                  );
                }}
              >
                <FontAwesomeIcon icon={faBell} />
              </ActionIcon>
            </Indicator>
          ) : (
            <ActionIcon
              className={classes.hoverEffect}
              onClick={() => {
                useHeaderStore.set(
                  "notificationsMenuOpen",
                  !notificationMenuOpen
                );
              }}
            >
              <FontAwesomeIcon icon={faBell} />
            </ActionIcon>
          )}
        </Popover.Target>
        <Popover.Dropdown>
          <ScrollArea h={250} scrollbarSize={4}>
            <Notifications
              notification={notificationData}
              notificationLoading={isNotificationDataLoading}
            />
          </ScrollArea>
          {notificationData && notificationData?.unread_count > 0 ? (
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
    </>
  );
};
