import { searchParamsParser } from "@/hooks/use-organization-setting-state";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { SCIMDirectory } from "@/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { AuditTimeline } from "./provisioning/audit-timeline";
import { emptyDirectory } from "./provisioning/constants";
import { DirectoryDetailHeader } from "./provisioning/directory-detail-header";
import { SCIMDirectoryPanel, type SCIMDirectoryPanelMode } from "./provisioning/directory-panel";
import { DirectoryRail } from "./provisioning/directory-rail";
import { MappingsPanelController } from "./provisioning/mappings-panel";
import { SCIMTokenPanel } from "./provisioning/scim-token-panel";

export function ProvisioningTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const auditQuery = useQuery(queries.organization.provisioningAudit(organizationId));
  const [directoryPanelMode, setDirectoryPanelMode] = useState<SCIMDirectoryPanelMode>("create");
  const [directoryPanelOpen, setDirectoryPanelOpen] = useState(false);
  const [editingDirectory, setEditingDirectory] = useState<SCIMDirectory | null>(null);
  const [directories, setDirectories] = useState<SCIMDirectory[]>([]);
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);

  const directoryId = searchParams.directoryId || directories[0]?.id || "";
  const selectedDirectory = directories.find((directory) => directory.id === directoryId);

  useEffect(() => {
    if (
      searchParams.tab === "security" &&
      searchParams.securityTab === "provisioning" &&
      !searchParams.directoryId &&
      directories[0]?.id
    ) {
      void setSearchParams({ directoryId: directories[0].id });
    }
  }, [
    directories,
    searchParams.directoryId,
    searchParams.securityTab,
    searchParams.tab,
    setSearchParams,
  ]);

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
      void setSearchParams({ directoryId: saved.id });
      await invalidateProvisioning();
    },
    [invalidateProvisioning, setSearchParams],
  );

  const handleDirectoryDeleted = useCallback(
    async (deletedDirectoryId: string) => {
      const remainingDirectory = directories.find(
        (directory) => directory.id !== deletedDirectoryId,
      );
      void setSearchParams({ directoryId: remainingDirectory?.id ?? null });
      await invalidateProvisioning();
    },
    [directories, invalidateProvisioning, setSearchParams],
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
    <div className="grid gap-3 xl:grid-cols-[320px_minmax(0,1fr)]">
      <div className="h-full">
        <DirectoryRail
          organizationId={organizationId}
          onAdd={addDirectory}
          onDirectoriesChange={setDirectories}
        />
      </div>

      <div className="min-w-0 space-y-3">
        <DirectoryDetailHeader
          directory={selectedDirectory}
          isDeleting={isDeletingDirectory}
          onEdit={editDirectory}
          onDelete={deleteDirectory}
        />
        <SCIMTokenPanel
          organizationId={organizationId}
          directoryId={directoryId}
          onProvisioningChange={invalidateProvisioning}
        />
        <MappingsPanelController
          organizationId={organizationId}
          directoryId={directoryId}
          onProvisioningChange={invalidateProvisioning}
        />
        <AuditTimeline records={auditQuery.data ?? []} isLoading={auditQuery.isLoading} />
      </div>

      <SCIMDirectoryPanel
        organizationId={organizationId}
        mode={directoryPanelMode}
        open={directoryPanelOpen}
        directory={editingDirectory}
        onOpenChange={setDirectoryPanelOpen}
        onSaved={handleDirectorySaved}
      />
    </div>
  );
}
