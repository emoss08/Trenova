import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AIProvider } from "@/types/ai-provider";
import { buildSavePayload, type ProviderFormValues } from "./build-save-payload";
import { useQuery } from "@tanstack/react-query";
import { ShieldAlertIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

type AIProviderFormProps = {
  provider: AIProvider | null;
  onClose: () => void;
  onSaved: () => Promise<void> | void;
};

type FormValues = ProviderFormValues;

const defaultValues: FormValues = {
  preset: "",
  name: "",
  description: "",
  kind: "OpenAIChat",
  baseUrl: "",
  model: "",
  apiKey: "",
  allowPrivateNetwork: false,
  structuredOutputMode: "Prompted",
  maxTokens: 8192,
  tasks: null,
  priority: 100,
  trusted: false,
  enabled: true,
  version: 0,
};

export function AIProviderForm({ provider, onClose, onSaved }: AIProviderFormProps) {
  const t = useT();

  const catalogQuery = useQuery(queries.aiProvider.catalog());
  const catalog = catalogQuery.data;

  const form = useForm<FormValues>({
    defaultValues: provider
      ? {
          preset: "",
          name: provider.name,
          description: provider.description,
          kind: provider.kind,
          baseUrl: provider.baseUrl,
          model: provider.model,
          // Left blank on edit: the secret is never sent to the client, and an
          // omitted value tells the server to keep what it already has.
          apiKey: "",
          allowPrivateNetwork: provider.allowPrivateNetwork,
          structuredOutputMode: provider.structuredOutputMode,
          maxTokens: provider.maxTokens,
          tasks: provider.tasks.length > 0 ? provider.tasks : null,
          priority: provider.priority,
          trusted: provider.trusted,
          enabled: provider.enabled,
          version: provider.version,
        }
      : defaultValues,
  });
  const { control, handleSubmit, setValue } = form;

  const kind = useWatch({ control, name: "kind" });
  const tasks = useWatch({ control, name: "tasks" });
  const allowPrivateNetwork = useWatch({ control, name: "allowPrivateNetwork" });

  const kindDescriptor = useMemo(
    () => catalog?.kinds.find((k) => k.kind === kind),
    [catalog?.kinds, kind],
  );

  const presetOptions = useMemo(
    () =>
      (catalog?.presets ?? []).map((preset) => ({
        label: preset.selfHosted ? `${preset.label}` : preset.label,
        value: preset.key,
      })),
    [catalog?.presets],
  );

  const applyPreset = useCallback(
    (key: string) => {
      const preset = catalog?.presets.find((p) => p.key === key);
      if (!preset) {
        return;
      }

      setValue("kind", preset.kind, { shouldDirty: true });
      setValue("baseUrl", preset.baseUrl, { shouldDirty: true });
      setValue("structuredOutputMode", preset.structuredOutputMode, { shouldDirty: true });
      setValue("allowPrivateNetwork", preset.allowPrivateNetwork, { shouldDirty: true });
      if (preset.exampleModel) {
        setValue("model", preset.exampleModel, { shouldDirty: true });
      }
    },
    [catalog?.presets, setValue],
  );

  const saveMutation = useApiMutation({
    mutationFn: (values: FormValues) => {
      const payload = buildSavePayload(values, provider !== null);

      return provider
        ? apiService.aiProviderService.update(provider.id, payload)
        : apiService.aiProviderService.create(payload);
    },
    form,
    resourceName: "AI Provider",
    onSuccess: async () => {
      toast.success(provider ? t("AI provider updated") : t("AI provider added"));
      await onSaved();
      onClose();
    },
  });

  const selectedLedgerTask = useMemo(
    () =>
      (tasks ?? []).some(
        (task) => catalog?.tasks.find((d) => d.task === task)?.requiresTrust ?? false,
      ),
    [tasks, catalog?.tasks],
  );

  const taskOptions = useMemo(
    () =>
      (catalog?.tasks ?? []).map((descriptor) => ({
        label: descriptor.label,
        value: descriptor.task,
        description: descriptor.volumeGuidance,
      })),
    [catalog?.tasks],
  );

  return (
    <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <SelectField
            name="preset"
            control={control}
            label={t("Start from a preset")}
            description={t(
              "Fills in the endpoint and output settings for a known deployment. Everything stays editable.",
            )}
            options={presetOptions}
            placeholder={t("Choose a preset (optional)")}
            onValueChange={applyPreset}
            isClearable
          />
        </FormControl>

        <FormControl>
          <InputField
            name="name"
            control={control}
            label={t("Name")}
            placeholder={t("Local Qwen")}
            rules={{ required: t("Name is required") }}
            description={t("How this endpoint appears when assigning work.")}
          />
        </FormControl>

        <FormControl>
          <SelectField
            name="kind"
            control={control}
            label={t("Protocol")}
            options={(catalog?.kinds ?? []).map((k) => ({ label: k.label, value: k.kind }))}
            description={kindDescriptor?.description}
          />
        </FormControl>

        <FormControl>
          <InputField
            name="model"
            control={control}
            label={t("Model")}
            placeholder="qwen3:32b"
            rules={{ required: t("Model is required") }}
            description={t("Exactly as the endpoint names it.")}
          />
        </FormControl>

        <FormControl>
          <InputField
            name="baseUrl"
            control={control}
            label={t("Base URL")}
            placeholder={kindDescriptor?.defaultBaseUrl || "http://localhost:8000/v1"}
            description={
              kindDescriptor?.requiresBaseUrl
                ? t("Required for an OpenAI-compatible endpoint.")
                : t("Leave blank to use the provider default.")
            }
          />
        </FormControl>

        <FormControl cols="full">
          <SensitiveField
            name="apiKey"
            control={control}
            label={
              provider
                ? t("API key (leave blank to keep the existing key)")
                : kindDescriptor?.requiresApiKey
                  ? t("API key")
                  : t("API key (optional)")
            }
            autoComplete="off"
            placeholder={provider ? "********" : t("Enter the API key")}
            description={
              kindDescriptor?.requiresApiKey
                ? t("Required for this provider.")
                : t("Many self-hosted servers need no credential. Leave blank if yours does not.")
            }
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            name="description"
            control={control}
            label={t("Description")}
            placeholder={t("What this endpoint is for")}
          />
        </FormControl>
      </FormGroup>

      <FormGroup cols={2}>
        <FormControl cols="full">
          <MultiCheckboxField
            name="tasks"
            control={control}
            label={t("Handles these tasks")}
            description={t(
              "Work is offered to providers in priority order. Assign a cheap model to high-volume tasks and keep a stronger one behind it.",
            )}
            options={taskOptions}
          />
        </FormControl>

        <FormControl>
          <NumberField
            name="priority"
            control={control}
            label={t("Priority")}
            description={t("Lower runs first. Providers behind it act as fallbacks.")}
          />
        </FormControl>

        <FormControl>
          <NumberField
            name="maxTokens"
            control={control}
            label={t("Max tokens")}
            description={t("Ceiling for a single reply.")}
          />
        </FormControl>

        <FormControl cols="full">
          <SelectField
            name="structuredOutputMode"
            control={control}
            label={t("Structured output")}
            options={[
              { label: t("Enforced by the server (JSON schema)"), value: "JSONSchema" },
              { label: t("Valid JSON only (no schema)"), value: "JSONMode" },
              { label: t("Requested in the prompt"), value: "Prompted" },
            ]}
            description={t(
              "How much the endpoint guarantees. Some OpenAI-compatible servers accept a schema and ignore it — test the connection to find out before relying on it.",
            )}
          />
        </FormControl>
      </FormGroup>

      <FormGroup cols={1}>
        <FormControl cols="full">
          <SwitchField
            name="allowPrivateNetwork"
            control={control}
            label={t("Allow private network access")}
            description={t(
              "Needed to reach a model server on your own network. Link-local addresses stay blocked either way.",
            )}
            outlined
          />
        </FormControl>

        <FormControl cols="full">
          <SwitchField
            name="trusted"
            control={control}
            label={t("Trusted for financial work")}
            description={t(
              "Required before this provider can handle tasks that change financial records.",
            )}
            outlined
          />
        </FormControl>

        <FormControl cols="full">
          <SwitchField
            name="enabled"
            control={control}
            label={t("Enabled")}
            description={t("Disabled providers are skipped entirely.")}
            outlined
          />
        </FormControl>
      </FormGroup>

      {allowPrivateNetwork && (
        <Alert variant="info">
          <ShieldAlertIcon className="size-4" />
          <AlertTitle>{t("This provider may reach your internal network")}</AlertTitle>
          <AlertDescription>
            {t(
              "Only point this at a model server you operate. Trenova still refuses link-local addresses, so cloud metadata endpoints remain unreachable.",
            )}
          </AlertDescription>
        </Alert>
      )}

      {selectedLedgerTask && (
        <Alert variant="warning">
          <ShieldAlertIcon className="size-4" />
          <AlertTitle>{t("This task can change financial records")}</AlertTitle>
          <AlertDescription>
            {t(
              "Billing diagnosis proposes changes against your ledger, so it only runs on a provider marked trusted. Open-weight models vary a lot in how reliably they follow tool definitions — verify this one before trusting it.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <DialogFooter className="flex flex-row items-center sm:justify-between">
        <Button type="button" variant="outline" onClick={onClose}>
          {t("Cancel")}
        </Button>
        <Button
          type="submit"
          size="sm"
          isLoading={saveMutation.isPending}
          loadingText={t("Saving...")}
          disabled={catalogQuery.isLoading}
        >
          {provider ? t("Save changes") : t("Add provider")}
        </Button>
      </DialogFooter>
    </Form>
  );
}
