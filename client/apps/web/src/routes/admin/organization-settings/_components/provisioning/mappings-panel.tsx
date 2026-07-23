import { DataTableLazyComponent } from "@/components/error-boundary";
import { Button } from "@/components/ui/button";
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
  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<UsersRoundIcon />}
        title="Group role mappings"
        description="Resolve external directory groups into application roles."
        action={
          <Button size="sm" onClick={onAdd} disabled={disabled}>
            <PlusIcon />
            Add mapping
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
            label="Select a directory"
            description="Choose a SCIM directory before loading group mappings."
            compact
          />
        )}
      </div>
    </div>
  );
});
