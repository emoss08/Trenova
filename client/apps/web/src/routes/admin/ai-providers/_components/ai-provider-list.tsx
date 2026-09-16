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
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AIProvider, AITaskDescriptor, TestAIProviderResult } from "@/types/ai-provider";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  PlugZapIcon,
  PlusIcon,
  ServerIcon,
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

  const providers = listQuery.data?.results ?? [];
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

  if (listQuery.isLoading) {
    return <ProviderListSkeleton />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">{t("Configured providers")}</h2>
          <p className="text-muted-foreground text-sm">
            {t(
              "Work falls through providers in priority order, so a cheaper model can handle a task first and hand off when it cannot.",
            )}
          </p>
        </div>
        <Button size="sm" onClick={openCreate}>
          <PlusIcon className="size-4" />
          {t("Add provider")}
        </Button>
      </div>

      {providers.length === 0 ? (
        <EmptyState onAdd={openCreate} />
      ) : (
        <div className="grid gap-3 lg:grid-cols-2">
          {providers.map((provider) => (
            <ProviderCard
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
    </div>
  );
}

type ProviderCardProps = {
  provider: AIProvider;
  taskDescriptors: AITaskDescriptor[];
  testResult?: TestAIProviderResult;
  isTesting: boolean;
  onTest: () => void;
  onEdit: () => void;
  onDelete: () => void;
};

function ProviderCard({
  provider,
  taskDescriptors,
  testResult,
  isTesting,
  onTest,
  onEdit,
  onDelete,
}: ProviderCardProps) {
  const t = useT();

  const taskLabels = useMemo(() => {
    const byTask = new Map(taskDescriptors.map((d) => [d.task, d.label]));
    return provider.tasks.map((task) => byTask.get(task) ?? task);
  }, [provider.tasks, taskDescriptors]);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <CardTitle className="flex items-center gap-2">
              <span className="truncate">{provider.name}</span>
              {provider.allowPrivateNetwork && (
                <ServerIcon className="text-muted-foreground size-4 shrink-0" />
              )}
            </CardTitle>
            <CardDescription className="truncate font-mono text-xs">
              {provider.model}
            </CardDescription>
          </div>
          <div className="flex shrink-0 gap-1">
            <Badge variant={provider.enabled ? "active" : "inactive"}>
              {provider.enabled ? t("Enabled") : t("Disabled")}
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <dl className="grid grid-cols-2 gap-2 text-xs">
          <div>
            <dt className="text-muted-foreground">{t("Endpoint")}</dt>
            <dd className="truncate font-mono">{provider.baseUrl || t("Provider default")}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">{t("Priority")}</dt>
            <dd>{provider.priority}</dd>
          </div>
        </dl>

        <div className="flex flex-wrap gap-1">
          {taskLabels.length === 0 ? (
            <span className="text-muted-foreground text-xs">{t("No tasks assigned")}</span>
          ) : (
            taskLabels.map((label) => (
              <Badge key={label} variant="secondary">
                {label}
              </Badge>
            ))
          )}
          {provider.trusted && <Badge variant="active">{t("Trusted")}</Badge>}
        </div>

        {testResult && <TestResultBanner result={testResult} />}

        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={onTest} isLoading={isTesting}>
            <PlugZapIcon className="size-4" />
            {t("Test")}
          </Button>
          <Button size="sm" variant="outline" onClick={onEdit}>
            {t("Edit")}
          </Button>
          <Button size="sm" variant="outline" onClick={onDelete}>
            <Trash2Icon className="size-4" />
          </Button>
        </div>
      </CardContent>
    </Card>
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
    <div className={`space-y-1 rounded-md border p-2 text-xs ${toneClass}`}>
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
    <Card>
      <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
        <PlugZapIcon className="text-muted-foreground size-8" />
        <div>
          <p className="font-medium">{t("No AI providers configured")}</p>
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
      </CardContent>
    </Card>
  );
}

function ProviderListSkeleton() {
  return (
    <div className="grid gap-3 lg:grid-cols-2">
      <Skeleton className="h-52" />
      <Skeleton className="h-52" />
    </div>
  );
}
