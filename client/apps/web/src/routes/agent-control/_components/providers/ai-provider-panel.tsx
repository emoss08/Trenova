import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { AIProvider } from "@/types/ai-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { AIProviderForm } from "./ai-provider-form";
import { buildSavePayload, type ProviderFormValues } from "./build-save-payload";
import {
  providerFormDefaults,
  providerFormSchema,
  type ProviderPanelRow,
} from "./provider-form-schema";

/** Prefix-matches the provider list, readiness and overview queries, which all sit under the `aiProvider` key scope. */
const PROVIDER_QUERY_SCOPE = "aiProvider";

export function AIProviderPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<ProviderPanelRow>) {
  const t = useT();

  const form = useForm<ProviderFormValues>({
    resolver: zodResolver(providerFormSchema) as Resolver<ProviderFormValues>,
    defaultValues: providerFormDefaults,
  });

  if (mode === "edit") {
    return (
      <FormEditPanel<ProviderFormValues, ProviderPanelRow, ProviderFormValues, AIProvider>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey={PROVIDER_QUERY_SCOPE}
        title={t("AI Provider")}
        fieldKey="name"
        size="lg"
        formComponent={<AIProviderForm mode="edit" />}
        mutationFn={(values, current) =>
          apiService.aiProviderService.update(current.id, buildSavePayload(values, true))
        }
        useDock
      />
    );
  }

  return (
    <FormCreatePanel<ProviderFormValues, ProviderPanelRow, ProviderFormValues, AIProvider>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={PROVIDER_QUERY_SCOPE}
      title={t("AI Provider")}
      description={t(
        "Point Trenova at a model endpoint and choose which work it handles. Start from a preset, or configure any OpenAI-compatible server directly.",
      )}
      size="lg"
      formComponent={<AIProviderForm mode="create" />}
      mutationFn={(values) => apiService.aiProviderService.create(buildSavePayload(values, false))}
      useDock
    />
  );
}
