import logoRainbow from "@/assets/logo.webp";
import { LazyImage } from "@/components/image";
import { Metadata } from "@/components/metadata";
import { SidebarLayout } from "@/components/navigation";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { usePermissionPolling } from "@/hooks/use-permission-polling";
import { useRealtimeConnection } from "@/hooks/use-realtime-connection";
import { cn } from "@trenova/shared/lib/utils";
import { authService } from "@trenova/shared/services/auth";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import type { RoleSummary } from "@trenova/shared/types/role";
import { Check, ShieldCheck, UserRound } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useState } from "react";
import { Outlet } from "react-router";

function resolveAuthorizedRoles(manifest: PermissionManifest): RoleSummary[] {
  return manifest.authorizedRoles.length
    ? manifest.authorizedRoles
    : manifest.authorizedRoleIds.map((roleId) => ({
        id: roleId,
        name: roleId,
        description: "",
        isSystem: false,
      }));
}

function RoleActivationGate({ manifest }: { manifest: PermissionManifest }) {
  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const authorizedRoles = resolveAuthorizedRoles(manifest);
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>(() =>
    authorizedRoles.length === 1 ? [authorizedRoles[0].id] : [],
  );
  const [isActivating, setIsActivating] = useState(false);

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
    } catch (error) {
      handleMutationError({ error, resourceName: "Role" });
    } finally {
      setIsActivating(false);
    }
  };

  const selectedCount = selectedRoleIds.length;
  const buttonLabel =
    selectedCount === 0
      ? "Activate roles"
      : selectedCount === 1
        ? "Activate 1 role"
        : `Activate ${selectedCount} roles`;

  return (
    <>
      <Metadata title="Select roles" description="Choose the roles to activate for this session" />
      <div className="fixed inset-0 h-svh w-full overflow-hidden bg-background">
        <div className="relative flex h-full flex-col items-center justify-center gap-6 p-6 md:p-10">
          <LazyImage
            src={logoRainbow}
            alt="Trenova Logo"
            className="size-14 object-contain drop-shadow-[0_4px_24px_rgba(255,255,255,0.25)]"
          />
          <m.div
            className="w-full max-w-[400px]"
            initial={{ opacity: 0, y: 8, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            transition={{ duration: 0.22, ease: "easeOut" }}
          >
            <Card className="rounded-2xl border-border bg-background backdrop-blur-md">
              <CardHeader className="text-left">
                <CardTitle>Select active roles</CardTitle>
                <CardDescription className="mt-1">
                  Choose the roles to activate for this session.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="space-y-2">
                    {authorizedRoles.map((role, index) => {
                      const isSelected = selectedRoleIds.includes(role.id);
                      const RoleIcon = role.isSystem ? ShieldCheck : UserRound;

                      return (
                        <m.button
                          key={role.id}
                          type="button"
                          aria-pressed={isSelected}
                          initial={{ opacity: 0, y: 6 }}
                          animate={{ opacity: 1, y: 0 }}
                          transition={{ duration: 0.2, ease: "easeOut", delay: 0.04 * index }}
                          className={cn(
                            "flex min-h-13 w-full cursor-pointer items-center gap-3 rounded-md border px-3 py-2 text-left transition-colors hover:bg-muted",
                            isSelected ? "border-primary bg-primary/5" : "border-border bg-background",
                          )}
                          disabled={isActivating}
                          onClick={() => toggleRole(role.id)}
                        >
                          <span
                            className={cn(
                              "grid size-8 shrink-0 place-items-center rounded-md transition-colors",
                              isSelected
                                ? "bg-primary/10 text-primary"
                                : "bg-muted text-muted-foreground",
                            )}
                          >
                            <RoleIcon className="size-4" />
                          </span>
                          <span className="min-w-0 flex-1">
                            <span className="block truncate text-sm font-medium">{role.name}</span>
                            {role.description && (
                              <span className="block truncate text-xs text-muted-foreground">
                                {role.description}
                              </span>
                            )}
                          </span>
                          {role.isSystem && <Badge variant="outline">System</Badge>}
                          <span
                            className={cn(
                              "grid size-5 shrink-0 place-items-center rounded-full border transition-colors",
                              isSelected
                                ? "border-primary bg-primary text-primary-foreground"
                                : "border-border",
                            )}
                          >
                            <AnimatePresence>
                              {isSelected && (
                                <m.span
                                  initial={{ scale: 0.5, opacity: 0 }}
                                  animate={{ scale: 1, opacity: 1 }}
                                  exit={{ scale: 0.5, opacity: 0 }}
                                  transition={{ duration: 0.15, ease: "easeOut" }}
                                >
                                  <Check className="size-3" />
                                </m.span>
                              )}
                            </AnimatePresence>
                          </span>
                        </m.button>
                      );
                    })}
                  </div>
                  <Button
                    className="w-full"
                    disabled={selectedCount === 0}
                    isLoading={isActivating}
                    loadingText="Activating..."
                    onClick={() => void activateRoles()}
                  >
                    {buttonLabel}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </m.div>
        </div>
      </div>
    </>
  );
}

export function AppLayout() {
  usePermissionPolling();
  useRealtimeConnection();
  const manifest = usePermissionStore((state) => state.manifest);

  if (manifest?.requiresRoleActivation) {
    return <RoleActivationGate key={manifest.organizationId} manifest={manifest} />;
  }

  return (
    <SidebarLayout>
      <Outlet />
    </SidebarLayout>
  );
}
