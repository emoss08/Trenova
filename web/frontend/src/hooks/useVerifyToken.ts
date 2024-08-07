/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
