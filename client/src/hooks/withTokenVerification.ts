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

import { useEffect } from "react";
import { useAuthStore } from "@/stores/AuthStore";
import axios from "@/lib/AxiosConfig";
import { clearUserSessionInfo, getUserId } from "@/lib/utils";

/**
 * Custom hook to verify the user's token.
 *
 * On mount, it checks if the user ID is present in the session storage. If it is, it sends a request to
 * verify the token. Depending on the result of the request, it updates the authentication status and
 * clears the session storage if necessary. It also manages loading states during the verification process.
 */
export const useVerifyToken = (): void => {
  const setIsAuthenticated = useAuthStore((state) => state.setIsAuthenticated);
  const setLoading = useAuthStore((state) => state.setLoading);
  const setInitialLoading = useAuthStore((state) => state.setInitialLoading);

  // Create a new broadcast channel
  const broadcast = new BroadcastChannel("sessionSync");

  // When we receive a message on our broadcast channel, update session data
  broadcast.onmessage = function (event) {
    sessionStorage.setItem(event.data.key, JSON.stringify(event.data.data));
  };

  useEffect(() => {
    const verifyToken = async (): Promise<void> => {
      setInitialLoading(true);

      const userId = getUserId();

      if (!userId) {
        clearUserSessionInfo();
        setIsAuthenticated(false);
        return;
      }

      try {
        setLoading(true);
        await axios.post("verify_token/");

        setIsAuthenticated(true);
      } catch (error) {
        sessionStorage.clear();
        setIsAuthenticated(false);
      } finally {
        setLoading(false);
      }
    };

    verifyToken().then(() => {
      setInitialLoading(false);
    });
  }, [setIsAuthenticated, setLoading, setInitialLoading]);
};
