import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { authService } from "@trenova/shared/services/auth";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";
import type { RoleSummary } from "@trenova/shared/types/role";
import { ShieldIcon } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import { AuthCardBody } from "./auth-card";
import { AuthSubmit } from "./auth-field";
import {
  AuthOption,
  AuthTray,
  KeyHint,
  StepCrumbs,
  StepHeading,
  Tally,
  useOptionListKeyboard,
} from "./auth-primitives";

/**
 * A manifest that requires activation always carries the ids; the summaries are the
 * richer form and are what the rows want. When only ids came back — an older cached
 * manifest, say — the id stands in for the name rather than the step rendering empty.
 */
export function resolveAuthorizedRoles(manifest: PermissionManifest): RoleSummary[] {
  return manifest.authorizedRoles.length
    ? manifest.authorizedRoles
    : manifest.authorizedRoleIds.map((roleId) => ({
        id: roleId,
        name: roleId,
        description: "",
        isSystem: false,
      }));
}

export function RoleSelection({
  roles,
  organizationName,
  stepLabel,
  onBack,
  onActivated,
}: {
  roles: RoleSummary[];
  organizationName?: string;
  stepLabel: string;
  onBack?: () => void;
  onActivated?: (activated: RoleSummary[]) => Promise<void> | void;
}) {
  const t = useT();

  const fetchManifest = usePermissionStore((state) => state.fetchManifest);
  const listRef = useRef<HTMLDivElement>(null);
  // A single authorized role is not a choice; preselect it so the step is one keystroke.
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>(() =>
    roles.length === 1 ? [roles[0].id] : [],
  );
  const [isActivating, setIsActivating] = useState(false);

  const selectedRoles = useMemo(
    () => roles.filter((role) => selectedRoleIds.includes(role.id)),
    [roles, selectedRoleIds],
  );
  const selectedCount = selectedRoleIds.length;
  // Suppressed rather than shown as zero when any role on offer arrived without a count
  // — see countPermissions in auth-form.tsx for why one can. The test is over every
  // role, not the selected ones: with nothing selected yet an empty sum is 0 either
  // way, and the crumb must not promise a tally it cannot keep once a row is picked.
  const permissionTotal = roles.some((role) => role.permissionCount === undefined)
    ? undefined
    : selectedRoles.reduce((total, role) => total + (role.permissionCount ?? 0), 0);

  const toggleRole = useCallback((roleId: string) => {
    setSelectedRoleIds((current) =>
      current.includes(roleId) ? current.filter((id) => id !== roleId) : [...current, roleId],
    );
  }, []);

  const activateRoles = useCallback(async () => {
    if (selectedRoleIds.length === 0 || isActivating) {
      return;
    }

    setIsActivating(true);
    try {
      await authService.activateSessionRoles(selectedRoleIds);
      await fetchManifest();
      await onActivated?.(selectedRoles);
    } catch (error) {
      handleMutationError({ error, resourceName: "Role" });
    } finally {
      setIsActivating(false);
    }
  }, [fetchManifest, isActivating, onActivated, selectedRoleIds, selectedRoles]);

  useOptionListKeyboard({
    listRef,
    enabled: !isActivating,
    onSelectIndex: (index) => {
      const role = roles[index];
      if (role) {
        toggleRole(role.id);
      }
    },
    onAdvance: () => void activateRoles(),
  });

  return (
    <AuthCardBody>
      <StepCrumbs
        left={stepLabel}
        right={
          permissionTotal === undefined ? (
            `${selectedCount} of ${roles.length} selected`
          ) : (
            <>
              <Tally value={permissionTotal} /> {t("permission{0}", permissionTotal === 1 ? "" : "s")}
            </>
          )
        }
      />
      <StepHeading title={t("Select active roles")}>
        {organizationName
          ? t("Scope this session at {0}. You can switch later without signing out.", organizationName)
          : t("Scope this session. You can switch later without signing out.")}
      </StepHeading>

      <div
        ref={listRef}
        role="group"
        aria-label={t("Authorized roles")}
        className="mt-4 mb-3.5 flex flex-col gap-2"
      >
        {roles.map((role, index) => (
          <AuthOption
            key={role.id}
            role="checkbox"
            selected={selectedRoleIds.includes(role.id)}
            disabled={isActivating}
            onSelect={() => toggleRole(role.id)}
            leading={<ShieldIcon className="size-3.5" />}
            name={role.name}
            meta={role.description || undefined}
            chip={role.isSystem ? "System" : "Custom"}
            shortcut={index < 9 ? formatShortcut(String(index + 1)) : undefined}
          />
        ))}
      </div>

      <AuthSubmit
        disabled={selectedCount === 0}
        isLoading={isActivating}
        loadingText={t("Issuing credential")}
        onClick={() => void activateRoles()}
      >
        {selectedCount === 0
          ? t("Select at least one role")
          : t("Activate {0} role{1}", selectedCount, selectedCount === 1 ? "" : "s")}
      </AuthSubmit>

      {onBack ? (
        <AuthTray
          onBack={onBack}
          hints={
            <>
              <KeyHint>{formatShortcut("1–9")}</KeyHint> toggle{" "}
              <KeyHint>{formatShortcut("↵")}</KeyHint> activate
            </>
          }
        />
      ) : (
        <div className="text-subtle-foreground mt-3.5 flex items-center justify-end gap-1.5 text-[11.5px] whitespace-nowrap">
          <KeyHint>{formatShortcut("1–9")}</KeyHint> toggle <KeyHint>{formatShortcut("↵")}</KeyHint>{" "}
          activate
        </div>
      )}
    </AuthCardBody>
  );
}
