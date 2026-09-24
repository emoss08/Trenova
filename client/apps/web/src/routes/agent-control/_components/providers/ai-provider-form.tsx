import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { accentVar, toneVar } from "@/components/kpi/tone";
import { queries } from "@/lib/queries";
import type { AIProviderPreset } from "@/types/ai-provider";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { KeyRoundIcon, ShieldAlertIcon, WaypointsIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import type { ProviderFormValues } from "./build-save-payload";
import { kindMark } from "./kind-marks";
import { PresetPicker } from "./preset-picker";
import { kindSupportsEmbedding } from "./provider-form-schema";

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
  const embeds = (tasks ?? []).includes("Embedding");

  const kindDescriptor = useMemo(
    () => catalog?.kinds.find((descriptor) => descriptor.kind === kind),
    [catalog?.kinds, kind],
  );

  const selectedPreset = useMemo(
    () => catalog?.presets.find((entry) => entry.key === preset),
    [catalog?.presets, preset],
  );

  // An embedding preset brings the task, the vector size and the input style
  // with it; picking a text preset afterwards takes them away again, since an
  // embedding model serves nothing else.
  const applyEmbeddingPreset = useCallback(
    (picked: AIProviderPreset) => {
      if (picked.tasks.length > 0) {
        setValue("tasks", [...picked.tasks], { shouldDirty: true });
      } else if ((getValues("tasks") ?? []).includes("Embedding")) {
        setValue("tasks", null, { shouldDirty: true });
      }
      setValue(
        "embeddingDimensionsChoice",
        picked.embeddingDimensions > 0 ? String(picked.embeddingDimensions) : "",
        { shouldDirty: true },
      );
      setValue("embeddingInputStyle", picked.embeddingInputStyle ?? "None", {
        shouldDirty: true,
      });
    },
    [getValues, setValue],
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
      applyEmbeddingPreset(picked);
    },
    [applyEmbeddingPreset, getValues, setValue],
  );

  const embeddingSupported = kindDescriptor?.supportsEmbedding ?? kindSupportsEmbedding(kind);

  const taskOptions = useMemo(
    () =>
      (catalog?.tasks ?? []).map((descriptor) => {
        const unsupported = descriptor.task === "Embedding" && !embeddingSupported;

        return {
          label: descriptor.label,
          value: descriptor.task,
          disabled: unsupported,
          description: unsupported
            ? t(
                "This protocol has no embedding endpoint. Use an OpenAI, OpenAI-compatible or Ollama provider for embeddings.",
              )
            : descriptor.volumeGuidance,
        };
      }),
    [catalog?.tasks, embeddingSupported, t],
  );

  const dimensionOptions = useMemo(
    () =>
      (catalog?.embeddingDimensions ?? []).map((dimensions) => ({
        label: t("{0, plural, one {# dimension} other {# dimensions}}", dimensions),
        value: String(dimensions),
      })),
    [catalog?.embeddingDimensions, t],
  );

  const inputStyleOptions = useMemo(
    () =>
      (catalog?.embeddingInputStyles ?? []).map((descriptor) => ({
        label: descriptor.label,
        value: descriptor.style,
        description: descriptor.description,
      })),
    [catalog?.embeddingInputStyles],
  );

  const selectedLedgerTask = useMemo(
    () =>
      (tasks ?? []).some(
        (task) => catalog?.tasks.find((descriptor) => descriptor.task === task)?.requiresTrust,
      ),
    [tasks, catalog?.tasks],
  );

  return (
    <div className="flex flex-col gap-6">
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
              placeholder={t("Local qwen")}
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
                icon: kindMark(descriptor.kind),
                description: descriptor.description,
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
          <FormControl cols="full">
            <NumberField
              name="priority"
              control={control}
              label={t("Priority")}
              description={t("Lower runs first. Providers behind it act as fallbacks.")}
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

      {embeds && (
        <FormSection
          title={t("Embedding")}
          description={t(
            "Vectors from this model are what agents search by meaning. Changing the model or its size re-indexes everything it has embedded.",
          )}
        >
          <FormGroup cols={2}>
            <FormControl>
              <SelectField
                name="embeddingDimensionsChoice"
                control={control}
                label={t("Dimensions")}
                placeholder={t("Choose a size")}
                options={dimensionOptions}
                description={t(
                  "The vector size the model returns. A reply of any other size is refused.",
                )}
              />
            </FormControl>
            <FormControl>
              <SelectField
                name="embeddingInputStyle"
                control={control}
                label={t("Input style")}
                options={inputStyleOptions}
                description={t("How the endpoint is told a stored document from a search query.")}
              />
            </FormControl>
          </FormGroup>
          <Alert variant="info" size="sm">
            <WaypointsIcon />
            <AlertDescription>
              {t(
                "Documents are sent with social security, card and bank account numbers masked. Only one embedding model is searched at a time; providers with the same model and size back each other up.",
              )}
            </AlertDescription>
          </Alert>
        </FormSection>
      )}

      <FormSection
        title={t("Output and limits")}
        description={t("How much the endpoint guarantees, and how far a single reply may go.")}
      >
        <FormGroup cols={2}>
          {!embeds && (
            <>
              <FormControl>
                <SelectField
                  name="structuredOutputMode"
                  control={control}
                  label={t("Structured output")}
                  // Colour here is the guarantee, strongest first: a server that
                  // enforces the schema cannot return the wrong shape, one that only
                  // promises JSON can, and one merely asked in the prompt often does.
                  options={[
                    {
                      label: t("Enforced by the server (JSON schema)"),
                      value: "JSONSchema",
                      color: toneVar("success"),
                    },
                    {
                      label: t("Valid JSON only (no schema)"),
                      value: "JSONMode",
                      color: toneVar("warning"),
                    },
                    {
                      label: t("Requested in the prompt"),
                      value: "Prompted",
                      color: toneVar("muted"),
                    },
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
                <SelectField
                  name="reasoningEffort"
                  control={control}
                  label={t("Reasoning")}
                  // Off is the safe default: the reasoning parameter is refused by
                  // models without it. The levels are a categorical scale of
                  // effort, not severities, so they take one accent.
                  options={[
                    {
                      label: t("Off — answer directly"),
                      value: "Off",
                      color: toneVar("muted"),
                    },
                    { label: t("Low"), value: "Low", color: accentVar("teal") },
                    { label: t("Medium"), value: "Medium", color: accentVar("teal") },
                    { label: t("High"), value: "High", color: accentVar("teal") },
                  ]}
                  description={t(
                    "Asks a model that can think to do so before it answers, and shows the thinking in the panel. Turn it on only for a model that reasons; others reject the request.",
                  )}
                />
              </FormControl>
            </>
          )}

          <FormControl cols="full">
            <TextareaField
              name="extraBodyText"
              control={control}
              label={t("Extra request fields")}
              rows={5}
              placeholder={
                '{\n  "chat_template_kwargs": { "enable_thinking": true },\n  "reasoning_budget": 16384\n}'
              }
              description={t(
                "JSON merged into every request to this endpoint, for the fields its server takes that the protocol does not define. Copy them from the provider's own example. What this system sets — the model, the messages, the tools, the schema and whether the call streams — cannot be overridden here.",
              )}
            />
          </FormControl>

          <FormControl>
            <NumberField
              name="inputCostPerMillion"
              control={control}
              label={t("Input price, USD per million tokens")}
              description={
                embeds
                  ? t("Embeddings are priced on input alone. Leave empty if unknown.")
                  : t("From the provider's price list. Leave empty if unknown.")
              }
            />
          </FormControl>

          {!embeds && (
            <FormControl>
              <NumberField
                name="outputCostPerMillion"
                control={control}
                label={t("Output price, USD per million tokens")}
                description={t(
                  "With both prices set, every call and every turn shows what it cost.",
                )}
              />
            </FormControl>
          )}

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
