import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { ApiKey, ApiKeyPermissionInput, CreateApiKeyRequest } from "@/types/api-key";
import type { DataTablePanelProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  CopyIcon,
  KeyRoundIcon,
  RefreshCwIcon,
  ShieldAlertIcon,
  ShieldXIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { APIKeyForm } from "./api-key-form";
import { APIKeyPermissionsEditor } from "./api-key-permissions-editor";

const apiKeyPanelSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  description: z.string(),
  expiresAtInput: z.string(),
  permissions: z
    .array(
      z.object({
        resource: z.string(),
        operations: z.array(z.string()).min(1),
        dataScope: z.string(),
      }),
    )
    .min(1, "At least one permission is required"),
});

export type ApiKeyPanelFormValues = z.infer<typeof apiKeyPanelSchema>;

export function APIKeyPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<ApiKey>) {
  if (mode === "edit" && row) {
    return <APIKeyEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }

  return <APIKeyCreatePanel open={open} onOpenChange={onOpenChange} />;
}

type CreatePanelProps = Pick<DataTablePanelProps<ApiKey>, "open" | "onOpenChange">;

function APIKeyCreatePanel({ open, onOpenChange }: CreatePanelProps) {
  const queryClient = useQueryClient();
  const [successToken, setSuccessToken] = useState("");
  const [successDialogOpen, setSuccessDialogOpen] = useState(false);

  const form = useForm<ApiKeyPanelFormValues, unknown, ApiKeyPanelFormValues>({
    resolver: zodResolver(apiKeyPanelSchema),
    defaultValues: getDefaultValues(),
  });

  const {
    setError,
    reset,
    formState: { isSubmitting },
  } = form;

  const handleClose = useCallback(() => {
    reset(getDefaultValues());
    onOpenChange(false);
  }, [onOpenChange, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: ApiKeyPanelFormValues) =>
      apiService.apiKeyService.create(toRequestPayload(values)),
    onSuccess: async (result) => {
      toast.success("API key created");
      await queryClient.invalidateQueries({ queryKey: ["api-key-list"] });
      reset(getDefaultValues());
      onOpenChange(false);
      setSuccessToken(result.token);
      setSuccessDialogOpen(true);
    },
    setFormError: setError,
    resourceName: "API Key",
  });

  const onSubmit = useCallback(
    async (values: ApiKeyPanelFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    if (!open) {
      reset(getDefaultValues());
    }
  }, [open, reset]);

  const handleSuccessDialogOpenChange = useCallback((nextOpen: boolean) => {
    setSuccessDialogOpen(nextOpen);
    if (!nextOpen) {
      setSuccessToken("");
    }
  }, []);

  return (
    <>
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title="Create API Key"
        description="Define the bearer credential and grant only the resources the integration needs."
        size="xl"
        footer={
          <>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="api-key-create-form"
              isLoading={isSubmitting}
              loadingText="Creating..."
            >
              Create API Key
            </Button>
          </>
        }
      >
        <FormProvider {...form}>
          <Form id="api-key-create-form" onSubmit={form.handleSubmit(onSubmit)}>
            <fieldset className="space-y-6">
              <APIKeyForm />
              <div className="border-t border-border/70 pt-6">
                <APIKeyPermissionsEditor />
              </div>
            </fieldset>
          </Form>
        </FormProvider>
      </DataTablePanelContainer>
      <TokenSuccessDialog
        open={successDialogOpen}
        onOpenChange={handleSuccessDialogOpenChange}
        token={successToken}
      />
    </>
  );
}

type EditPanelProps = Pick<DataTablePanelProps<ApiKey>, "open" | "onOpenChange" | "row"> & {
  row: ApiKey;
};

