import { WEB_SOCKET_URL } from "@/lib/constants";
import { createWebsocketManager } from "@/lib/websockets";
import { useUserStore } from "@/stores/AuthStore";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { toast } from "sonner";

export function useNotificationListener() {
  const queryClient = useQueryClient();
  const webSocketManager = createWebsocketManager();
  const { id: userId } = useUserStore.get("user");

  useEffect(() => {
    if (!userId) {
      return;
    }

    webSocketManager.connect("notifications", `${WEB_SOCKET_URL}/${userId}`, {
      onOpen: () => console.info("Connected to notifications websocket"),
      onMessage: (event: MessageEvent) => {
        const data = JSON.parse(event.data);
        queryClient
          .invalidateQueries({
            queryKey: ["userNotifications", userId],
          })
          .then(() => {
            toast.success(
              <div className="flex flex-col space-y-1">
                <span className="font-semibold">New Report Available!</span>
                <span className="text-xs">{data.content}</span>
              </div>,
            );
          });
      },
      onClose: (event: CloseEvent) =>
        console.info(`Websocket closed: ${event.reason}`),
    });

    return () => {
      webSocketManager.disconnect("notifications");
    };
  }, [queryClient, userId, webSocketManager]);
}
