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
import type { AgentDefinition } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, BotIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { AgentDialog } from "./agent-dialog";

export function AgentList() {
  const t = useT();
  const queryClient = useQueryClient();

  const listQuery = useQuery(queries.assistant.agents(false));
  const templatesQuery = useQuery(queries.assistant.agentTemplates());

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<AgentDefinition | null>(null);
  const [deleting, setDeleting] = useState<AgentDefinition | null>(null);

  const agents = listQuery.data?.results ?? [];
  const templates = templatesQuery.data?.templates ?? [];

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.assistant.agents(false).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.agents(true).queryKey }),
    ]);
  }, [queryClient]);

  const deleteMutation = useApiMutation({
    mutationFn: (id: string) => apiService.agentDefinitionService.remove(id),
    onSuccess: async () => {
      toast.success(t("Agent removed"));
      setDeleting(null);
      await invalidate();
    },
    resourceName: "Agent",
  });

  const openCreate = useCallback(() => {
    setEditing(null);
    setDialogOpen(true);
  }, []);

  if (listQuery.isLoading) {
    return (
      <div className="grid gap-3 lg:grid-cols-2">
        <Skeleton className="h-44" />
        <Skeleton className="h-44" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">{t("Configured agents")}</h2>
          <p className="text-muted-foreground text-sm">
            {t("Enabled agents appear in the assistant for everyone who can use it.")}
          </p>
        </div>
        <Button size="sm" onClick={openCreate}>
          <PlusIcon className="size-4" />
          {t("Add agent")}
        </Button>
      </div>

      {agents.length === 0 ? (
        <EmptyState onAdd={openCreate} />
      ) : (
        <div className="grid gap-3 lg:grid-cols-2">
          {agents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              templateLabel={
                templates.find((template) => template.kind === agent.kind)?.label ?? agent.kind
              }
              onEdit={() => {
                setEditing(agent);
                setDialogOpen(true);
              }}
              onDelete={() => setDeleting(agent)}
            />
          ))}
        </div>
      )}

      <AgentDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        agent={editing}
        templates={templates}
        onSaved={invalidate}
      />

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <AlertTriangleIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Remove agent")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Remove {0}. Existing conversations with this agent will remain but cannot be continued.",
                deleting?.name ?? t("this agent"),
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
              {t("Remove agent")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

type AgentCardProps = {
  agent: AgentDefinition;
  templateLabel: string;
  onEdit: () => void;
  onDelete: () => void;
};

function AgentCard({ agent, templateLabel, onEdit, onDelete }: AgentCardProps) {
  const t = useT();

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <CardTitle className="flex items-center gap-2">
              <BotIcon className="size-4 shrink-0" />
              <span className="truncate">{agent.name}</span>
            </CardTitle>
            <CardDescription>{templateLabel}</CardDescription>
          </div>
          <Badge variant={agent.enabled ? "active" : "inactive"}>
            {agent.enabled ? t("Enabled") : t("Disabled")}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {agent.description && <p className="text-muted-foreground text-sm">{agent.description}</p>}

        <div className="flex flex-wrap gap-1">
          {agent.toolNames.length === 0 ? (
            <Badge variant="secondary">{t("Answers only")}</Badge>
          ) : (
            agent.toolNames.map((tool) => (
              <Badge key={tool} variant="secondary" className="font-mono text-[11px]">
                {tool}
              </Badge>
            ))
          )}
        </div>

        <p className="text-muted-foreground text-xs">
          {t("Autonomy ceiling: {0}", agent.autonomyCeiling)}
        </p>

        <div className="flex gap-2">
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

function EmptyState({ onAdd }: { onAdd: () => void }) {
  const t = useT();

  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
        <BotIcon className="text-muted-foreground size-8" />
        <div>
          <p className="font-medium">{t("No agents configured")}</p>
          <p className="text-muted-foreground max-w-md text-sm">
            {t(
              "Add an agent so your team can use the assistant. Start with a read-only general assistant if you want to try it without giving it any tools.",
            )}
          </p>
        </div>
        <Button size="sm" onClick={onAdd}>
          <PlusIcon className="size-4" />
          {t("Add your first agent")}
        </Button>
      </CardContent>
    </Card>
  );
}
