import { api } from "@/services/api";
import { useAuthActions, useUser } from "@/stores/user-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";

const SESSION_CHECK_INTERVAL = 5 * 60 * 1000;

export function useAuth() {
  const user = useUser();
  const { setUser, clearAuth } = useAuthActions();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: async () => {
      const { data: sessionData } = await api.auth.validateSession();

      if (!sessionData.valid) {
        throw new Error("Session invalid");
      }

      const { data: userData } = await api.auth.getCurrentUser();
      return userData;
    },
    retry: false,
    refetchInterval: SESSION_CHECK_INTERVAL,
    enabled: !!user,
  });

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
    navigate,
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
  const { clearAuth } = useAuthActions();
  const navigate = useNavigate();

  return async () => {
    await api.auth.logout();
    clearAuth();
    queryClient.clear();
    navigate("/auth");
  };
}
