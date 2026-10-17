import { useApiMutation } from "@/hooks/use-api-mutation";
import { organization } from "@/lib/queries/organization";
import { apiService } from "@/services/api";
import { Settings01Icon } from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useSecurityAccess } from "./use-security-access";
import { SecuritySection } from "./shared";

export function RoleActivationSetting({ organizationId }: { organizationId: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const canUpdate = useSecurityAccess().accessPolicies.update;
  const settingsQuery = useQuery(organization.accessSettings(organizationId));

  const save = useApiMutation({
    mutationFn: (requireRoleActivation: boolean) =>
      apiService.organizationService.updateAccessSettings(organizationId, {
        requireRoleActivation,
        version: settingsQuery.data?.version ?? 0,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: organization.accessSettings(organizationId).queryKey,
      });
    },
    resourceName: t("Security settings"),
  });

  return (
    <SecuritySection title={t("Access settings")} icon={<Settings01Icon />}>
      <div className="flex items-start justify-between gap-4 py-3">
        <div className="min-w-0 space-y-0.5">
          <p id="role-activation-label" className="text-sm font-medium">
            {t("Require role activation")}
          </p>
          <p className="text-muted-foreground text-xs text-pretty">
            {t(
              "People choose which of their roles to work under each time they sign in or switch organization, and a newly assigned role waits until they choose it. Needed for dynamic separation of duty. When off, every role a person holds is in force at once.",
            )}
          </p>
        </div>
        {settingsQuery.isLoading ? (
          <Skeleton className="h-4.5 w-8 shrink-0 rounded-full" />
        ) : (
          <Switch
            checked={settingsQuery.data?.requireRoleActivation ?? false}
            disabled={!canUpdate || save.isPending || settingsQuery.isError}
            onCheckedChange={(checked) => save.mutate(checked)}
            aria-labelledby="role-activation-label"
          />
        )}
      </div>
    </SecuritySection>
  );
}
