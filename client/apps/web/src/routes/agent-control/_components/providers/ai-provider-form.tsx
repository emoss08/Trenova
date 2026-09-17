import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { queries } from "@/lib/queries";
import type { AIProviderPreset } from "@/types/ai-provider";
import { useQuery } from "@tanstack/react-query";
import { KeyRoundIcon, ShieldAlertIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import type { ProviderFormValues } from "./build-save-payload";
import { PresetPicker } from "./preset-picker";

type AIProviderFormProps = {
  mode: "create" | "edit";
};

export function AIProviderForm({ mode }: AIProviderFormProps) {
  const t = useT();
  const { control, setValue, getValues } = useFormContext<ProviderFormValues>();

  const catalogQuery = useQuery(queries.aiProvider.catalog());
  const catalog = catalogQuery.data;

  const preset = useWatch({ control, name: "preset" });
  const kind = useWatch({ control, name: "kind" });
  const tasks = useWatch({ control, name: "tasks" });
  const allowPrivateNetwork = useWatch({ control, name: "allowPrivateNetwork" });

  const kindDescriptor = useMemo(
    () => catalog?.kinds.find((descriptor) => descriptor.kind === kind),
    [catalog?.kinds, kind],
  );

  const selectedPreset = useMemo(
    () => catalog?.presets.find((entry) => entry.key === preset),
    [catalog?.presets, preset],
  );

  const applyPreset = useCallback(
    (picked: AIProviderPreset | null) => {
      if (!picked) {
        setValue("preset", "", { shouldDirty: true });
        return;
      }

      setValue("preset", picked.key, { shouldDirty: true });
      setValue("kind", picked.kind, { shouldDirty: true });
      setValue("baseUrl", picked.baseUrl, { shouldDirty: true });
      setValue("structuredOutputMode", picked.structuredOutputMode, { shouldDirty: true });
      setValue("allowPrivateNetwork", picked.allowPrivateNetwork, { shouldDirty: true });
      if (picked.exampleModel) {
        setValue("model", picked.exampleModel, { shouldDirty: true });
      }
      if (getValues("name").trim() === "") {
        setValue("name", picked.label.replace(/\s*\(self-hosted\)$/i, ""), { shouldDirty: true });
      }
    },
    [getValues, setValue],
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

  const selectedLedgerTask = useMemo(
    () =>
      (tasks ?? []).some(
        (task) => catalog?.tasks.find((descriptor) => descriptor.task === task)?.requiresTrust,
      ),
    [tasks, catalog?.tasks],
  );

  return (
    <div className="flex flex-col gap-6 pb-14">
      <FormSection
        title={t("Start from a preset")}
        description={t(
          "Fills in the endpoint and output settings for a known deployment. Everything stays editable.",
        )}
      >
        <PresetPicker
          control={control}
          presets={catalog?.presets ?? []}
          isLoading={catalogQuery.isLoading}
          onSelect={applyPreset}
        />
        {selectedPreset?.notes && (
          <p className="text-muted-foreground max-w-prose text-xs">{selectedPreset.notes}</p>
        )}
      </FormSection>

      <FormSection
        title={t("Endpoint")}
        description={t("Where requests go and how they are authenticated.")}
      >
        <FormGroup cols={2}>
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
              options={(catalog?.kinds ?? []).map((descriptor) => ({
                label: descriptor.label,
                value: descriptor.kind,
              }))}
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
                mode === "edit"
                  ? t("API key (leave blank to keep the existing key)")
                  : kindDescriptor?.requiresApiKey
                    ? t("API key")
                    : t("API key (optional)")
              }
              autoComplete="off"
              placeholder={mode === "edit" ? "••••••••" : t("Enter the API key")}
              description={
                kindDescriptor?.requiresApiKey
                  ? t("Required for this provider. Stored encrypted and never shown again.")
                  : t("Many self-hosted servers need no credential. Leave blank if yours does not.")
              }
            />
          </FormControl>

          <FormControl cols="full">
            <TextareaField
              name="description"
              control={control}
              label={t("Description")}
              placeholder={t(
                "A small model on the office GPU server; cheap enough for every request.",
              )}
              description={t(
                "For the next administrator. Where the endpoint runs and why it handles the tasks it does.",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Routing")}
        description={t(
          "Work is offered to providers in priority order. Assign a cheap model to high-volume tasks and keep a stronger one behind it.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <MultiCheckboxField
              name="tasks"
              control={control}
              label={t("Handles these tasks")}
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
        </FormGroup>

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
      </FormSection>

      <FormSection
        title={t("Output and limits")}
        description={t("How much the endpoint guarantees, and how far a single reply may go.")}
      >
        <FormGroup cols={2}>
          <FormControl>
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
                "Some OpenAI-compatible servers accept a schema and ignore it — test the connection to find out before relying on it.",
              )}
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
      </FormSection>

      <FormSection title={t("Availability")}>
        <FormGroup cols={1}>
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
        {mode === "edit" && (
          <p className="text-muted-foreground flex items-center gap-1.5 text-xs">
            <KeyRoundIcon className="size-3.5" />
            {t("Save, then use Test on the card to check the endpoint honours JSON schemas.")}
          </p>
        )}
      </FormSection>
    </div>
  );
}
