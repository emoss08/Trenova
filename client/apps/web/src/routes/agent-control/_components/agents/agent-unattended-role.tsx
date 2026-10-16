import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { setAgentUnattendedRole } from "@/lib/graphql/agent-access";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { aiControl } from "@/lib/queries/ai-control";
import { assistant } from "@/lib/queries/assistant";
import { listRoles } from "@/lib/role-api";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";

const CEILING_ONLY = "ceiling";

export function AgentUnattendedRole({
  agent,
  canUpdate,
}: {
  agent: AgentDefinitionRow;
  canUpdate: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdateRoles } = usePermission(Resource.Role, Operation.Update);
  const canChange = canUpdate && canUpdateRoles;

  const rolesQuery = useQuery({
    queryKey: ["role-list", "unattended-role-options"],
    queryFn: () => listRoles({ limit: 200, includeSystem: true }),
    enabled: canChange || Boolean(agent.unattendedRoleId),
    staleTime: 60_000,
  });
  const roles = useMemo(() => rolesQuery.data?.results ?? [], [rolesQuery.data?.results]);
  const current = agent.unattendedRoleId || CEILING_ONLY;
  const currentName = roles.find((role) => role.id === agent.unattendedRoleId)?.name;

  const items = useMemo(
    () => [
      { value: CEILING_ONLY, label: t("The agent ceiling alone") },
      ...roles.map((role) => ({ value: role.id ?? "", label: role.name })),
    ],
    [roles, t],
  );

  const save = useApiMutation({
    mutationFn: (roleId: string | null) => setAgentUnattendedRole(agent.id, roleId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: assistant._def }),
        queryClient.invalidateQueries({ queryKey: aiControl._def }),
      ]);
    },
    resourceName: t("Agent"),
  });

  return (
    <>
      <h4>{t("Runs on its own as")}</h4>
      {canChange ? (
        <Select
          value={current}
          items={items}
          disabled={save.isPending || rolesQuery.isLoading}
          onValueChange={(value) => {
            const next = value && value !== CEILING_ONLY ? value : null;
            if (next !== (agent.unattendedRoleId || null)) {
              save.mutate(next);
            }
          }}
        >
          <SelectTrigger className="h-8 w-full text-sm" aria-label={t("Runs on its own as")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : (
        <p className="ad-v">
          {agent.unattendedRoleId ? (currentName ?? t("A role")) : t("The agent ceiling alone")}
        </p>
      )}
      <p className="ad-h">
        {t(
          "When no person is present, the agent may do only what both the agent ceiling and this role allow. You can choose only a role whose access you hold yourself.",
        )}
      </p>
    </>
  );
}
