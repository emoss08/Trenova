import { RoleSelectAutocompleteField } from "@/components/autocomplete-fields";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { formatUnixDateTimeOrDash } from "@/lib/date";
import { queries } from "@/lib/queries";
import { cn, toTitleCase } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { ProvisioningAuditRecord, SCIMDirectory, SCIMGroupRoleMapping } from "@/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ActivityIcon,
  CheckIcon,
  ClipboardIcon,
  KeyRoundIcon,
  PlusIcon,
  SaveIcon,
  Trash2Icon,
  UsersRoundIcon,
} from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import {
  ActivityItem,
  EmptyState,
  ErrorState,
  Field,
  PanelHeader,
  RowSkeleton,
  ToggleRow,
} from "./security-access/shared";

type MappingFormValues = {
  externalGroupId: string;
  displayName: string;
  roleId: string;
};

type DirectoryFormValues = {
  tenantSlug: string;
  enabled: boolean;
};

const emptyDirectory: SCIMDirectory = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  tenantSlug: "",
  enabled: true,
  createdAt: 0,
  updatedAt: 0,
};

const emptyMapping: SCIMGroupRoleMapping = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  directoryId: "",
  externalGroupId: "",
  displayName: "",
  roleId: "",
  createdAt: 0,
  updatedAt: 0,
};

export function ProvisioningTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const directoriesQuery = useQuery(queries.organization.scimDirectories(organizationId));
  const auditQuery = useQuery(queries.organization.provisioningAudit(organizationId));
  const [editingDirectory, setEditingDirectory] = useState<SCIMDirectory>(emptyDirectory);
  const [directorySheetOpen, setDirectorySheetOpen] = useState(false);
  const [selectedDirectoryId, setSelectedDirectoryId] = useState("");

  const directories = useMemo(() => directoriesQuery.data ?? [], [directoriesQuery.data]);
  const directoryId = selectedDirectoryId || directories[0]?.id || "";
  const selectedDirectory = directories.find((item) => item.id === directoryId);

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
      setDirectorySheetOpen(false);
      setSelectedDirectoryId(saved.id);
      await invalidateProvisioning();
    },
    [invalidateProvisioning],
  );

  const openDirectorySheet = useCallback((value: SCIMDirectory) => {
    setEditingDirectory(value);
    setDirectorySheetOpen(true);
  }, []);

  return (
    <div className="grid gap-3 xl:grid-cols-[320px_minmax(0,1fr)]">
      <div className="space-y-3">
        <div className="rounded-lg border bg-background">
          <div className="flex items-center justify-between border-b p-3">
            <div>
              <div className="text-sm font-medium">SCIM directories</div>
              <div className="text-xs text-muted-foreground">Directory sync tenants</div>
            </div>
            <Button size="sm" onClick={() => openDirectorySheet(emptyDirectory)}>
              <PlusIcon />
              Add
            </Button>
          </div>
          {directoriesQuery.isLoading ? (
            <div className="space-y-2 p-3">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : directoriesQuery.isError ? (
            <ErrorState label="SCIM directories could not be loaded." compact />
          ) : directories.length > 0 ? (
            <div className="divide-y">
              {directories.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={cn(
                    "flex w-full items-center justify-between gap-3 px-3 py-3 text-left transition-colors hover:bg-muted/40",
                    item.id === directoryId && "bg-muted/60",
                  )}
                  onClick={() => setSelectedDirectoryId(item.id)}
                >
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium">{item.tenantSlug}</div>
                    <div className="text-xs text-muted-foreground">
                      Updated {formatUnixDateTimeOrDash(item.updatedAt || item.createdAt)}
                    </div>
                  </div>
                  <Badge variant={item.enabled ? "active" : "inactive"}>
                    {item.enabled ? "Enabled" : "Disabled"}
                  </Badge>
                </button>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<UsersRoundIcon />}
              label="No directories"
              description="Create a SCIM directory before issuing tokens or mapping groups."
              compact
            />
          )}
        </div>
      </div>

      <div className="min-w-0 space-y-3">
        <div className="rounded-lg border bg-background p-3">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="truncate text-base font-semibold tracking-tight">
                  {selectedDirectory?.tenantSlug || "Select a directory"}
                </h3>
                {selectedDirectory && (
                  <Badge variant={selectedDirectory.enabled ? "active" : "inactive"}>
                    {selectedDirectory.enabled ? "Enabled" : "Disabled"}
                  </Badge>
                )}
              </div>
              <p className="text-sm text-muted-foreground">
                Manage SCIM tokens, group-to-role mappings, and provisioning audit events.
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={!selectedDirectory}
              onClick={() => selectedDirectory && openDirectorySheet(selectedDirectory)}
            >
              Edit directory
            </Button>
          </div>
        </div>

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

      <DirectoryEditorSheet
        organizationId={organizationId}
        directory={editingDirectory}
        open={directorySheetOpen}
        onOpenChange={setDirectorySheetOpen}
        onSaved={handleDirectorySaved}
      />
    </div>
  );
}

