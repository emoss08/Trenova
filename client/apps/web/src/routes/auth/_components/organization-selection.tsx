import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { UserOrganization } from "@trenova/shared/types/organization";
import { useState } from "react";
import { useNavigate } from "react-router";

export function OrganizationSelection({ organizations }: { organizations: UserOrganization[] }) {
  const navigate = useNavigate();
  const setUser = useAuthStore((state) => state.setUser);
  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const clearPermissions = usePermissionStore((state) => state.clearPermissions);
  const [selectedOrganizationId, setSelectedOrganizationId] = useState(
    organizations.find((organization) => organization.isCurrent)?.id ?? organizations[0]?.id ?? "",
  );
  const [isContinuing, setIsContinuing] = useState(false);

  const selectedOrganization = organizations.find(
    (organization) => organization.id === selectedOrganizationId,
  );

  const continueWithOrganization = async () => {
    if (!selectedOrganization) {
      return;
    }

    setIsContinuing(true);
    try {
      const user = selectedOrganization.isCurrent
        ? await apiService.userService.currentUser()
        : await apiService.userService.switchOrganization({
            organizationId: selectedOrganization.id,
          });
      setUser(user);
      clearPermissions();
      await fetchManifest();
      void navigate("/", { replace: true });
    } catch (error) {
      handleMutationError({ error, resourceName: "Organization" });
    } finally {
      setIsContinuing(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        {organizations.map((organization) => {
          const isSelected = organization.id === selectedOrganizationId;
          const location = [organization.city, organization.state].filter(Boolean).join(", ");

          return (
            <button
              key={organization.id}
              type="button"
              aria-pressed={isSelected}
              className={cn(
                "flex min-h-13 w-full cursor-pointer items-center gap-3 rounded-md border px-3 py-2 text-left transition-colors hover:bg-muted",
                isSelected ? "border-primary bg-primary/5" : "border-border bg-background",
              )}
              disabled={isContinuing}
              onClick={() => setSelectedOrganizationId(organization.id)}
            >
              <span className="grid size-8 shrink-0 place-items-center rounded-md bg-muted text-xs font-semibold">
                {organization.name.slice(0, 2).toUpperCase()}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium">{organization.name}</span>
                {location && (
                  <span className="block truncate text-xs text-muted-foreground">{location}</span>
                )}
              </span>
              {organization.isCurrent && <Badge variant="outline">Current</Badge>}
            </button>
          );
        })}
      </div>
      <Button
        type="button"
        className="w-full"
        disabled={!selectedOrganization}
        isLoading={isContinuing}
        loadingText="Continuing..."
        onClick={() => void continueWithOrganization()}
      >
        Continue
      </Button>
    </div>
  );
}
