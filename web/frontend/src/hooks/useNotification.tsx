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

import { WEB_SOCKET_URL } from "@/lib/constants";
import { WebSocketManager, createWebsocketManager } from "@/lib/websockets";
import { useUserStore } from "@/stores/AuthStore";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

type WebsocketData = {
  type: string;
  title: string;
  content: string;
  clientId: string;
};

let webSocketManager: WebSocketManager | null = null;

export function useNotificationListener() {
  const queryClient = useQueryClient();
  const { id: userId } = useUserStore.get("user");

  if (!userId) {
    return;
  }

  if (!webSocketManager) {
    webSocketManager = createWebsocketManager();
  }

  if (webSocketManager.has(userId)) {
    return; // Prevent duplicate connections
  }

  webSocketManager.connect(
    "notifications",
    `${WEB_SOCKET_URL}/${userId}`,
    {
      onOpen: () =>
        // Colored console log for logging purposes
        console.log(
          "%c[Trenova] Connected to websocket for notifications",
          "color: #87f542; font-weight: bold",
        ),
      onMessage: (event: MessageEvent) => {
        const data = JSON.parse(event.data) as WebsocketData;

        queryClient
          .invalidateQueries({
            queryKey: ["userNotifications", userId],
          })
          .then(() => {
            toast.success(
              <div className="flex flex-col space-y-1">
                <span className="font-semibold">{data.title}</span>
                <span className="text-xs">{data.content}</span>
              </div>,
            );
          });
      },
      onClose: (event: CloseEvent) =>
        console.info(`Websocket closed: ${event.reason}`),
    },
    {
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
    },
  );
}
