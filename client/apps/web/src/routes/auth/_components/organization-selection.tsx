import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { getNameInitials } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { UserOrganization } from "@trenova/shared/types/organization";
import { useCallback, useRef, useState } from "react";
import { AuthSubmit } from "./auth-field";
import {
  AuthHeading,
  AuthOption,
  AuthTray,
  KeyHint,
  StepCrumbs,
  useOptionListKeyboard,
} from "./auth-primitives";

export function OrganizationSelection({
  organizations,
  stepLabel,
  onBack,
  onSelected,
}: {
  organizations: UserOrganization[];
  stepLabel: string;
  onBack: () => void;
  onSelected: (organization: UserOrganization) => Promise<void> | void;
}) {
  const t = useT();

  const setUser = useAuthStore((state) => state.setUser);
  const clearPermissions = usePermissionStore((state) => state.clearPermissions);
  const listRef = useRef<HTMLDivElement>(null);
  const [selectedOrganizationId, setSelectedOrganizationId] = useState(
    () =>
      organizations.find((organization) => organization.isCurrent)?.id ??
      organizations[0]?.id ??
      "",
  );
  const [isContinuing, setIsContinuing] = useState(false);

  const selectedOrganization = organizations.find(
    (organization) => organization.id === selectedOrganizationId,
  );

  const continueWithOrganization = useCallback(async () => {
    if (!selectedOrganization || isContinuing) {
      return;
    }

    setIsContinuing(true);
    try {
      // Staying put still needs a round trip: currentUser refreshes memberships that the
      // login response predates, and switching resets the session's active roles.
      const user = selectedOrganization.isCurrent
        ? await apiService.userService.currentUser()
        : await apiService.userService.switchOrganization({
            organizationId: selectedOrganization.id,
          });
      setUser(user);
      clearPermissions();
      await onSelected(selectedOrganization);
    } catch (error) {
      handleMutationError({ error, resourceName: "Organization" });
    } finally {
      setIsContinuing(false);
    }
  }, [clearPermissions, isContinuing, onSelected, selectedOrganization, setUser]);

  useOptionListKeyboard({
    listRef,
    enabled: !isContinuing,
    onSelectIndex: (index) => {
      const organization = organizations[index];
      if (organization) {
        setSelectedOrganizationId(organization.id);
      }
    },
    onAdvance: () => void continueWithOrganization(),
  });

  return (
    <>
      <StepCrumbs left={stepLabel} right={t("{0} available", organizations.length)} />
      <AuthHeading title={t("Select organization")}>
        {t("Choose the workspace for this session.")}
      </AuthHeading>

      <div
        ref={listRef}
        role="radiogroup"
        aria-label={t("Organizations")}
        className="mb-3.5 flex flex-col gap-2"
      >
        {organizations.map((organization, index) => (
          <AuthOption
            key={organization.id}
            role="radio"
            selected={organization.id === selectedOrganizationId}
            disabled={isContinuing}
            onSelect={() => setSelectedOrganizationId(organization.id)}
            leading={getNameInitials(organization.name, "ORG", { maxLength: 3, pad: true })}
            name={organization.name}
            meta={[organization.city, organization.state].filter(Boolean).join(", ")}
            chip={organization.isCurrent ? "Current" : undefined}
            shortcut={index < 9 ? formatShortcut(String(index + 1)) : undefined}
          />
        ))}
      </div>

      <AuthSubmit
        disabled={!selectedOrganization}
        busy={isContinuing}
        busyLabel={t("Opening workspace")}
        onClick={() => void continueWithOrganization()}
      >
        {t("Continue")}
      </AuthSubmit>

      <AuthTray
        onBack={onBack}
        hints={
          <>
            <KeyHint>↑↓</KeyHint> move <KeyHint>{formatShortcut("↵")}</KeyHint> continue
          </>
        }
      />
    </>
  );
}
