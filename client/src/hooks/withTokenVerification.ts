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

import { useEffect, useState } from "react";
import { useAuthStore } from "@/stores/AuthStore";
import axios from "@/lib/AxiosConfig";
import { getUserDetails } from "@/requests/UserRequestFactory";

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
  const setInitialLoading = useAuthStore((state) => state.setInitialLoading);
  const [isVerifying, setIsVerifying] = useState(true);

  useEffect(() => {
    const verifyToken = async (): Promise<void> => {
      try {
        setLoading(true);
        const response = await axios.post("verify_token/");
        sessionStorage.setItem("mt_user_id", response.data.user_id as string);
        sessionStorage.setItem(
          "mt_organization_id",
          response.data.organization_id as string
        );

        const userInfo = await getUserDetails(response.data.user_id as string);
        sessionStorage.setItem(
          "mt_user_permissions",
          JSON.stringify(userInfo.user_permissions)
        );
        sessionStorage.setItem(
          "mt_user_groups",
          JSON.stringify(userInfo.groups)
        );
        sessionStorage.setItem("mt_is_admin", userInfo.is_staff.toString());

        setIsAuthenticated(true);
      } catch (error) {
        sessionStorage.clear();
        setIsAuthenticated(false);
      } finally {
        setLoading(false);
      }
    };

    verifyToken().then(() => {
      setIsVerifying(false);
    });
  }, [setIsAuthenticated, setLoading, setInitialLoading]);
  return { isVerifying };
};
