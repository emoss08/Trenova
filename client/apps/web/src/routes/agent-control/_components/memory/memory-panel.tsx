import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  AGENT_MEMORY_LIST_KEY,
  createAgentMemory,
  updateAgentMemory,
  type AgentMemoryRow,
} from "@/lib/graphql/agent-memories";
import { zodResolver } from "@hookform/resolvers/zod";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useForm, type Resolver } from "react-hook-form";
import { MemoryForm } from "./memory-form";
import {
  memoryFormDefaults,
  memoryFormSchema,
  toMemoryInput,
  type MemoryFormValues,
} from "./memory-form-schema";

export function MemoryPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<AgentMemoryRow>) {
  const t = useT();
  const form = useForm<MemoryFormValues>({
    resolver: zodResolver(memoryFormSchema) as Resolver<MemoryFormValues>,
    defaultValues: memoryFormDefaults,
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel<
        MemoryFormValues,
        AgentMemoryRow,
        MemoryFormValues,
        Awaited<ReturnType<typeof updateAgentMemory>>
      >
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        useDock
        form={form}
        queryKey={AGENT_MEMORY_LIST_KEY}
        title={t("Memory")}
        fieldKey="content"
        mutationFn={(values, current) => updateAgentMemory(current.id, toMemoryInput(values))}
        formComponent={<MemoryForm />}
      />
    );
  }

  return (
    <FormCreatePanel<
      MemoryFormValues,
      AgentMemoryRow,
      MemoryFormValues,
      Awaited<ReturnType<typeof createAgentMemory>>
    >
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={AGENT_MEMORY_LIST_KEY}
      title={t("Memory")}
      description={t("Every agent that asks for memory reads this on its next run.")}
      mutationFn={(values) => createAgentMemory(toMemoryInput(values))}
      formComponent={<MemoryForm />}
    />
  );
}
