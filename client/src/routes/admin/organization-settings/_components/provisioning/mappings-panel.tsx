import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiService } from "@/services/api";
import type { SCIMGroupRoleMapping } from "@/types/iam";
import { useMutation, useQuery } from "@tanstack/react-query";
import { PlusIcon, Trash2Icon, UsersRoundIcon } from "lucide-react";
import { memo, useCallback, useState } from "react";
import { toast } from "sonner";
import { EmptyState, PanelHeader, RowSkeleton } from "../security-access/shared";
import { emptyMapping } from "./constants";
import { SCIMGroupMappingPanel, type SCIMGroupMappingPanelMode } from "./mapping-panel";

type MappingsPanelControllerProps = {
  organizationId: string;
  directoryId: string;
  onProvisioningChange: () => Promise<void>;
};

export const MappingsPanelController = memo(function MappingsPanelController({
  organizationId,
  directoryId,
  onProvisioningChange,
}: MappingsPanelControllerProps) {
  const [panelMode, setPanelMode] = useState<SCIMGroupMappingPanelMode>("create");
  const [selectedMapping, setSelectedMapping] = useState<SCIMGroupRoleMapping | null>(null);
  const [panelOpen, setPanelOpen] = useState(false);
  const mappingsQuery = useQuery({
    queryKey: ["scim-mappings", organizationId, directoryId],
    queryFn: async () =>
      apiService.organizationService.listSCIMGroupRoleMappings(organizationId, directoryId),
    enabled: Boolean(directoryId),
  });
  const { mutate: removeMapping } = useMutation({
    mutationFn: async (mappingId: string) =>
      apiService.organizationService.deleteSCIMGroupRoleMapping(
        organizationId,
        directoryId,
        mappingId,
      ),
    onSuccess: async () => {
      toast.success("Group mapping removed");
      await onProvisioningChange();
    },
  });

  const addMapping = useCallback(() => {
    setPanelMode("create");
    setSelectedMapping(emptyMapping);
    setPanelOpen(true);
  }, []);

  const editMapping = useCallback((mapping: SCIMGroupRoleMapping) => {
    setPanelMode("edit");
    setSelectedMapping(mapping);
    setPanelOpen(true);
  }, []);

  const deleteMapping = useCallback(
    (mappingId: string) => removeMapping(mappingId),
    [removeMapping],
  );

  return (
    <>
      <MappingsPanel
        mappings={mappingsQuery.data ?? []}
        isLoading={mappingsQuery.isLoading}
        onAdd={addMapping}
        onEdit={editMapping}
        onDelete={deleteMapping}
        disabled={!directoryId}
      />
      <SCIMGroupMappingPanel
        organizationId={organizationId}
        directoryId={directoryId}
        mode={panelMode}
        open={panelOpen}
        mapping={selectedMapping}
        onOpenChange={setPanelOpen}
        onSaved={onProvisioningChange}
      />
    </>
  );
});

const MappingsPanel = memo(function MappingsPanel({
  mappings,
  isLoading,
  onAdd,
  onEdit,
  onDelete,
  disabled,
}: {
  mappings: SCIMGroupRoleMapping[];
  isLoading: boolean;
  onAdd: () => void;
  onEdit: (mapping: SCIMGroupRoleMapping) => void;
  onDelete: (mappingId: string) => void;
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
      <div className="p-0">
        {isLoading ? (
          <RowSkeleton rows={2} />
        ) : mappings.length > 0 ? (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>External group</TableHead>
                  <TableHead>Display name</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead className="w-36">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mappings.map((mapping) => (
                  <TableRow key={mapping.id}>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                        {mapping.externalGroupId}
                      </code>
                    </TableCell>
                    <TableCell>{mapping.displayName || "-"}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{mapping.roleId}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button size="sm" variant="outline" onClick={() => onEdit(mapping)}>
                          Edit
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => onDelete(mapping.id)}
                        >
                          <Trash2Icon />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={<UsersRoundIcon />}
            label="No group mappings"
            description="Map directory groups to roles before enabling automated access assignment."
            compact
          />
        )}
      </div>
    </div>
  );
});
