import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type {
  SwitchOrganizationRequest,
  SwitchOrganizationResponse,
} from "@trenova/shared/types/organization";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { toast } from "sonner";

export function useSwitchOrganization() {
  const t = useT();

  const queryClient = useQueryClient();
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

      queryClient.clear();

      void navigate("/");

      toast.success(t("Organization switched"), {
        description: t("You are now working in a different organization."),
      });
    },
  });
}
