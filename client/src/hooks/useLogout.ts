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

import axios from "axios";
import { useNavigate } from "react-router-dom";
import { clearAllCookies } from "@/helpers/auth";
import { useAuthStore } from "@/stores/AuthStore";

export function useLogout() {
  const navigate = useNavigate();
  const [, setIsAuthenticated] = useAuthStore((state) => [
    state.isAuthenticated,
    state.setIsAuthenticated,
  ]);
  return async () => {
    try {
      await axios.post("/logout/");
      sessionStorage.clear();
      const returnUrl = sessionStorage.getItem("returnUrl");
      if (returnUrl !== "/login" && returnUrl !== "/logout") {
        sessionStorage.removeItem("returnUrl");
      }
      // You can also add the setIsAuthenticated here if it is needed
      setIsAuthenticated(false);

      clearAllCookies(); // Clear all cookies to prevent auto-login
      navigate("/login");
    } catch (exception) {
      // handle errors
    }
  };
}
