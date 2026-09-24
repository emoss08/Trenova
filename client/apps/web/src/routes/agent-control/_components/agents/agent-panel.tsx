import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { AgentAccess } from "@/lib/graphql/agent-access";
import { apiService } from "@/services/api";
import type { AgentDefinition } from "@/types/assistant";
import { zodResolver } from "@hookform/resolvers/zod";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useRef } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { accessToSave, NEW_AGENT_ACCESS } from "./agent-access-save";
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
          const access = accessOf(values);
          // One request, one transaction: the agent and who may use it are
          // saved together or not at all, so a refusal leaves nothing to undo.
          const agent = await apiService.agentDefinitionService.update(
            current.id,
            toSaveRequest(values, accessToSave(access, lastSaved)),
          );
          savedAccess.current = { agentId: current.id, access };
          return agent;
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
      // Created with who may use it in one request, so an agent meant for
      // some roles is restricted from the moment it exists, enabled or not,
      // and a refusal of either leaves no agent behind to clean up.
      mutationFn={(values) =>
        apiService.agentDefinitionService.create(
          toSaveRequest(values, accessToSave(accessOf(values), NEW_AGENT_ACCESS)),
        )
      }
      useDock
    />
  );
}
