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
