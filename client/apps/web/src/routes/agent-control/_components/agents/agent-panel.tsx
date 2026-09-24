import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { describeApiError } from "@/lib/api-error-message";
import { setAgentAccess, type AgentAccess } from "@/lib/graphql/agent-access";
import { apiService } from "@/services/api";
import type { AgentDefinition } from "@/types/assistant";
import { zodResolver } from "@hookform/resolvers/zod";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useRef } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { saveEditedAgent, saveNewAgent, type NewAgentProblem } from "./agent-access-save";
import { AgentForm } from "./agent-form";
import {
  accessOf,
  agentFormDefaults,
  agentFormSchema,
  toSaveRequest,
  type AgentFormValues,
  type AgentPanelRow,
} from "./agent-form-schema";

/** Prefix-matches every assistant query, cards and picker alike. */
const AGENT_QUERY_SCOPE = "assistant";

export function AgentPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<AgentPanelRow>) {
  const t = useT();

  const form = useForm<AgentFormValues>({
    resolver: zodResolver(agentFormSchema) as Resolver<AgentFormValues>,
    defaultValues: agentFormDefaults,
  });

  // Who may use the agent as last saved in this session. The row is the
  // agent as it was loaded and the form is not reset after a save, so
  // without this a change saved and then undone would compare equal to the
  // row and never be sent.
  const savedAccess = useRef<{ agentId: string; access: AgentAccess } | null>(null);

  const newAgentProblemTitle = (problem: NewAgentProblem): string => {
    switch (problem.kind) {
      case "access-not-saved-left-disabled":
        return t(
          "The agent was created but left disabled, because who can use it could not be saved. Open it to try again.",
        );
      case "access-not-saved":
        return t(
          "The agent was created, but who can use it could not be saved. Open it to try again.",
        );
      case "not-enabled":
        return t(
          "The agent was created and who can use it was saved, but it could not be enabled. Open it to enable it.",
        );
    }
  };

  if (mode === "edit") {
    return (
      <FormEditPanel<AgentFormValues, AgentPanelRow, AgentFormValues, AgentDefinition>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey={AGENT_QUERY_SCOPE}
        title={t("Agent")}
        fieldKey="name"
        size="lg"
        formComponent={
          <AgentForm
            mode="edit"
            agentId={row?.id ?? ""}
            systemKey={row?.systemKey ?? ""}
            savedDelegates={row?.delegates}
          />
        }
        mutationFn={async (values, current) => {
          const lastSaved =
            savedAccess.current?.agentId === current.id
              ? savedAccess.current.access
              : accessOf(current);
          let accessSaved = false;
          try {
            const { agent } = await saveEditedAgent({
              access: accessOf(values),
              savedAccess: lastSaved,
              setAccess: (access) => setAgentAccess(current.id, access),
              onAccessSaved: (access) => {
                accessSaved = true;
                savedAccess.current = { agentId: current.id, access };
              },
              saveAgent: () =>
                apiService.agentDefinitionService.update(current.id, toSaveRequest(values)),
            });
            return agent;
          } catch (error) {
            if (accessSaved) {
              toast.warning(t("Who can use this agent was saved, but the rest of it was not."), {
                description: t("Fix what is shown and save again."),
              });
            }
            throw error;
          }
        }}
        useDock
      />
    );
  }

  return (
    <FormCreatePanel<AgentFormValues, AgentPanelRow, AgentFormValues, AgentDefinition>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={AGENT_QUERY_SCOPE}
      title={t("Agent")}
      description={t(
        "Write what the agent is for, choose the tools it may call and how much it may do on its own, and decide when it runs.",
      )}
      size="lg"
      formComponent={<AgentForm mode="create" />}
      mutationFn={async (values) => {
        const { agent, problem } = await saveNewAgent({
          request: toSaveRequest(values),
          access: accessOf(values),
          createAgent: (request) => apiService.agentDefinitionService.create(request),
          updateAgent: (id, request) => apiService.agentDefinitionService.update(id, request),
          setAccess: (agentId, access) => setAgentAccess(agentId, access),
        });
        // The agent exists, so the panel closes as a create; what is left to
        // do is said rather than thrown, because saving again would create it
        // a second time.
        if (problem) {
          toast.warning(newAgentProblemTitle(problem), {
            description: describeApiError(problem.error),
          });
        }
        return agent;
      }}
      useDock
    />
  );
}
