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

import axios from "@/lib/axiosConfig";
import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import { useEffect, useState } from "react";

/**
 * Custom hook to verify the user's token.
 *
 * On mount, it checks if the user ID is present in the session storage. If it is, it sends a request to
 * verify the token. Depending on the result of the request, it updates the authentication status and
 * clears the session storage if necessary. It also manages loading states during the verification process.
 */
export const useVerifyToken = () => {
  const setIsAuthenticated = useAuthStore((state) => state.setIsAuthenticated);
  const setLoading = useAuthStore((state) => state.setLoading);
  const [user, setUser] = useUserStore.use("user");
  const [isVerifying, setIsVerifying] = useState(false);
  const [isInitializationComplete, setInitializationComplete] = useState(false);

  useEffect(() => {
    // Attempt to retrieve the user ID from localStorage
    const userId = localStorage.getItem("trenova-user-id");

    if (!user.id && userId) {
      // If there's a user ID in localStorage but not in state, verify the user
      setUser({ ...user, id: userId });
    }

    if (!user.id && !userId) {
      console.log("No user id found. Skipping token verification.");
      setInitializationComplete(true);
      setIsAuthenticated(false);
      setLoading(false);
      return;
    }

    const verifyToken = async () => {
      setIsVerifying(true);
      setLoading(true);

      try {
        const response = await axios.get("users/me/", {
          withCredentials: true,
        });
        if (response.status === 200) {
          setUser(response.data);
          localStorage.setItem("trenova-user-id", response.data.id); // Persist user ID to localStorage
          setIsAuthenticated(true);
        } else {
          setIsAuthenticated(false);
          localStorage.removeItem("trenova-user-id"); // Clear user ID from localStorage
        }
      } catch (error) {
        console.error("Error verifying token:", error);
        setIsAuthenticated(false);
        localStorage.removeItem("trenova-user-id"); // Clear user ID from localStorage
      } finally {
        setIsVerifying(false);
        setInitializationComplete(true);
        setLoading(false);
      }
    };

    verifyToken();
  }, [setIsAuthenticated, setLoading, setUser, user.id]);

  return { isVerifying, isInitializationComplete };
};
