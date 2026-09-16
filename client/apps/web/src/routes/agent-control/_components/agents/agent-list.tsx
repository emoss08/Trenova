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
import type { AgentDefinition, AutonomyTier } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, BotIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { AgentDialog } from "./agent-dialog";

const AUTONOMY_LABELS: Record<AutonomyTier, string> = {
  Propose: "Proposes only",
  ActWithApproval: "Acts with approval",
  AutoExecute: "Acts on its own",
};

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

  const enabledCount = agents.filter((agent) => agent.enabled).length;

  return (
    <>
      <Card className="gap-0 p-0">
        <CardHeader className="flex flex-row items-center justify-between border-b py-3">
          <CardTitle className="flex items-center gap-1.5 text-sm font-medium">
            <BotIcon className="text-muted-foreground size-3.5" />
            {t("Agents")}
            {agents.length > 0 ? (
              <span className="bg-muted text-muted-foreground ml-1 rounded-full px-1.5 py-0.5 text-[11px] font-medium tabular-nums">
                {t("{0} of {1} enabled", enabledCount, agents.length)}
              </span>
            ) : null}
          </CardTitle>
          <Button size="xs" onClick={openCreate}>
            <PlusIcon className="size-3" />
            {t("Add agent")}
          </Button>
        </CardHeader>
        <CardContent className="p-2">
          {listQuery.isLoading ? (
            <div className="space-y-2 p-2">
              {Array.from({ length: 3 }).map((_, index) => (
                <Skeleton key={index} className="h-12 w-full" />
              ))}
            </div>
          ) : agents.length === 0 ? (
            <EmptyState onAdd={openCreate} />
          ) : (
            <div className="divide-y">
              {agents.map((agent) => (
                <AgentRow
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
        </CardContent>
      </Card>

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
    </>
  );
}

type AgentRowProps = {
  agent: AgentDefinition;
  templateLabel: string;
  onEdit: () => void;
  onDelete: () => void;
};

function AgentRow({ agent, templateLabel, onEdit, onDelete }: AgentRowProps) {
  const t = useT();

  return (
    <div className="flex items-start gap-3 px-2 py-2.5">
      <span
        aria-hidden
        className={`mt-1.5 size-2 shrink-0 rounded-full ${agent.enabled ? "bg-success" : "bg-muted-foreground/40"}`}
      />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <button
            type="button"
            onClick={onEdit}
            className="truncate text-sm font-medium hover:underline"
          >
            {agent.name}
          </button>
          <Badge variant="outline">{templateLabel}</Badge>
          <Badge variant={agent.enabled ? "active" : "inactive"}>
            {agent.enabled ? t("Enabled") : t("Disabled")}
          </Badge>
        </div>
        {agent.description && (
          <p className="text-muted-foreground mt-0.5 text-xs">{agent.description}</p>
        )}
        <div className="text-muted-foreground mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px]">
          <span>{t(AUTONOMY_LABELS[agent.autonomyCeiling])}</span>
          <span>·</span>
          {agent.toolNames.length === 0 ? (
            <span>{t("Answers only")}</span>
          ) : (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="cursor-default">
                    {t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
                  </span>
                }
              />
              <TooltipContent className="font-mono text-[11px]">
                {agent.toolNames.join(", ")}
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <Button size="icon-xs" variant="ghost" aria-label={t("Edit agent")} onClick={onEdit}>
          <PencilIcon className="size-3.5" />
        </Button>
        <Button
          size="icon-xs"
          variant="ghost"
          aria-label={t("Remove agent")}
          className="text-muted-foreground hover:text-destructive"
          onClick={onDelete}
        >
          <Trash2Icon className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  const t = useT();

  return (
    <div className="flex flex-col items-center gap-3 py-10 text-center">
      <span className="border-border bg-muted/50 flex size-10 items-center justify-center rounded-full border">
        <BotIcon className="text-muted-foreground size-5" />
      </span>
      <div>
        <p className="text-sm font-medium">{t("No agents configured")}</p>
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
    </div>
  );
}
