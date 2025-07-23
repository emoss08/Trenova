/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { http } from "@/lib/http-client";
import { api } from "@/services/api";
import { useAuthActions, useUser } from "@/stores/user-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

// Validation interval - check every 5 minutes
const VALIDATION_INTERVAL = 5 * 60 * 1000;

export function useSession() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useUser();
  const { setUser, clearAuth } = useAuthActions();

  // Query to fetch current user and handle session validation
  const { isLoading } = useQuery({
    queryKey: ["currentUser"],
    queryFn: async () => {
      // First validate session
      const sessionResponse = await api.auth.validateSession();
      if (!sessionResponse.data.valid) {
        throw new Error("Session invalid");
      }

      // Then fetch user data
      const userResponse = await http.get("/users/me");
      return userResponse.data;
    },
    retry: false,
    refetchInterval: VALIDATION_INTERVAL,
  });

  if (!isLoading && !user) {
    clearAuth();
    queryClient.clear();
    navigate("/auth", { replace: true });
  } else if (user) {
    setUser(user);
  }

  return {
    user,
    isLoading,
    isAuthenticated: !!user,
  };
}
