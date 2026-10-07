import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

export function useSpecIntegrationConfig({
  integrationType,
  name,
  open,
  onChanged,
}: {
  integrationType: string;
  name: string;
  open: boolean;
  onChanged?: () => Promise<unknown>;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const configQuery = useQuery({
    ...queries.integration.config(integrationType),
    enabled: open,
  });
  const response = configQuery.data;
  const spec = useMemo(() => response?.spec ?? [], [response]);

  const form = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: { enabled: false, configuration: {} },
  });
  const { reset } = form;

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    const valueByKey = new Map(response.fields.map((field) => [field.key, field.value ?? ""]));
    const configuration: Record<string, string> = {};
    for (const field of response.spec) {
      configuration[field.key] = field.sensitive
        ? ""
        : (valueByKey.get(field.key) ?? field.default ?? "");
    }

    reset({ enabled: response.enabled, configuration });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig(integrationType, payload),
    form,
    resourceName: `${name} configuration`,
    onSuccess: async () => {
      toast.success(t("{0} integration updated", name));
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config(integrationType).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
        onChanged?.(),
      ]);
    },
  });

  const testConnectionMutation = useMutation({
    mutationFn: () => apiService.integrationService.testConnection(integrationType),
    onSuccess: async () => {
      toast.success(t("{0} connection successful", name));
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
        onChanged?.(),
      ]);
    },
    onError: () => toast.error(t("{0} connection test failed", name)),
  });

  const storedByKey = useMemo(
    () => new Map(response?.fields.map((field) => [field.key, field.hasValue]) ?? []),
    [response],
  );

  return {
    configQuery,
    response,
    spec,
    form,
    storedByKey,
    saveMutation,
    testConnectionMutation,
  };
}
