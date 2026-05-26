import { SidebarLayout } from "@/components/navigation";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { usePermissionPolling } from "@/hooks/use-permission-polling";
import { useRealtimeConnection } from "@/hooks/use-realtime-connection";
import { authService } from "@/services/auth";
import { usePermissionStore } from "@/stores/permission-store";
import { useState } from "react";
import { Outlet } from "react-router";

export function AppLayout() {
  usePermissionPolling();
  useRealtimeConnection();
  const manifest = usePermissionStore((state) => state.manifest);
  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>([]);
  const [isActivating, setIsActivating] = useState(false);

  if (manifest?.requiresRoleActivation) {
    const authorizedRoles = manifest.authorizedRoles.length
      ? manifest.authorizedRoles
      : manifest.authorizedRoleIds.map((roleId) => ({
          id: roleId,
          name: roleId,
          description: "",
          isSystem: false,
        }));

    const toggleRole = (roleId: string) => {
      setSelectedRoleIds((current) =>
        current.includes(roleId) ? current.filter((id) => id !== roleId) : [...current, roleId],
      );
    };

    const activateRoles = async () => {
      setIsActivating(true);
      try {
        await authService.activateSessionRoles(selectedRoleIds);
        await fetchManifest();
      } finally {
        setIsActivating(false);
      }
    };

    return (
      <div className="grid min-h-dvh place-items-center bg-background px-6">
        <section className="w-full max-w-md space-y-5">
          <div className="space-y-1">
            <h1 className="text-xl font-semibold">Select active roles</h1>
            <p className="text-sm text-muted-foreground">
              Choose the roles to activate for this session.
            </p>
          </div>
          <div className="space-y-2 rounded-md border bg-card p-3">
            {authorizedRoles.map((role) => (
              <label
                key={role.id}
                className="flex min-h-12 cursor-pointer items-start gap-3 rounded px-2 py-2 text-sm hover:bg-muted"
              >
                <Checkbox
                  className="mt-0.5"
                  checked={selectedRoleIds.includes(role.id)}
                  onCheckedChange={() => toggleRole(role.id)}
                />
                <span className="min-w-0 flex-1 space-y-1">
                  <span className="block font-medium">{role.name}</span>
                  {role.description && (
                    <span className="block text-xs text-muted-foreground">{role.description}</span>
                  )}
                </span>
              </label>
            ))}
          </div>
          <Button
            className="w-full"
            disabled={selectedRoleIds.length === 0}
            isLoading={isActivating}
            onClick={activateRoles}
          >
            Activate roles
          </Button>
        </section>
      </div>
    );
  }

  return (
    <SidebarLayout>
      <Outlet />
    </SidebarLayout>
  );
}
