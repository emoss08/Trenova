import { useAuthStore } from "@/stores/user-store";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import { getCurrentUser, logout, validateSession } from "@/services/auth";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { toast } from "sonner";

const SESSION_CHECK_INTERVAL = 5 * 60 * 1000; // 5 minutes

export function useAuth() {
  const queryClient = useQueryClient();
  const { user, setUser, clearAuth } = useAuthStore();
  const navigate = useNavigate();

  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: async () => {
      const { data: sessionData } = await validateSession();

      if (!sessionData.valid) {
        throw new Error("Session invalid");
      }

      const { data: userData } = await getCurrentUser();
      return userData;
    },
    retry: false,
    refetchInterval: SESSION_CHECK_INTERVAL,
    enabled: !!user, // Only run if we have a user
  });

  // Handle authentication state changes
  useEffect(() => {
    if (sessionQuery.isSuccess && sessionQuery.data) {
      setUser(sessionQuery.data);
    } else if (sessionQuery.isError) {
      clearAuth();
      queryClient.clear();
      navigate("/auth");
      toast.error("Your session has expired. Please sign in again.");
    }
  }, [
    sessionQuery.isSuccess,
    sessionQuery.isError,
    sessionQuery.data,
    setUser,
    clearAuth,
    queryClient,
  ]);

  return {
    isLoading: sessionQuery.isPending,
    isError: sessionQuery.isError,
    isAuthenticated: !!user && !sessionQuery.isError,
  };
}

export function useLogout() {
  const queryClient = useQueryClient();
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const navigate = useNavigate();

  return async () => {
    await logout();
    clearAuth();
    queryClient.clear();
    navigate("/auth");
  };
}
