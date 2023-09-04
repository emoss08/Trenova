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

import React, { Suspense, useEffect } from "react";
import { Card, Flex, Skeleton } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faX } from "@fortawesome/pro-duotone-svg-icons";
import { usePageStyles } from "@/assets/styles/PageStyles";
import { WebSocketManager, WebsocketMessageProps } from "@/helpers/websockets";
import {
  ENABLE_WEBSOCKETS,
  getUserId,
  WEB_SOCKET_URL,
  WEBSOCKET_RETRY_INTERVAL,
} from "@/helpers/constants";
import { useAuthStore } from "@/stores/AuthStore";
import { billingClientStore } from "@/stores/BillingStores";
import { BillingExceptionModal } from "@/components/billing/_partials/BillingExceptionModal";
import { TransferConfirmModal } from "@/components/billing/_partials/TransferConfirmModal";

const webSocketManager = new WebSocketManager();
export const STEPS = [
  "get_started",
  "orders_ready",
  "billing_queue",
  "good_job",
];

/** Partial Page components */
const GettingStartedPage = React.lazy(
  () => import("../../components/billing/GettingStarted"),
);
const OrdersReadyPage = React.lazy(
  () => import("../../components/billing/OrdersReady"),
);
const BillingQueuePage = React.lazy(
  () => import("../../components/billing/BillingQueue"),
);
const GoodJobPage = React.lazy(
  () => import("../../components/billing/GoodJob"),
);

const BillingClient: React.FC = () => {
  const { classes } = usePageStyles();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const userId = getUserId() || "";
  const [websocketMessage, setWebsocketMessage] =
    billingClientStore.use("websocketMessage");
  const [step, setStep] = billingClientStore.use("step");

  useEffect(() => {
    if (ENABLE_WEBSOCKETS && isAuthenticated && userId) {
      webSocketManager.connect(
        "billing_client",
        `${WEB_SOCKET_URL}/billing_client/`,
        {
          onOpen: () => {
            notifications.show({
              title: "Success",
              message: "Successfully connected to the Billing Client",
              color: "green",
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
          },

          onMessage: (event: MessageEvent) => {
            const data = JSON.parse(event.data) as WebsocketMessageProps;

            console.log("Billing Client Websocket Message", data);
            if (data.action === "orders_ready" && data.step === 2) {
              notifications.show({
                id: "billing_client_orders_ready",
                title: "Please wait...",
                message: data.message as string,
                color: "blue",
                loading: true,
                autoClose: false,
                withCloseButton: false,
              });
            }
            if (data.action === "billing_queue" && data.step === 3) {
              notifications.update({
                id: "billing_client_orders_ready",
                title: "Completed",
                message: data.message as string,
                color: "green",
                icon: <FontAwesomeIcon icon={faCheck} />,
                autoClose: 5_000,
              });
            }
            if (
              data.action === "bill_orders" &&
              data.step === 4 &&
              data.status === "PROCESSING"
            ) {
              notifications.show({
                id: "bill_orders",
                title: "Please wait...",
                message: data.message as string,
                color: "blue",
                loading: true,
                autoClose: false,
                withCloseButton: false,
              });
            }
            if (
              data.action === "bill_orders" &&
              data.step === 4 &&
              data.status === "FAILURE"
            ) {
              notifications.update({
                id: "bill_orders",
                title: "Exceptions Raised",
                message: "We've found orders that need your attention!",
                color: "red",
                icon: <FontAwesomeIcon icon={faX} />,
                autoClose: 5_000,
              });
              billingClientStore.set("exceptionModalOpen", true);
            }
            if (
              data.action === "bill_orders" &&
              data.step === 4 &&
              data.status === "SUCCESS"
            ) {
              notifications.update({
                id: "bill_orders",
                title: "Success!",
                message: "Successfully billed orders!",
                color: "green",
                icon: <FontAwesomeIcon icon={faCheck} />,
                autoClose: 5_000,
              });
            }
            setWebsocketMessage(data);
          },

          onClose: (event: CloseEvent) => {
            if (event.wasClean) {
              console.info(
                `[close] Connection closed cleanly, code=${event.code} reason=${event.reason}`,
              );
            } else {
              console.info(
                "[close] Connection died. Reconnect will be attempted in 1 second.",
              );
              setTimeout(
                () =>
                  webSocketManager.connect(
                    "billing_client",
                    `${WEB_SOCKET_URL}/billing_client/`,
                  ),
                WEBSOCKET_RETRY_INTERVAL,
              );
            }
          },

          onError: (error: Event) => {
            console.log(`[error] ${error}`);
          },
        },
      );
    } else if (isAuthenticated && !userId) {
      webSocketManager.disconnect("billing_client");
    }

    // On component unmount, disconnect the websocket
    return () => {
      if (isAuthenticated) {
        webSocketManager.disconnect("billing_client");
      }
    };
  }, [isAuthenticated, userId]); // add dependencies here if necessary

  useEffect(() => {
    if (websocketMessage.action) {
      const newStep = STEPS.indexOf(websocketMessage.action);
      if (newStep > -1) {
        setStep(newStep);
      }
    }
  }, [websocketMessage]);

  const renderStep = () => {
    switch (step) {
      case 0:
        return (
          <Suspense fallback={<Skeleton height={700} />}>
            <GettingStartedPage websocketManager={webSocketManager} />
          </Suspense>
        );
      case 1:
        return (
          <Suspense fallback={<Skeleton height={700} />}>
            <OrdersReadyPage websocketManager={webSocketManager} />
          </Suspense>
        );
      case 2:
        return (
          <Suspense fallback={<Skeleton height={700} />}>
            <BillingQueuePage websocketManager={webSocketManager} />
          </Suspense>
        );
      case 3:
        return (
          <Suspense fallback={<Skeleton height={700} />}>
            <GoodJobPage websocketManager={webSocketManager} />
          </Suspense>
        );
      // Add more cases here for more steps...
      default:
        return null;
    }
  };
  return (
    <>
      <Flex>
        <Card className={classes.card}>{renderStep()}</Card>
      </Flex>
      <BillingExceptionModal
        websocketMessage={websocketMessage}
        websocketManager={webSocketManager}
      />
      <TransferConfirmModal />
    </>
  );
};
export default BillingClient;
