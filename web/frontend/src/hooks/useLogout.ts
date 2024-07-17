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
import { createWebsocketManager } from "@/lib/websockets";
import { useAuthStore } from "@/stores/AuthStore";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

const webSocketManager = createWebsocketManager();

export function useLogout() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [, setIsAuthenticated] = useAuthStore((state) => [
    state.isAuthenticated,
    state.setIsAuthenticated,
  ]);
  return () => {
    try {
      axios.post("/auth/logout/").then(() => {});
      const returnUrl = sessionStorage.getItem("returnUrl");
      if (returnUrl !== "/login" && returnUrl !== "/logout") {
        sessionStorage.removeItem("returnUrl");
      }
      setIsAuthenticated(false);
      localStorage.removeItem("trenova-user-id"); // Clear user ID from localStorage
      navigate("/login");

      // Clear all queries
      queryClient.clear();

      webSocketManager.disconnectFromAll();
    } catch (exception) {
      console.error("[Trenova] Logout", exception);
    }
  };
}
