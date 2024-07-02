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
