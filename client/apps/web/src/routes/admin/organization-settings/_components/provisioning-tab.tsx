import { directoryIdParser } from "@/hooks/use-organization-setting-state";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { SCIMDirectory } from "@trenova/shared/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { UsersRoundIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { AuditTimeline } from "./provisioning/audit-timeline";
import { emptyDirectory } from "./provisioning/constants";
import { DirectoryDetailHeader } from "./provisioning/directory-detail-header";
import { SCIMDirectoryPanel, type SCIMDirectoryPanelMode } from "./provisioning/directory-panel";
import { DirectoryRail } from "./provisioning/directory-rail";
import { MappingsPanelController } from "./provisioning/mappings-panel";
import { SCIMTokenPanel } from "./provisioning/scim-token-panel";
import { EmptyState } from "./security-access/shared";

export function ProvisioningTab({
  organizationId,
  isActive,
}: {
  organizationId: string;
  isActive: boolean;
}) {
  const queryClient = useQueryClient();
  const auditQuery = useQuery(queries.organization.provisioningAudit(organizationId));
  const [directoryPanelMode, setDirectoryPanelMode] = useState<SCIMDirectoryPanelMode>("create");
  const [directoryPanelOpen, setDirectoryPanelOpen] = useState(false);
  const [editingDirectory, setEditingDirectory] = useState<SCIMDirectory | null>(null);
  const [directories, setDirectories] = useState<SCIMDirectory[]>([]);
  const [directoryId, setDirectoryId] = useQueryState("directoryId", directoryIdParser);

  const selectedDirectory = directories.find((directory) => directory.id === directoryId);

  useEffect(() => {
    if (isActive && !directoryId && directories[0]?.id) {
      void setDirectoryId(directories[0].id);
    }
  }, [directories, directoryId, isActive, setDirectoryId]);

  const invalidateProvisioning = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: queries.organization.scimDirectories(organizationId).queryKey,
      }),
      queryClient.invalidateQueries({
        queryKey: ["scim-tokens", organizationId, directoryId],
      }),
      queryClient.invalidateQueries({
        queryKey: ["scim-mappings", organizationId, directoryId],
      }),
      queryClient.invalidateQueries({
        queryKey: queries.organization.provisioningAudit(organizationId).queryKey,
      }),
    ]);
  }, [directoryId, organizationId, queryClient]);

  const handleDirectorySaved = useCallback(
    async (saved: SCIMDirectory) => {
      void setDirectoryId(saved.id);
      await invalidateProvisioning();
    },
    [invalidateProvisioning, setDirectoryId],
  );

  const handleDirectoryDeleted = useCallback(
    async (deletedDirectoryId: string) => {
      const remainingDirectory = directories.find(
        (directory) => directory.id !== deletedDirectoryId,
      );
      void setDirectoryId(remainingDirectory?.id ?? null);
      await invalidateProvisioning();
    },
    [directories, invalidateProvisioning, setDirectoryId],
  );

  const { mutate: deleteDirectory, isPending: isDeletingDirectory } = useMutation({
    mutationFn: async (directoryId: string) =>
      apiService.organizationService.deleteSCIMDirectory(organizationId, directoryId),
    onSuccess: async (_data, deletedDirectoryId) => {
      toast.success("SCIM directory removed");
      await handleDirectoryDeleted(deletedDirectoryId);
    },
  });

  const addDirectory = useCallback(() => {
    setDirectoryPanelMode("create");
    setEditingDirectory(emptyDirectory);
    setDirectoryPanelOpen(true);
  }, []);

  const editDirectory = useCallback((directory: SCIMDirectory) => {
    setDirectoryPanelMode("edit");
    setEditingDirectory(directory);
    setDirectoryPanelOpen(true);
  }, []);

  return (
    <TabOuter>
      <RailOuter>
        <DirectoryRail
          organizationId={organizationId}
          onAdd={addDirectory}
          onDirectoriesChange={setDirectories}
        />
      </RailOuter>

      <ContentOuter>
        <DirectoryDetailHeader
          directory={selectedDirectory}
          isDeleting={isDeletingDirectory}
          onEdit={editDirectory}
          onDelete={deleteDirectory}
        />
        {selectedDirectory ? (
          <>
            <SCIMTokenPanel
              organizationId={organizationId}
              directoryId={selectedDirectory.id}
              onProvisioningChange={invalidateProvisioning}
            />
            <MappingsPanelController
              organizationId={organizationId}
              directoryId={selectedDirectory.id}
              onProvisioningChange={invalidateProvisioning}
            />
          </>
        ) : (
          <EmptyState
            icon={<UsersRoundIcon />}
            label="Select a directory"
            description="Choose or create a SCIM directory before managing tokens and group mappings."
          />
        )}
        <AuditTimeline records={auditQuery.data ?? []} isLoading={auditQuery.isLoading} />
      </ContentOuter>
      <SCIMDirectoryPanel
        organizationId={organizationId}
        mode={directoryPanelMode}
        open={directoryPanelOpen}
        directory={editingDirectory}
        onOpenChange={setDirectoryPanelOpen}
        onSaved={handleDirectorySaved}
      />
    </TabOuter>
  );
}

function ContentOuter({ children }: { children: React.ReactNode }) {
  return <div className="min-w-0 space-y-3">{children}</div>;
}

function TabOuter({ children }: { children: React.ReactNode }) {
  return <div className="grid gap-3 xl:grid-cols-[320px_minmax(0,1fr)]">{children}</div>;
}

function RailOuter({ children }: { children: React.ReactNode }) {
  return <div className="h-full">{children}</div>;
}
