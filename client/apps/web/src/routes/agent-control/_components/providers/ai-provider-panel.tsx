import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { AIProvider } from "@/types/ai-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { AIProviderForm } from "./ai-provider-form";
import { buildSavePayload, type ProviderFormValues } from "./build-save-payload";
import {
  PRESET_FIELDS,
  presetValues,
  providerFormDefaults,
  providerFormSchema,
  type ProviderPanelRow,
} from "./provider-form-schema";

/** Prefix-matches the provider list, readiness and overview queries, which all sit under the `aiProvider` key scope. */
const PROVIDER_QUERY_SCOPE = "aiProvider";

type AIProviderPanelProps = DataTablePanelProps<ProviderPanelRow> & {
  /** The catalog preset a new provider starts from, when the person picked one elsewhere. */
  preset?: string | null;
};

export function AIProviderPanel({ open, onOpenChange, mode, row, preset }: AIProviderPanelProps) {
  const t = useT();
  const catalogQuery = useQuery(queries.aiProvider.catalog());

  const form = useForm<ProviderFormValues>({
    resolver: zodResolver(providerFormSchema) as Resolver<ProviderFormValues>,
    defaultValues: providerFormDefaults,
  });

  // The create panel resets the form as it opens; the preset is laid over that, as if the
  // person had picked it in the form, so it reads as their change and can be undone.
  const picked = catalogQuery.data?.presets.find((entry) => entry.key === preset);
  useEffect(() => {
    if (!open || mode !== "create" || !picked) {
      return;
    }
    const next = presetValues(picked, form.getValues());
    for (const field of PRESET_FIELDS) {
      form.setValue(field, next[field], { shouldDirty: true });
    }
  }, [form, mode, open, picked]);

  if (mode === "edit") {
    return (
      <FormEditPanel<ProviderFormValues, ProviderPanelRow, ProviderFormValues, AIProvider>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey={PROVIDER_QUERY_SCOPE}
        title={t("AI provider")}
        fieldKey="name"
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
      title={t("AI provider")}
      description={t(
        "Point Trenova at a model endpoint and choose which work it handles. Start from a preset, or configure any OpenAI-compatible server directly.",
      )}
      formComponent={<AIProviderForm mode="create" />}
      mutationFn={(values) => apiService.aiProviderService.create(buildSavePayload(values, false))}
      useDock
    />
  );
}
