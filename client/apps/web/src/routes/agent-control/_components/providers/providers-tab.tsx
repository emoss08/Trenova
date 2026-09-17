import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyState } from "@/components/empty-state";
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
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import { apiService } from "@/services/api";
import type { PanelMode } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, CloudIcon, CpuIcon, PlugZapIcon, PlusIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { AIProviderPanel } from "./ai-provider-panel";
import { ProviderCard } from "./provider-card";
import { toProviderPanelRow, type ProviderPanelRow } from "./provider-form-schema";

type PanelState = {
  open: boolean;
  mode: PanelMode;
  row: ProviderPanelRow | null;
};

export default function ProvidersTab() {
  const t = useT();
  const queryClient = useQueryClient();

  const listQuery = useQuery(queries.aiProvider.list());
  const catalogQuery = useQuery(queries.aiProvider.catalog());

  const { allowed: canCreate } = usePermission(Resource.AIProvider, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.AIProvider, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.AIProvider, Operation.Delete);
  const { allowed: canManage } = usePermission(Resource.AIProvider, Operation.Manage);

  const [panel, setPanel] = useState<PanelState>({ open: false, mode: "create", row: null });
  const [deleting, setDeleting] = useState<AIProviderRow | null>(null);

  const providers = listQuery.data ?? [];

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queries.aiProvider._def });
  }, [queryClient]);

  const testMutation = useApiMutation({
    mutationFn: (id: string) => apiService.aiProviderService.test(id),
    onSuccess: async (result) => {
      if (result.success && result.schemaHonoured) {
        toast.success(result.message);
      } else if (result.success) {
        toast.warning(result.message, { description: result.detail || undefined });
      } else {
        toast.error(result.message, { description: result.detail || undefined });
      }
      await invalidate();
    },
    resourceName: "AI Provider",
  });

  const deleteMutation = useApiMutation({
    mutationFn: (id: string) => apiService.aiProviderService.remove(id),
    onSuccess: async () => {
      toast.success(t("AI provider removed"));
      setDeleting(null);
      await invalidate();
    },
    resourceName: "AI Provider",
  });

  const openCreate = useCallback(() => setPanel({ open: true, mode: "create", row: null }), []);
  const openEdit = useCallback(
    (provider: AIProviderRow) =>
      setPanel({ open: true, mode: "edit", row: toProviderPanelRow(provider) }),
    [],
  );

  const enabledCount = providers.filter((provider) => provider.enabled).length;

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-base font-semibold">
            {t("Providers")}
            {providers.length > 0 && (
              <span className="bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-[11px] font-medium tabular-nums">
                {t("{0} of {1} enabled", enabledCount, providers.length)}
              </span>
            )}
          </h2>
          <p className="text-muted-foreground max-w-prose text-sm">
            {t(
              "Where AI work goes. Work is offered to providers in priority order, so a cheap model can take a task first and hand off when it cannot.",
            )}
          </p>
        </div>
        {canCreate && providers.length > 0 && (
          <Button size="sm" onClick={openCreate}>
            <PlusIcon className="size-3.5" />
            {t("Add provider")}
          </Button>
        )}
      </div>

      {listQuery.isLoading ? (
        <div className="grid gap-3 md:grid-cols-2 2xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-44" />
          ))}
        </div>
      ) : providers.length === 0 ? (
        <div className="flex justify-center py-6">
          <EmptyState
            icons={[CloudIcon, PlugZapIcon, CpuIcon]}
            title={t("No AI providers yet")}
            description={t(
              "Connect a hosted API, a gateway such as OpenRouter, or a model server on your own hardware. AI features stay off until a provider is assigned to a task.",
            )}
            action={
              canCreate
                ? { icon: PlusIcon, label: t("Add your first provider"), onClick: openCreate }
                : undefined
            }
          />
        </div>
      ) : (
        <div className="grid gap-3 md:grid-cols-2 2xl:grid-cols-3">
          {providers.map((provider) => (
            <ProviderCard
              key={provider.id}
              provider={provider}
              catalog={catalogQuery.data}
              isTesting={testMutation.isPending && testMutation.variables === provider.id}
              canManage={canManage}
              canUpdate={canUpdate}
              canDelete={canDelete}
              onTest={() => testMutation.mutate(provider.id)}
              onEdit={() => openEdit(provider)}
              onDelete={() => setDeleting(provider)}
            />
          ))}
        </div>
      )}

      <AIProviderPanel
        open={panel.open}
        onOpenChange={(open) => setPanel((current) => ({ ...current, open }))}
        mode={panel.mode}
        row={panel.row}
      />

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <AlertTriangleIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Remove AI provider")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Remove {0}. Any task routed only to this provider will stop working until another one is assigned.",
                deleting?.name ?? t("this provider"),
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => deleting && deleteMutation.mutate(deleting.id)}
              disabled={deleteMutation.isPending}
            >
              {t("Remove provider")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
