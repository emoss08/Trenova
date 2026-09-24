import { AgentTile } from "@/components/agent-identity/agent-tile";
import { AgentPickerList } from "@/components/assistant/agent-picker";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { setRoleAgentAccess, type RoleAgent } from "@/lib/graphql/agent-access";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { InfoIcon, PlusIcon, XIcon } from "lucide-react";
import { useMemo, useState } from "react";

const NO_RECENT: readonly string[] = [];

/** A system agent is open to everyone and granted to no role, so it is never offered. */
const isSystemAgent = (agent: AgentChoice) => agent.systemKey !== "";

/**
 * The agents a role is granted, alphabetical. An agent limited to specific
 * roles is usable by the people holding one of them; one open to everyone
 * is usable by everyone who can use the assistant, granted or not.
 */
export function sortRoleAgents(agents: readonly RoleAgent[]): RoleAgent[] {
  return [...agents].sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * The agents this role grants, added and removed here and saved as each
 * change is made, apart from the role's own Save. The list replaces the
 * role's grants as a whole, so each change sends the list as it now stands.
 */
export function RoleAgentsSection({ roleId }: { roleId: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);

  const agentsQuery = useQuery(queries.assistant.roleAgents(roleId));
  const agents = useMemo(
    () => sortRoleAgents(agentsQuery.data?.agents ?? []),
    [agentsQuery.data?.agents],
  );
  const grantedIds = useMemo(() => new Set(agents.map((agent) => agent.id)), [agents]);

  const saveMutation = useApiMutation({
    mutationFn: (agentIds: string[]) => setRoleAgentAccess(roleId, agentIds),
    onSuccess: async (saved) => {
      queryClient.setQueryData(queries.assistant.roleAgents(roleId).queryKey, saved);
      // Who may use each agent changed, so every list of agents and every
      // agent's access in AI Control is read again.
      await queryClient.invalidateQueries({ queryKey: queries.assistant._def });
    },
    // A refused change leaves the list as the server has it, not as it was
    // drawn before the change was tried.
    onError: () =>
      void queryClient.invalidateQueries({
        queryKey: queries.assistant.roleAgents(roleId).queryKey,
      }),
    resourceName: "Role",
  });

  const add = (agent: AgentChoice) => {
    setOpen(false);
    if (grantedIds.has(agent.id)) {
      return;
    }
    saveMutation.mutate([...grantedIds, agent.id]);
  };

  const remove = (agentId: string) =>
    saveMutation.mutate([...grantedIds].filter((id) => id !== agentId));

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Agents")}</CardTitle>
        <CardDescription>
          {t(
            "The agents limited to specific roles that people with this role can use. Changes here are saved as you make them.",
          )}
        </CardDescription>
        <CardAction>
          <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger
              disabled={agentsQuery.isLoading || agentsQuery.isError || saveMutation.isPending}
              render={<Button type="button" size="sm" variant="outline" />}
            >
              <PlusIcon className="size-3.5" />
              {t("Add an agent")}
            </PopoverTrigger>
            <PopoverContent
              align="end"
              side="bottom"
              sideOffset={6}
              className="ui-lift-float w-80 gap-0 overflow-hidden p-0"
            >
              {open && (
                <AgentPickerList
                  selectedId={null}
                  recentIds={NO_RECENT}
                  hiddenIds={grantedIds}
                  exclude={isSystemAgent}
                  source="grantable"
                  onSelect={add}
                  emptyMessage={t("Every agent is already granted to this role.")}
                />
              )}
            </PopoverContent>
          </Popover>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <Alert size="sm">
          <InfoIcon />
          <AlertDescription>
            {t(
              "Agents open to everyone who can use the assistant are included for this role automatically and need not be added.",
            )}
          </AlertDescription>
        </Alert>

        {agentsQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-2/3" />
          </div>
        ) : agentsQuery.isError ? (
          <Alert variant="destructive" size="sm">
            <InfoIcon />
            <AlertDescription className="flex flex-wrap items-center gap-2">
              {t("The agents this role grants could not be loaded.")}
              <Button
                type="button"
                size="xs"
                variant="outline"
                onClick={() => void agentsQuery.refetch()}
              >
                {t("Try again")}
              </Button>
            </AlertDescription>
          </Alert>
        ) : agents.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            {t("This role is granted no agents limited to specific roles.")}
          </p>
        ) : (
          <ul
            aria-label={t("Agents this role grants")}
            className="border-border flex flex-col rounded-lg border"
          >
            {agents.map((agent) => (
              <RoleAgentRow
                key={agent.id}
                agent={agent}
                disabled={saveMutation.isPending}
                onRemove={() => remove(agent.id)}
              />
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

function RoleAgentRow({
  agent,
  disabled,
  onRemove,
}: {
  agent: RoleAgent;
  disabled: boolean;
  onRemove: () => void;
}) {
  const t = useT();
  const open = agent.accessMode === "Everyone";

  return (
    <li className="border-border-subtle flex min-w-0 items-center gap-2.5 border-b px-3 py-2 last:border-b-0">
      <AgentTile agent={agent} size="sm" className={cn(!agent.enabled && "opacity-60 grayscale")} />
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-sm",
          agent.enabled ? "text-foreground" : "text-foreground-muted",
        )}
      >
        {agent.name}
      </span>
      {open && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Badge variant="neutral" appearance="outline" className="shrink-0">
                {t("Open to everyone")}
              </Badge>
            }
          />
          <TooltipContent>
            {t(
              "Everyone who can use the assistant can use it now. This grant applies once it is limited to specific roles.",
            )}
          </TooltipContent>
        </Tooltip>
      )}
      {!agent.enabled && (
        <Badge variant="neutral" appearance="outline" className="shrink-0">
          {t("Disabled")}
        </Badge>
      )}
      <Button
        type="button"
        size="icon-xs"
        variant="ghost"
        aria-label={t("Remove {0}", agent.name)}
        className="text-muted-foreground hover:text-foreground shrink-0"
        disabled={disabled}
        onClick={onRemove}
      >
        <XIcon className="size-3" />
      </Button>
    </li>
  );
}
