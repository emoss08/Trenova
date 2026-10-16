import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Button } from "@trenova/shared/components/ui/button";
import { PlusIcon, Users01Icon } from "@trenova/shared/components/icons";
import { lazy, memo, useCallback, useState } from "react";
import { EmptyState, SecuritySection } from "../security-access/shared";
import { useSecurityAccess } from "../security-access/use-security-access";
import { SCIMGroupMappingCreatePanel } from "./mapping-panel";

type MappingsPanelControllerProps = {
  organizationId: string;
  directoryId: string;
  onProvisioningChange: () => Promise<void>;
};

const SCIMGroupRoleMappingsTable = lazy(() => import("./mappings-table"));

export const MappingsPanelController = memo(function MappingsPanelController({
  organizationId,
  directoryId,
}: MappingsPanelControllerProps) {
  const [createPanelOpen, setCreatePanelOpen] = useState(false);

  const addMapping = useCallback(() => {
    if (!directoryId) {
      return;
    }
    setCreatePanelOpen(true);
  }, [directoryId]);

  return (
    <>
      <MappingsPanel
        organizationId={organizationId}
        directoryId={directoryId}
        onAdd={addMapping}
        disabled={!directoryId}
      />
      <SCIMGroupMappingCreatePanel
        organizationId={organizationId}
        directoryId={directoryId}
        open={Boolean(directoryId) && createPanelOpen}
        onOpenChange={setCreatePanelOpen}
      />
    </>
  );
});

const MappingsPanel = memo(function MappingsPanel({
  organizationId,
  directoryId,
  onAdd,
  disabled,
}: {
  organizationId: string;
  directoryId: string;
  onAdd: () => void;
  disabled: boolean;
}) {
  const t = useT();
  const canManage = useSecurityAccess().scimDirectories.update;

  return (
    <SecuritySection
      title={t("Group role mappings")}
      icon={<Users01Icon />}
      help={t("Each mapping gives the members of a directory group a role here.")}
      action={
        canManage ? (
          <Button size="sm" onClick={onAdd} disabled={disabled}>
            <PlusIcon />
            {t("New mapping")}
          </Button>
        ) : undefined
      }
    >
      {directoryId ? (
        <DataTableLazyComponent rowCount={3} columnCount={5}>
          <SCIMGroupRoleMappingsTable organizationId={organizationId} directoryId={directoryId} />
        </DataTableLazyComponent>
      ) : (
        <EmptyState
          icon={<Users01Icon />}
          label={t("Select a directory")}
          description={t("Choose a SCIM directory before loading group mappings.")}
        />
      )}
    </SecuritySection>
  );
});