function APIKeyEditPanel({ open, onOpenChange, row }: EditPanelProps) {
  const queryClient = useQueryClient();
  const [successToken, setSuccessToken] = useState("");
  const [successDialogOpen, setSuccessDialogOpen] = useState(false);
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false);

  const form = useForm<ApiKeyPanelFormValues, unknown, ApiKeyPanelFormValues>({
    resolver: zodResolver(apiKeyPanelSchema),
    defaultValues: getDefaultValues(),
  });

  const {
    setError,
    reset,
    formState: { isSubmitting },
  } = form;

  const detailQuery = useQuery({
    ...queries.integration.apiKey(row.id),
    enabled: open,
  });

  useEffect(() => {
    if (detailQuery.data) {
      reset(toFormValues(detailQuery.data));
    }
  }, [detailQuery.data, reset]);

  useEffect(() => {
    if (!open) {
      setRevokeDialogOpen(false);
    }
  }, [open]);

  const handleSuccessDialogOpenChange = useCallback((nextOpen: boolean) => {
    setSuccessDialogOpen(nextOpen);
    if (!nextOpen) {
      setSuccessToken("");
    }
  }, []);

  const invalidateData = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["api-key-list"] }),
      queryClient.invalidateQueries({
        queryKey: queries.integration.apiKey(row.id).queryKey,
      }),
    ]);
  }, [queryClient, row.id]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: ApiKeyPanelFormValues) =>
      apiService.apiKeyService.update(row.id, toRequestPayload(values)),
    onSuccess: async () => {
      toast.success("API key updated");
      await invalidateData();
      reset(getDefaultValues());
      onOpenChange(false);
    },
    setFormError: setError,
    resourceName: "API Key",
  });

  const rotateMutation = useApiMutation({
    mutationFn: async () => apiService.apiKeyService.rotate(row.id),
    onSuccess: async (result) => {
      toast.success("API key rotated");
      await invalidateData();
      onOpenChange(false);
      setSuccessToken(result.token);
      setSuccessDialogOpen(true);
    },
    resourceName: "API Key",
  });

  const revokeMutation = useApiMutation({
    mutationFn: async () => apiService.apiKeyService.revoke(row.id),
    onSuccess: async () => {
      toast.success("API key revoked");
      setRevokeDialogOpen(false);
      await invalidateData();
      onOpenChange(false);
    },
    resourceName: "API Key",
  });

  const onSubmit = useCallback(
    async (values: ApiKeyPanelFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const detail = detailQuery.data ?? row;
  const isRevoked = detail.status === "revoked";

  const panelDescription = useMemo(() => {
    if (isRevoked) {
      return "This credential has been revoked and can no longer authenticate requests.";
    }
    return "Update metadata, refine resource grants, or rotate the bearer secret.";
  }, [isRevoked]);

  const headerActions = useMemo(() => {
    if (detailQuery.isLoading) return null;

    return (
      <>
        {!isRevoked && (
          <>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="text-muted-foreground hover:text-foreground"
                    onClick={() => rotateMutation.mutate(undefined)}
                    disabled={rotateMutation.isPending}
                  >
                    <RefreshCwIcon className="size-4" />
                  </Button>
                }
              />
              <TooltipContent>Rotate Secret</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="text-muted-foreground hover:text-destructive"
                    onClick={() => setRevokeDialogOpen(true)}
                    disabled={revokeMutation.isPending}
                  >
                    <ShieldXIcon className="size-4" />
                  </Button>
                }
              />
              <TooltipContent>Revoke Key</TooltipContent>
            </Tooltip>
          </>
        )}
      </>
    );
  }, [isRevoked, rotateMutation, revokeMutation, detailQuery.isLoading]);

  return (
    <>
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={detail.name}
        description={panelDescription}
        size="xl"
        headerActions={headerActions}
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="api-key-edit-form"
              isLoading={isSubmitting}
              loadingText="Saving..."
              disabled={isRevoked}
            >
              Save Changes
            </Button>
          </>
        }
      >
        <div className="space-y-6">
          {detailQuery.isLoading ? (
            <div className="space-y-6">
              <div className="space-y-4">
                <Skeleton className="h-5 w-24" />
                <div className="grid grid-cols-2 gap-4">
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-10 w-full" />
                </div>
                <Skeleton className="h-20 w-full" />
              </div>
              <div className="space-y-3">
                <Skeleton className="h-5 w-28" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-40 w-full" />
              </div>
            </div>
          ) : (
            <>
              {isRevoked && (
                <div className="flex items-center gap-3 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3">
                  <ShieldAlertIcon className="size-4 shrink-0 text-destructive" />
                  <p className="text-sm text-destructive">
                    This API key has been revoked. All fields are read-only.
                  </p>
                </div>
              )}
              <FormProvider {...form}>
                <Form id="api-key-edit-form" onSubmit={form.handleSubmit(onSubmit)}>
                  <fieldset disabled={isRevoked} className="space-y-6">
                    <APIKeyForm />
                    <div className="border-t border-border/70 pt-6">
                      <APIKeyPermissionsEditor />
                    </div>
                  </fieldset>
                </Form>
              </FormProvider>
            </>
          )}
        </div>
      </DataTablePanelContainer>
      <TokenSuccessDialog
        open={successDialogOpen}
        onOpenChange={handleSuccessDialogOpenChange}
        token={successToken}
      />
      <AlertDialog open={revokeDialogOpen} onOpenChange={setRevokeDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <ShieldAlertIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Revoke API Key</AlertDialogTitle>
            <AlertDialogDescription>
              Revoke this bearer credential immediately. Existing integrations will stop
              authenticating until a new key is provisioned.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => revokeMutation.mutate(undefined)}
              disabled={revokeMutation.isPending}
            >
              Revoke Key
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function TokenSuccessDialog({
  open,
  onOpenChange,
  token,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  token: string;
}) {
  const { copy, isCopied } = useCopyToClipboard();

  if (!token) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <KeyRoundIcon className="size-4" />
            Copy this API key now
          </DialogTitle>
          <DialogDescription>
            This plaintext token is only shown once after create or rotate.
          </DialogDescription>
        </DialogHeader>
        <pre className="overflow-x-auto rounded-md border border-border/70 bg-muted/30 p-4 font-mono text-xs">
          {token}
        </pre>
        <div className="flex items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3">
          <AlertTriangleIcon className="size-4 shrink-0 text-amber-500" />
          <p className="text-sm text-amber-700 dark:text-amber-400">
            Store this key securely. It will not be displayed again.
          </p>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <Button type="button" onClick={() => copy(token, { withToast: true })}>
            {isCopied ? <CheckIcon className="size-4" /> : <CopyIcon className="size-4" />}
            {isCopied ? "Copied" : "Copy API Key"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function getDefaultValues(): ApiKeyPanelFormValues {
  return {
    name: "",
    description: "",
    expiresAtInput: "",
    permissions: [],
  };
}

function toFormValues(key: ApiKey): ApiKeyPanelFormValues {
  return {
    name: key.name,
    description: key.description ?? "",
    expiresAtInput: key.expiresAt ? unixToInputValue(key.expiresAt) : "",
    permissions: key.permissions ?? [],
  };
}

function toRequestPayload(values: ApiKeyPanelFormValues): CreateApiKeyRequest {
  return {
    name: values.name.trim(),
    description: values.description?.trim() ?? "",
    expiresAt: values.expiresAtInput
      ? Math.floor(new Date(values.expiresAtInput).getTime() / 1000)
      : 0,
    permissions: values.permissions.map(
      (permission): ApiKeyPermissionInput => ({
        resource: permission.resource,
        operations: permission.operations,
        dataScope: permission.dataScope,
      }),
    ),
  };
}

function unixToInputValue(value: number) {
  const date = new Date(value * 1000);
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}
