import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { AgentDefinition } from "@/types/assistant";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { AgentForm } from "./agent-form";
import {
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
        size="xl"
        formComponent={<AgentForm mode="edit" systemKey={row?.systemKey ?? ""} />}
        mutationFn={(values, current) =>
          apiService.agentDefinitionService.update(current.id, toSaveRequest(values))
        }
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
      size="xl"
      formComponent={<AgentForm mode="create" />}
      mutationFn={(values) => apiService.agentDefinitionService.create(toSaveRequest(values))}
      useDock
    />
  );
}
