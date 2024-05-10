import { createGlobalStore } from "@/lib/useGlobalStore";
import { WebSocketMessageProps } from "@/lib/websockets";
import { OrdersReadyProps } from "@/types/billing";
import { Row } from "@tanstack/react-table";

interface BillingClientStoreProps {
  step: number;
  websocketMessage: WebSocketMessageProps;
  exceptionModalOpen: boolean;
  transferConfirmModalOpen: boolean;
  approveTransfer: boolean;
  invalidOrders: Row<OrdersReadyProps>[];
}

export const billingClientStore = createGlobalStore<BillingClientStoreProps>({
  step: 0,
  websocketMessage: {
    action: "",
    status: "SUCCESS",
    message: "",
  },
  exceptionModalOpen: false,
  transferConfirmModalOpen: false,
  approveTransfer: false,
  invalidOrders: [],
});
