import { http } from "@/lib/http-client";
import { validateSession } from "@/services/auth";
import { useAuthStore } from "@/stores/user-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

// Validation interval - check every 5 minutes
const VALIDATION_INTERVAL = 5 * 60 * 1000;

export function useSession() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user, setUser } = useAuthStore();

  // Query to fetch current user and handle session validation
  const { isLoading } = useQuery({
    queryKey: ["currentUser"],
    queryFn: async () => {
      // First validate session
      const sessionResponse = await validateSession();
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
    clearUser();
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