function DirectoryEditorSheet({
  organizationId,
  directory,
  open,
  onOpenChange,
  onSaved,
}: {
  organizationId: string;
  directory: SCIMDirectory;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: (directory: SCIMDirectory) => Promise<void>;
}) {
  const saveDirectoryMutation = useMutation({
    mutationFn: async (value: SCIMDirectory) =>
      value.id
        ? apiService.organizationService.updateSCIMDirectory(organizationId, value)
        : apiService.organizationService.createSCIMDirectory(organizationId, value),
    onSuccess: async (saved) => {
      toast.success("SCIM directory saved");
      await onSaved(saved);
    },
  });

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[calc(100vw-1rem)] sm:max-w-md">
        <SheetHeader className="border-b">
          <SheetTitle>{directory.id ? "Edit SCIM directory" : "Add SCIM directory"}</SheetTitle>
          <SheetDescription>
            Configure the tenant slug and provisioning availability.
          </SheetDescription>
        </SheetHeader>
        <DirectoryEditorForm
          key={`${directory.id || "new"}-${open ? "open" : "closed"}`}
          directory={directory}
          isSaving={saveDirectoryMutation.isPending}
          onSave={(value) => saveDirectoryMutation.mutate(value)}
        />
      </SheetContent>
    </Sheet>
  );
}

function DirectoryEditorForm({
  directory,
  isSaving,
  onSave,
}: {
  directory: SCIMDirectory;
  isSaving: boolean;
  onSave: (directory: SCIMDirectory) => void;
}) {
  const { handleSubmit, register, setValue, watch } = useForm<DirectoryFormValues>({
    defaultValues: {
      tenantSlug: directory.tenantSlug,
      enabled: directory.enabled,
    },
  });

  return (
    <form
      className="flex min-h-0 flex-1 flex-col"
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit((values) => onSave({ ...directory, ...values }))(event);
      }}
    >
      <div className="space-y-3 px-4">
        <Field label="Tenant slug">
          <Input {...register("tenantSlug", { required: true })} />
        </Field>
        <ToggleRow
          label="Enabled"
          description="Allow SCIM API calls for this directory."
          checked={watch("enabled")}
          onCheckedChange={(enabled) =>
            setValue("enabled", enabled, { shouldDirty: true, shouldValidate: true })
          }
        />
      </div>
      <SheetFooter className="border-t">
        <Button type="submit" size="sm" isLoading={isSaving} loadingText="Saving...">
          <SaveIcon />
          Save directory
        </Button>
      </SheetFooter>
    </form>
  );
}

