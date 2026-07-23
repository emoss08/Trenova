import { parseInvalidationEvent, RESOURCE_EVENT_NAME } from "@/hooks/realtime-patching";
import { notification } from "@/lib/queries/notification";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { toast } from "sonner";

const DASH_RESOURCE_QUERY_KEYS: Record<string, string[]> = {
  shipments: ["dash-loads"],
  shipment_comment: ["dash-load-comments"],
  driver_settlement: [
    "dash-settlements",
    "dash-settlement",
    "dash-period-summary",
    "dash-recent-pay-events",
  ],
  driver_pay_event: ["dash-period-summary", "dash-recent-pay-events", "dash-loads"],
  settlement_dispute: ["dash-disputes"],
  document: ["dash-load-documents", "dash-profile-documents"],
  driver_expense: ["dash-expenses"],
  worker_pto: ["dash-pto"],
  dash_control: ["dash-features"],
};

export function useDashRealtime() {
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const queryClient = useQueryClient();

  const orgId = user?.currentOrganizationId;
  const buId = user?.businessUnitId;
  const userId = user?.id;

  useEffect(() => {
    if (!isAuthenticated || !orgId || !buId) {
      return;
    }

    const realtime = apiService.realtimeService;
    realtime.connect();
    const channel = realtime.getChannel(realtime.getDataEventsChannelName(orgId, buId));

    const onEvent = (message: { name?: string; data?: unknown }) => {
      if (message.name !== RESOURCE_EVENT_NAME) {
        return;
      }
      const event = parseInvalidationEvent(message.data);
      if (!event || event.organizationId !== orgId || event.businessUnitId !== buId) {
        return;
      }

      if (event.resource === "notifications" && event.action === "created") {
        const entity = event.entity as
          | { targetUserId?: string | null; title?: string; message?: string }
          | undefined;
        const isForMe = !entity?.targetUserId || entity.targetUserId === userId;
        if (isForMe && entity?.title) {
          toast.info(entity.title, { description: entity.message });
        }
        void queryClient.invalidateQueries({ queryKey: notification._def });
        return;
      }

      const keys = DASH_RESOURCE_QUERY_KEYS[event.resource];
      if (!keys) {
        return;
      }
      for (const key of keys) {
        void queryClient.invalidateQueries({ queryKey: [key], refetchType: "all" });
      }
    };

    channel.subscribe(onEvent);
    return () => {
      channel.unsubscribe(onEvent);
    };
  }, [isAuthenticated, orgId, buId, userId, queryClient]);
}
