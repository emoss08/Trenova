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