const SCIMTokenPanel = memo(function SCIMTokenPanel({
  organizationId,
  directoryId,
  onProvisioningChange,
}: {
  organizationId: string;
  directoryId: string;
  onProvisioningChange: () => Promise<void>;
}) {
  const [tokenName, setTokenName] = useState("");
  const [createdToken, setCreatedToken] = useState("");
  const tokensQuery = useQuery({
    queryKey: ["scim-tokens", organizationId, directoryId],
    queryFn: async () => apiService.organizationService.listSCIMTokens(organizationId, directoryId),
    enabled: Boolean(directoryId),
  });
  const { mutate: createToken, isPending: isCreatingToken } = useMutation({
    mutationFn: async () =>
      apiService.organizationService.createSCIMToken(organizationId, directoryId, tokenName),
    onSuccess: async (response) => {
      setCreatedToken(response.token);
      setTokenName("");
      toast.success("SCIM token created");
      await onProvisioningChange();
    },
  });
  const { mutate: revokeToken } = useMutation({
    mutationFn: async (tokenId: string) =>
      apiService.organizationService.revokeSCIMToken(organizationId, tokenId),
    onSuccess: async () => {
      toast.success("SCIM token revoked");
      await onProvisioningChange();
    },
  });
  const tokens = tokensQuery.data ?? [];
  const createDisabled = !directoryId || isCreatingToken;

  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<KeyRoundIcon />}
        title="SCIM tokens"
        description="Issue bearer tokens for directory synchronization."
      />
      <div className="space-y-3 p-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            value={tokenName}
            placeholder="Token name"
            onChange={(event) => setTokenName(event.target.value)}
          />
          <Button
            size="sm"
            onClick={() => createToken()}
            disabled={createDisabled || tokenName.trim() === ""}
          >
            <PlusIcon />
            Create token
          </Button>
        </div>
        {createdToken && <CopyableSecretBlock value={createdToken} />}
        {tokensQuery.isLoading ? (
          <RowSkeleton rows={2} />
        ) : tokens.length > 0 ? (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Prefix</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last used</TableHead>
                  <TableHead className="w-28">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell className="font-medium">{token.name}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{token.prefix}</code>
                    </TableCell>
                    <TableCell>
                      <Badge variant={token.status === "active" ? "active" : "inactive"}>
                        {toTitleCase(token.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {token.lastUsedAt ? formatUnixDateTimeOrDash(token.lastUsedAt) : "Never"}
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={token.status !== "active"}
                        onClick={() => revokeToken(token.id)}
                      >
                        Revoke
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={<KeyRoundIcon />}
            label="No SCIM tokens"
            description="Create a token and copy it into your directory sync application."
            compact
          />
        )}
      </div>
    </div>
  );
});

function CopyableSecretBlock({ value }: { value: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="rounded-lg border border-amber-600/30 bg-amber-600/10 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="text-sm font-medium text-amber-800 dark:text-amber-300">
            Copy this token now
          </div>
          <div className="text-xs text-amber-700/80 dark:text-amber-300/80">
            The plaintext token is only shown once.
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={() => void copy(value, { withToast: true })}>
          {isCopied ? <CheckIcon /> : <ClipboardIcon />}
          {isCopied ? "Copied" : "Copy"}
        </Button>
      </div>
      <code className="block rounded-md border bg-background/80 p-2 font-mono text-xs break-all">
        {value}
      </code>
    </div>
  );
}

