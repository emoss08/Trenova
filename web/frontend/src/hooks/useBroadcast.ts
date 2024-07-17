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

import { useQueryClient } from "@tanstack/react-query";
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
            `%c[Trenova] Query invalidation message received: ${message.data.queryKeys}`,
            "color: #a742f5; font-weight: bold",
          );
          message.data.queryKeys.forEach((keyPattern: string) => {
            queryClient.invalidateQueries({
              predicate: (query) =>
                query.queryKey.some(
                  (keyPart) =>
                    typeof keyPart === "string" && keyPart.includes(keyPattern),
                ),
            });
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
      "%c[Trenova] Query invalidation listener registered",
      "color: #87f542; font-weight: bold",
    );

    broadcastChannel.addEventListener("message", handleInvalidationMessage);
    // Colored console log for logging purposes
    console.log(
      "%c[Trenova] Listening for query invalidation messages...",
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
