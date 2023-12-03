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

import { InvalidateQueryFilters, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";

/**
 * This hook listens for query invalidation messages and invalidates the corresponding queries.
 * It is used to invalidate queries on other tabs when a mutation is performed on one tab.
 * @returns void
 * @example
 * useQueryInvalidationListener();
 *
 * @see https://react-query.tanstack.com/guides/mutations#invalidating-queries
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Broadcast_Channel_API
 */
export const useQueryInvalidationListener = () => {
  const queryClient = useQueryClient();

  const handleInvalidationMessage = useCallback(
    (message: MessageEvent) => {
      try {
        if (
          message.data?.type === "invalidate" &&
          Array.isArray(message.data.queryKeys)
        ) {
          console.log(
            `%c[MONTA] Query invalidation message received: ${message.data.queryKeys}`,
            "color: #a742f5; font-weight: bold",
          );
          message.data.queryKeys.forEach((queryKey: InvalidateQueryFilters) => {
            queryClient.invalidateQueries(queryKey);
          });
        }
      } catch (error) {
        console.error("Error handling query invalidation message: ", error);
      }
    },
    [queryClient],
  );

  useEffect(() => {
    const broadcastChannel = new BroadcastChannel("query-invalidation");
    // Colored console log for logging purposes
    console.log(
      "%c[MONTA] Query invalidation listener registered",
      "color: #87f542; font-weight: bold",
    );

    broadcastChannel.addEventListener("message", handleInvalidationMessage);
    // Colored console log for logging purposes
    console.log(
      "%c[MONTA] Listening for query invalidation messages...",
      "color: #87f542; font-weight: bold",
    );

    return () => {
      broadcastChannel.removeEventListener(
        "message",
        handleInvalidationMessage,
      );
      broadcastChannel.close();
    };
  }, [handleInvalidationMessage]);
};