const MappingsPanelController = memo(function MappingsPanelController({
  organizationId,
  directoryId,
  onProvisioningChange,
}: {
  organizationId: string;
  directoryId: string;
  onProvisioningChange: () => Promise<void>;
}) {
  const [mapping, setMapping] = useState<SCIMGroupRoleMapping>(emptyMapping);
  const [mappingSheetOpen, setMappingSheetOpen] = useState(false);
  const mappingsQuery = useQuery({
    queryKey: ["scim-mappings", organizationId, directoryId],
    queryFn: async () =>
      apiService.organizationService.listSCIMGroupRoleMappings(organizationId, directoryId),
    enabled: Boolean(directoryId),
  });
  const { mutate: saveMapping, isPending: isSavingMapping } = useMutation({
    mutationFn: async (value: SCIMGroupRoleMapping) =>
      value.id
        ? apiService.organizationService.updateSCIMGroupRoleMapping(
            organizationId,
            directoryId,
            value,
          )
        : apiService.organizationService.createSCIMGroupRoleMapping(
            organizationId,
            directoryId,
            value,
          ),
    onSuccess: async () => {
      toast.success("Group mapping saved");
      setMapping(emptyMapping);
      setMappingSheetOpen(false);
      await onProvisioningChange();
    },
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
  const openMappingSheet = useCallback((value: SCIMGroupRoleMapping) => {
    setMapping(value);
    setMappingSheetOpen(true);
  }, []);
  const addMapping = useCallback(() => openMappingSheet(emptyMapping), [openMappingSheet]);
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
        onEdit={openMappingSheet}
        onDelete={deleteMapping}
        disabled={!directoryId}
      />
      <Sheet open={mappingSheetOpen} onOpenChange={setMappingSheetOpen}>
        <SheetContent className="w-[calc(100vw-1rem)] sm:max-w-md">
          <SheetHeader className="border-b">
            <SheetTitle>{mapping.id ? "Edit group mapping" : "Add group mapping"}</SheetTitle>
            <SheetDescription>Map an external SCIM group to a Trenova role.</SheetDescription>
          </SheetHeader>
          <MappingEditor
            mapping={mapping}
            disabled={!directoryId || isSavingMapping}
            onSave={saveMapping}
          />
        </SheetContent>
      </Sheet>
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
      <div className="p-3">
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
                {mappings.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                        {item.externalGroupId}
                      </code>
                    </TableCell>
                    <TableCell>{item.displayName || "-"}</TableCell>
                    <TableCell className="text-muted-foreground">{item.roleId}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button size="sm" variant="outline" onClick={() => onEdit(item)}>
                          Edit
                        </Button>
                        <Button size="sm" variant="destructive" onClick={() => onDelete(item.id)}>
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

function MappingEditor({
  mapping,
  disabled,
  onSave,
}: {
  mapping: SCIMGroupRoleMapping;
  disabled: boolean;
  onSave: (mapping: SCIMGroupRoleMapping) => void;
}) {
  const { control, handleSubmit, register, reset } = useForm<MappingFormValues>({
    defaultValues: {
      externalGroupId: mapping.externalGroupId,
      displayName: mapping.displayName,
      roleId: mapping.roleId,
    },
  });

  useEffect(() => {
    reset({
      externalGroupId: mapping.externalGroupId,
      displayName: mapping.displayName,
      roleId: mapping.roleId,
    });
  }, [mapping, reset]);

  return (
    <form
      className="flex min-h-0 flex-1 flex-col"
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit((values) => onSave({ ...mapping, ...values }))(event);
      }}
    >
      <div className="space-y-3 px-4">
        <Field label="External group ID">
          <Input {...register("externalGroupId", { required: true })} disabled={disabled} />
        </Field>
        <Field label="Display name">
          <Input {...register("displayName")} disabled={disabled} />
        </Field>
        <RoleSelectAutocompleteField<MappingFormValues>
          control={control}
          name="roleId"
          label="Role"
          placeholder="Select role"
          clearable
          disabled={disabled}
          rules={{ required: true }}
        />
      </div>
      <SheetFooter className="border-t">
        <Button type="submit" size="sm" disabled={disabled}>
          <SaveIcon />
          Save mapping
        </Button>
      </SheetFooter>
    </form>
  );
}

const AuditTimeline = memo(function AuditTimeline({
  records,
  isLoading,
}: {
  records: ProvisioningAuditRecord[];
  isLoading: boolean;
}) {
  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<ActivityIcon />}
        title="Provisioning audit"
        description="Recent user and group synchronization events."
      />
      <div className="divide-y">
        {isLoading ? (
          <div className="space-y-2 p-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : records.length > 0 ? (
          records
            .slice(0, 8)
            .map((item) => (
              <ActivityItem
                key={item.id}
                title={`${toTitleCase(item.action)} ${item.resourceType}`}
                detail={
                  item.errorMessage || item.externalId || item.resourceId || "Provisioning event"
                }
                badge={item.status}
                when={formatUnixDateTimeOrDash(item.createdAt)}
              />
            ))
        ) : (
          <EmptyState
            icon={<ActivityIcon />}
            label="No provisioning events"
            description="SCIM activity will appear after your directory starts syncing."
            compact
          />
        )}
      </div>
    </div>
  );
});
