import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Button } from "@trenova/shared/components/ui/button";
import { PlusIcon, UsersRoundIcon } from "lucide-react";
import { lazy, memo, useCallback, useState } from "react";
import { EmptyState, PanelHeader } from "../security-access/shared";
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

  return (
    <div className="bg-background rounded-lg border">
      <PanelHeader
        icon={<UsersRoundIcon />}
        title={t("Group role mappings")}
        description={t("Resolve external directory groups into application roles.")}
        action={
          <Button size="sm" onClick={onAdd} disabled={disabled}>
            <PlusIcon />
            {t("Add mapping")}
          </Button>
        }
      />
      <div className="px-2 pb-2">
        {directoryId ? (
          <DataTableLazyComponent rowCount={3} columnCount={5}>
            <SCIMGroupRoleMappingsTable organizationId={organizationId} directoryId={directoryId} />
          </DataTableLazyComponent>
        ) : (
          <EmptyState
            icon={<UsersRoundIcon />}
            label={t("Select a directory")}
            description={t("Choose a SCIM directory before loading group mappings.")}
            compact
          />
        )}
      </div>
    </div>
  );
});
