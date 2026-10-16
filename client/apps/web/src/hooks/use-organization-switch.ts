import { useApiMutation } from "@/hooks/use-api-mutation";
import { clearSessionCache } from "@/lib/session-cache";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type {
  SwitchOrganizationRequest,
  SwitchOrganizationResponse,
} from "@trenova/shared/types/organization";
import { useNavigate } from "react-router";

export function useSwitchOrganization() {
  const navigate = useNavigate();
  const { setUser } = useAuthStore();

  return useApiMutation<SwitchOrganizationResponse, SwitchOrganizationRequest>({
    mutationFn: (data: SwitchOrganizationRequest) =>
      apiService.userService.switchOrganization(data),
    resourceName: "Organization",
    onSuccess: async (user) => {
      const { clearPermissions, fetchManifest } = usePermissionStore.getState();
      try {
        await fetchManifest();
      } catch (error) {
        clearPermissions();
        console.error("Failed to refresh permissions after organization switch:", error);
      }

      setUser(user);

      await clearSessionCache();

      void navigate("/");
    },
  });
}
