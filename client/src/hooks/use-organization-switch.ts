import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { toast } from "sonner";

export function useSwitchOrganization() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { setUser } = useAuthStore();

  return useApiMutation({
    mutationFn: (data) => apiService.userService.switchOrganization(data),
    resourceName: "Organization",
    onSuccess: (response) => {
      setUser(response.user);

      queryClient.clear();

      void navigate("/");

      toast.success("Organization switched", {
        description: "You are now working in a different organization.",
      });
    },
  });
}
