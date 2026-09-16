import { useT } from "@trenova/shared/i18n/use-t";
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
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { cn } from "@trenova/shared/lib/utils";
import type { AIProvider, AITaskDescriptor, TestAIProviderResult } from "@/types/ai-provider";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  PencilIcon,
  PlugZapIcon,
  PlusIcon,
  ServerIcon,
  ShieldCheckIcon,
  Trash2Icon,
  XCircleIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { AIProviderDialog } from "./ai-provider-dialog";

export function AIProviderList() {
  const t = useT();
  const queryClient = useQueryClient();

  const listQuery = useQuery(queries.aiProvider.list());
  const catalogQuery = useQuery(queries.aiProvider.catalog());

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<AIProvider | null>(null);
  const [deleting, setDeleting] = useState<AIProvider | null>(null);
  const [testResults, setTestResults] = useState<Record<string, TestAIProviderResult>>({});

  // Priority order is routing order, so the list is read the way the router
  // reads it: the provider a task reaches first is at the top.
  const providers = useMemo(
    () => [...(listQuery.data?.results ?? [])].sort((a, b) => a.priority - b.priority),
    [listQuery.data?.results],
  );
  const taskDescriptors = catalogQuery.data?.tasks ?? [];

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queries.aiProvider.list().queryKey });
  }, [queryClient]);

  const testMutation = useApiMutation({
    mutationFn: async (id: string) => {
      const result = await apiService.aiProviderService.test(id);
      return { id, result };
    },
    onSuccess: ({ id, result }) => {
      setTestResults((prev) => ({ ...prev, [id]: result }));
      if (result.success && result.schemaHonoured) {
        toast.success(result.message);
      } else if (result.success) {
        toast.warning(result.message);
      } else {
        toast.error(result.message);
      }
    },
    resourceName: "AI Provider",
  });

  const deleteMutation = useApiMutation({
    mutationFn: async (id: string) => apiService.aiProviderService.remove(id),
    onSuccess: async () => {
      toast.success(t("AI provider removed"));
      setDeleting(null);
      await invalidate();
    },
    resourceName: "AI Provider",
  });

  const openCreate = useCallback(() => {
    setEditing(null);
    setDialogOpen(true);
  }, []);

  const openEdit = useCallback((provider: AIProvider) => {
    setEditing(provider);
    setDialogOpen(true);
  }, []);

  const enabledCount = providers.filter((provider) => provider.enabled).length;

  return (
    <>
      <Card className="gap-0 p-0">
        <CardHeader className="flex flex-row items-center justify-between border-b py-3">
          <CardTitle className="flex items-center gap-1.5 text-sm font-medium">
            <PlugZapIcon className="text-muted-foreground size-3.5" />
            {t("Providers")}
            {providers.length > 0 ? (
              <span className="bg-muted text-muted-foreground ml-1 rounded-full px-1.5 py-0.5 text-[11px] font-medium tabular-nums">
                {t("{0} of {1} enabled", enabledCount, providers.length)}
              </span>
            ) : null}
          </CardTitle>
          <Button size="xs" onClick={openCreate}>
            <PlusIcon className="size-3" />
            {t("Add provider")}
          </Button>
        </CardHeader>
        <CardContent className="p-2">
          {listQuery.isLoading ? (
            <div className="space-y-2 p-2">
              {Array.from({ length: 3 }).map((_, index) => (
                <Skeleton key={index} className="h-14 w-full" />
              ))}
            </div>
          ) : providers.length === 0 ? (
            <EmptyState onAdd={openCreate} />
          ) : (
            <div className="divide-y">
              {providers.map((provider) => (
                <ProviderRow
                  key={provider.id}
                  provider={provider}
                  taskDescriptors={taskDescriptors}
                  testResult={testResults[provider.id]}
                  isTesting={testMutation.isPending && testMutation.variables === provider.id}
                  onTest={() => testMutation.mutate(provider.id)}
                  onEdit={() => openEdit(provider)}
                  onDelete={() => setDeleting(provider)}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <AIProviderDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        provider={editing}
        onSaved={invalidate}
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
    </>
  );
}

type ProviderRowProps = {
  provider: AIProvider;
  taskDescriptors: AITaskDescriptor[];
  testResult?: TestAIProviderResult;
  isTesting: boolean;
  onTest: () => void;
  onEdit: () => void;
  onDelete: () => void;
};

function ProviderRow({
  provider,
  taskDescriptors,
  testResult,
  isTesting,
  onTest,
  onEdit,
  onDelete,
}: ProviderRowProps) {
  const t = useT();

  const taskLabels = useMemo(() => {
    const byTask = new Map(taskDescriptors.map((d) => [d.task, d.label]));
    return provider.tasks.map((task) => byTask.get(task) ?? task);
  }, [provider.tasks, taskDescriptors]);

  return (
    <div className="flex flex-col gap-2 px-2 py-2.5">
      <div className="flex items-start gap-3">
        <span
          aria-hidden
          className={cn(
            "mt-1.5 size-2 shrink-0 rounded-full",
            provider.enabled ? "bg-success" : "bg-muted-foreground/40",
          )}
        />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <button
              type="button"
              onClick={onEdit}
              className="truncate text-sm font-medium hover:underline"
            >
              {provider.name}
            </button>
            <span className="text-muted-foreground truncate font-mono text-xs">
              {provider.model}
            </span>
            <Badge variant={provider.enabled ? "active" : "inactive"}>
              {provider.enabled ? t("Enabled") : t("Disabled")}
            </Badge>
            {provider.trusted && (
              <Badge variant="info" className="gap-1">
                <ShieldCheckIcon className="size-3" />
                {t("Trusted")}
              </Badge>
            )}
            {provider.allowPrivateNetwork && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span className="text-muted-foreground inline-flex">
                      <ServerIcon className="size-3.5" />
                    </span>
                  }
                />
                <TooltipContent>{t("May reach a server on your own network")}</TooltipContent>
              </Tooltip>
            )}
          </div>
          <div className="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px]">
            <span className="truncate font-mono">{provider.baseUrl || t("Provider default")}</span>
            <span>·</span>
            <span className="tabular-nums">{t("Priority {0}", provider.priority)}</span>
          </div>
          <div className="mt-1.5 flex flex-wrap gap-1">
            {taskLabels.length === 0 ? (
              <span className="text-muted-foreground text-[11px]">{t("No tasks assigned")}</span>
            ) : (
              taskLabels.map((label) => (
                <Badge key={label} variant="secondary" className="text-[10px]">
                  {label}
                </Badge>
              ))
            )}
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Button size="xs" variant="outline" onClick={onTest} isLoading={isTesting}>
            <PlugZapIcon className="size-3" />
            {t("Test")}
          </Button>
          <Button size="icon-xs" variant="ghost" aria-label={t("Edit provider")} onClick={onEdit}>
            <PencilIcon className="size-3.5" />
          </Button>
          <Button
            size="icon-xs"
            variant="ghost"
            aria-label={t("Remove provider")}
            className="text-muted-foreground hover:text-destructive"
            onClick={onDelete}
          >
            <Trash2Icon className="size-3.5" />
          </Button>
        </div>
      </div>
      {testResult && <TestResultBanner result={testResult} />}
    </div>
  );
}

/**
 * A successful connection is not the same as a usable one. An endpoint that
 * answers but ignores the JSON schema will fail later, deep inside a billing
 * diagnosis, so that case is surfaced here as a warning rather than a success.
 */
function TestResultBanner({ result }: { result: TestAIProviderResult }) {
  const t = useT();

  const tone = !result.success ? "error" : result.schemaHonoured ? "ok" : "warn";

  const toneClass = {
    ok: "border-green-600/30 bg-green-600/10",
    warn: "border-amber-600/30 bg-amber-600/10",
    error: "border-destructive/30 bg-destructive/10",
  }[tone];

  const Icon = { ok: CheckCircle2Icon, warn: AlertTriangleIcon, error: XCircleIcon }[tone];

  return (
    <div className={cn("ml-5 space-y-1 rounded-md border p-2 text-xs", toneClass)}>
      <div className="flex items-center gap-1.5 font-medium">
        <Icon className="size-3.5 shrink-0" />
        <span>{result.message}</span>
      </div>
      {result.detail && <p className="text-muted-foreground">{result.detail}</p>}
      {result.success && (
        <p className="text-muted-foreground">
          {t("Responded in {0}ms", String(result.latencyMs))}
          {result.schemaHonoured ? ` · ${t("Schema honoured")}` : ` · ${t("Schema not honoured")}`}
        </p>
      )}
    </div>
  );
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  const t = useT();

  return (
    <div className="flex flex-col items-center gap-3 py-10 text-center">
      <span className="border-border bg-muted/50 flex size-10 items-center justify-center rounded-full border">
        <PlugZapIcon className="text-muted-foreground size-5" />
      </span>
      <div>
        <p className="text-sm font-medium">{t("No AI providers configured")}</p>
        <p className="text-muted-foreground max-w-md text-sm">
          {t(
            "Add a hosted API, a gateway such as OpenRouter, or a model server running on your own hardware. AI features stay off until at least one provider is assigned to a task.",
          )}
        </p>
      </div>
      <Button size="sm" onClick={onAdd}>
        <PlusIcon className="size-4" />
        {t("Add your first provider")}
      </Button>
    </div>
  );
}
