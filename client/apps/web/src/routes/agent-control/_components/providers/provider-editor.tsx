import {
  fetchAIProvider,
  testAIProviderDraft,
  type AIProviderDraftTestResult,
  type AIProviderModelOption,
  type AIProviderRow,
  type AIProviderWithLimits,
  type AITaskRoute,
} from "@/lib/graphql/ai-provider";
import { queries } from "@/lib/queries";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { FormPanelSection, type ChangeFields } from "@/components/form-changes";
import { FieldWrapper, type FieldLayout } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { ToggleChipsField } from "@/components/fields/toggle-chips-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { JsonEditorField } from "@/components/fields/json-editor-field";
import { apiService } from "@/services/api";
import type {
  AIProviderCatalog,
  AIProviderPreset,
  AITask,
  EmbeddingInputStyle,
  ReasoningEffort,
  StructuredOutputMode,
  ThinkingStyle,
} from "@/types/ai-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  BracketsCheckIcon,
  BracketsIcon,
  Coins01Icon,
  Dataflow01Icon,
  Edit05Icon,
  MessageTextSquare01Icon,
  Power01Icon,
  RulerIcon,
  SearchLgIcon,
  Settings01Icon,
  SlashCircle01Icon,
  Speedometer01Icon,
  Speedometer02Icon,
  Speedometer03Icon,
  Speedometer04Icon,
} from "@trenova/shared/components/icons";
import type { SelectOption } from "@trenova/shared/types/fields";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import { useCallback, useMemo, useState, type ReactNode } from "react";
import {
  useController,
  useForm,
  useFormContext,
  useWatch,
  type Control,
  type FieldPath,
  type Resolver,
} from "react-hook-form";
import type { EditSection } from "../edit/edit-sheet";
import { Callout } from "../edit/callout";
import { aicFieldTrigger } from "../edit/field-trigger";
import { SwitchRow } from "../edit/setting-row";
import { Ic } from "../kit/ic";
import { Mark } from "../kit/marks";
import { groupPresets, presetDisplayName } from "./preset-options";
import { ConfirmDialog } from "../kit/modal";
import {
  cachePriceDefault,
  draftTestInput,
  editorProblem,
  editorValuesFromPreset,
  editorValuesFromProvider,
  providerEditorSchema,
  reportsCacheWrites,
  routingDraft,
  toSaveRequest,
  type KeyRule,
  type ProviderEditorValues,
} from "./provider-editor-model";
import {
  baseUrlProblem,
  formatContext,
  keyField,
  keyPlaceholder,
  type TaskMeta,
} from "./provider-model";
import {
  initialModelView,
  modelPage,
  searchModels,
  splitModels,
  type ModelGroup,
} from "./model-list";
import { kindSupportsEmbedding } from "./provider-form-schema";
import { TASK_FALLBACKS } from "./task-fallbacks";

const ROUTE_PREVIEW_DEBOUNCE_MS = 250;
const FIELD_STACK = "flex flex-col gap-4";
const MODEL_ROW =
  "group grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-2.5 px-2.5 py-1.5 text-left transition-colors hover:bg-foreground/5 aria-checked:bg-brand/7 ui-focus-ring";
const MODEL_DOT =
  "size-3.5 rounded-full border-[1.5px] border-border-strong transition-all group-aria-checked:border-4 group-aria-checked:border-brand";
const PAGE_BUTTON =
  "ui-focus-ring hover:bg-foreground/5 hover:text-foreground inline-flex size-6 items-center justify-center rounded-md transition-colors disabled:pointer-events-none disabled:opacity-40";
/** The panels cache what they save under this key; nothing else reads it. */
const EDITOR_QUERY_KEY = "ai-provider-editor";

/** A saved provider as its editor holds it: the form's values, named by the provider. */
type ProviderEditorRow = ProviderEditorValues & { id: string };
const FIELD_PAIR = "grid grid-cols-2 gap-4";
const DOLLAR = <span className="text-muted-foreground text-xs">$</span>;

/** What the editor is open on: a saved provider, or a new one from a preset. */
export type ProviderEditorTarget =
  | { kind: "edit"; provider: AIProviderWithLimits }
  | { kind: "create"; preset: AIProviderPreset };

type ProviderEditorProps = {
  target: ProviderEditorTarget;
  providers: readonly AIProviderRow[];
  catalog: AIProviderCatalog;
  metas: readonly TaskMeta[];
  keyRequired: KeyRule;
  canDelete: boolean;
  onClose: () => void;
  onRemove: (provider: AIProviderRow) => Promise<void>;
  /** A new provider's preset changed in the panel, so the address can follow it. */
  onPresetChange?: (key: string) => void;
};

type DraftTest = { state: "run" } | { state: "done"; result: AIProviderDraftTestResult };

/**
 * A provider's connection, model, key, tasks, access and limits in the shared editor. It
 * checks the address as it is typed, asks the endpoint for its models, says where each
 * task goes once it is saved, and can try the draft before anything is stored.
 */
export function ProviderEditor({
  target,
  providers,
  catalog,
  metas,
  keyRequired,
  canDelete,
  onClose,
  onRemove,
  onPresetChange,
}: ProviderEditorProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const saved = target.kind === "edit" ? target.provider : null;
  const create = saved === null;
  // A new provider's preset is chosen in the panel, so it is the editor's to change.
  const [preset, setPreset] = useState<AIProviderPreset | null>(
    target.kind === "create" ? target.preset : null,
  );
  const [loaded] = useState(() =>
    target.kind === "edit"
      ? editorValuesFromProvider(target.provider)
      : editorValuesFromPreset(target.preset, providers),
  );
  const resolver = useMemo<Resolver<ProviderEditorValues>>(() => {
    const schema = zodResolver(providerEditorSchema) as Resolver<ProviderEditorValues>;
    return async (draft, context, options) => {
      const result = await schema(draft, context, options);
      const problem = editorProblem(draft, {
        create: target.kind === "create",
        keyRequired: keyRequired(draft.kind) || Boolean(preset?.requiresApiKey),
        hasStoredKey: target.kind === "edit" && Boolean(target.provider.hasApiKey),
        requiresBaseUrl:
          catalog.kinds.find((entry) => entry.kind === draft.kind)?.requiresBaseUrl ?? false,
      });
      if (!problem || problem.field in result.errors) {
        return result;
      }
      return {
        values: {},
        errors: {
          ...result.errors,
          [problem.field]: { type: "validate", message: problem.message },
        },
      };
    };
  }, [catalog.kinds, keyRequired, preset, target]);
  const form = useForm<ProviderEditorValues>({
    resolver,
    defaultValues: loaded,
    mode: "onChange",
  });
  const values = useWatch({ control: form.control }) as ProviderEditorValues;
  const [ran, setRan] = useState<{ connection: string; test: DraftTest } | null>(null);
  const [removing, setRemoving] = useState(false);
  const [removeBusy, setRemoveBusy] = useState(false);

  const kindLabel = useCallback(
    (kind: string) => catalog.kinds.find((entry) => entry.kind === kind)?.label ?? kind,
    [catalog.kinds],
  );
  const presetGroups = useMemo(
    () =>
      groupPresets(catalog.presets).map((group) => ({
        label: group.key === "hosted" ? t("Hosted") : t("On your network"),
        options: group.presets.map((entry) => ({
          value: entry.key,
          label: presetDisplayName(entry),
          description: entry.selfHosted ? entry.baseUrl : entry.exampleModel,
          icon: <Mark provider={{ name: presetDisplayName(entry) }} preset={entry} s={16} />,
        })),
      })),
    [catalog.presets, t],
  );
  // Choosing another preset starts the draft over from that preset's settings, as
  // picking it from the list used to.
  const choosePreset = useCallback(
    (key: string) => {
      const next = catalog.presets.find((entry) => entry.key === key);
      if (!next || next.key === preset?.key) return;
      setPreset(next);
      setRan(null);
      form.reset(editorValuesFromPreset(next, providers));
      onPresetChange?.(next.key);
    },
    [catalog.presets, form, onPresetChange, preset?.key, providers],
  );
  const presetNeedsKey = Boolean(preset?.requiresApiKey);
  const keyMandatory = keyRequired(values.kind) || presetNeedsKey;
  const needsKeyField = keyField({
    mandatory: keyMandatory,
    hasStoredKey: Boolean(saved?.hasApiKey),
    baseUrl: values.baseUrl,
  });
  const hasStoredKey = Boolean(saved?.hasApiKey);
  const requiresBaseUrl =
    catalog.kinds.find((entry) => entry.kind === values.kind)?.requiresBaseUrl ?? false;
  const editRow = useMemo<ProviderEditorRow | undefined>(
    () => (saved ? { ...loaded, id: saved.id } : undefined),
    [loaded, saved],
  );
  const connection = [
    values.kind,
    values.baseUrl,
    values.model,
    values.apiKey,
    values.allowPrivateNetwork,
  ].join("|");
  const test = ran?.connection === connection ? ran.test : null;
  const testOk = test?.state === "done" && test.result.success;

  const save = useCallback(
    async (next: ProviderEditorValues) => {
      const request = toSaveRequest(next, { editing: !create, enable: testOk });
      const result = saved
        ? await apiService.aiProviderService.update(saved.id, request)
        : await apiService.aiProviderService.create(request);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.aiProvider._def }),
        queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
      ]);
      if (saved) {
        const fresh = await fetchAIProvider(result.id);
        if (fresh) {
          form.reset(editorValuesFromProvider(fresh));
        }
      }
      return result;
    },
    [create, form, queryClient, saved, testOk],
  );

  const runTest = async () => {
    const tested = connection;
    setRan({ connection: tested, test: { state: "run" } });
    try {
      const result = await testAIProviderDraft(draftTestInput(values, saved?.id ?? null));
      setRan({ connection: tested, test: { state: "done", result } });
    } catch (error: unknown) {
      setRan({
        connection: tested,
        test: {
          state: "done",
          result: {
            success: false,
            message: t("The test could not run"),
            detail: error instanceof Error ? error.message : "",
            hint: "",
            modelIdentifier: "",
            schemaHonoured: false,
            latencyMs: 0,
          },
        },
      });
    }
  };

  const models = useEndpointModels(form.control, saved?.id ?? null, !create || !needsKeyField);

  const index = saved ? providers.findIndex((provider) => provider.id === saved.id) : -1;
  const taskLabel = useCallback(
    (task: string | number) => metas.find((meta) => meta.task === task)?.label ?? String(task),
    [metas],
  );
  const changeFields: ChangeFields = {
    name: { label: t("Name") },
    kind: { label: t("Kind"), format: (value) => kindLabel(String(value)) },
    baseUrl: { label: t("Base URL") },
    model: { label: t("Model") },
    apiKey: {
      label: t("API key"),
      format: (value) =>
        typeof value === "string" && value !== ""
          ? t("New key ••••{0}", value.slice(-4))
          : t("Unchanged"),
    },
    keepPreviousKey: { label: t("Keep old key 24h") },
    tasks: { label: t("Handles"), item: taskLabel },
    trusted: { label: t("Trusted") },
    allowPrivateNetwork: { label: t("Private network") },
    timeoutSeconds: { label: t("Timeout"), format: (value) => t("{0}s", String(value)) },
    maxConcurrent: { label: t("Concurrent calls") },
    monthlyCapUsd: {
      label: t("Monthly cap"),
      format: (value) => (value === "" ? t("None") : `$${String(value)}`),
    },
    onCap: {
      label: t("At the cap"),
      format: (value) => (value === "Stop" ? t("Stop") : t("Hand to next")),
    },
    inputCostPerMillion: { label: t("Input price") },
    outputCostPerMillion: { label: t("Output price") },
    cacheReadCostPerMillion: {
      label: t("Cache read price"),
      format: (value) => (value === "" ? t("Default") : `$${String(value)}`),
    },
    cacheWriteCostPerMillion: {
      label: t("Cache write price"),
      format: (value) => (value === "" ? t("Default") : `$${String(value)}`),
    },
    description: { label: t("Description") },
    structuredOutputMode: { label: t("Structured output") },
    reasoningEffort: { label: t("Reasoning") },
    thinkingStyle: { label: t("Thinking") },
    maxTokens: { label: t("Most tokens per reply") },
    embeddingDimensionsChoice: { label: t("Vector size") },
    embeddingInputStyle: { label: t("Embedding input") },
    extraBodyText: { label: t("Extra request fields") },
  };
  const footerProblem =
    editorProblem(values, {
      create,
      keyRequired: keyMandatory,
      hasStoredKey,
      requiresBaseUrl,
    })?.message ?? null;

  const urlProblem = baseUrlProblem(values.baseUrl, values.allowPrivateNetwork);
  const trustNeeded = metas.some((meta) => meta.trust && values.tasks.includes(meta.task));
  const trustLabels = metas
    .filter((meta) => meta.trust && values.tasks.includes(meta.task))
    .map((meta) => meta.label);
  const taskOptions = useMemo(
    () =>
      metas.map((meta) => ({
        value: meta.task,
        label: meta.label,
        mark: meta.trust ? <Ic n="shield" s={10} /> : undefined,
      })),
    [metas],
  );

  const sections: EditSection[] = [
    {
      id: "conn",
      label: t("Connection"),
      note: t("Where requests go and which API answers them."),
      keys: ["name", "kind", "baseUrl"],
      warning:
        urlProblem === "private"
          ? t(
              "This address is on your own network. Turn on Private network so Trenova can reach it.",
            )
          : undefined,
      content: (
        <div className={FIELD_STACK}>
          {create ? (
            <SelectField<ProviderEditorValues>
              control={form.control}
              name="preset"
              rules={{ required: true }}
              description={t("Where it runs. Choosing one fills in its usual settings.")}
              label={t("Provider")}
              placeholder={t("Choose a provider")}
              groups={presetGroups}
              onValueChange={choosePreset}
              triggerClassName={aicFieldTrigger}
              layout="inline"
            />
          ) : null}
          <InputField<ProviderEditorValues>
            control={form.control}
            name="name"
            rules={{ required: true }}
            description={t("What this provider is called across Trenova.")}
            label={t("Name")}
            autoFocus={create}
            inputClassProps={aicFieldTrigger}
            layout="inline"
          />
          <SelectField<ProviderEditorValues>
            control={form.control}
            name="kind"
            rules={{ required: true }}
            description={t("The API this endpoint speaks.")}
            label={t("Kind")}
            placeholder={t("Kind")}
            options={catalog.kinds.map((entry) => ({
              value: entry.kind,
              label: entry.label,
              description: entry.description,
              icon: <Mark provider={{ name: entry.label, kind: entry.kind }} s={16} />,
            }))}
            triggerClassName={aicFieldTrigger}
            layout="inline"
          />
          <EditorTextInput
            name="baseUrl"
            layout="inline"
            required={requiresBaseUrl}
            label={t("Base URL")}
            placeholder="https://api.example.com/v1"
            error={
              urlProblem === "private"
                ? t(
                    "This address is on your own network. Turn on Private network so Trenova can reach it.",
                  )
                : urlProblem === "scheme"
                  ? t("Start with http:// or https://")
                  : undefined
            }
            description={
              values.baseUrl.trim() === ""
                ? requiresBaseUrl
                  ? t("This kind of endpoint has no default; give its address.")
                  : t("Leave empty to use {0}'s default endpoint.", kindLabel(values.kind))
                : t("Where requests to this provider are sent.")
            }
          />
          {urlProblem === "private" && (
            <div className="grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)] gap-x-4">
              <button
                type="button"
                className="text-muted-foreground hover:text-foreground ui-focus-ring col-start-2 justify-self-start rounded-sm text-sm transition-colors"
                onClick={() =>
                  form.setValue("allowPrivateNetwork", true, {
                    shouldDirty: true,
                    shouldValidate: true,
                  })
                }
              >
                {t("Turn on Private network")}
              </button>
            </div>
          )}
        </div>
      ),
    },
    {
      id: "model",
      label: t("Model"),
      note: t("The model every task sent here runs on."),
      help: t(
        "Fetch the list to see what this endpoint actually serves. When it reports a model's context window and price, they're filled in for you.",
      ),
      keys: ["model"],
      actions: (
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={models.fetching}
          onClick={models.fetch}
        >
          {models.fetching ? (
            <>
              <span className="border-border-strong border-t-foreground size-3 animate-spin rounded-full border-[1.5px]" />
              {t("Asking {0}…", kindLabel(values.kind))}
            </>
          ) : (
            <>
              <Ic n="refresh" s={12} />
              {models.list ? t("Refresh") : t("Fetch models")}
            </>
          )}
        </Button>
      ),
      content: <ModelSection models={models} local={!needsKeyField} />,
    },
    ...(needsKeyField
      ? [
          {
            id: "key",
            label: t("API key"),
            note: t("The credential Trenova signs each request with."),
            help: t(
              "A new key takes effect when you save. Keep the old one working for a day if other systems still use it.",
            ),
            keys: ["apiKey", "keepPreviousKey"],
            content: (
              <div className={FIELD_STACK}>
                {saved?.apiKey && values.apiKey === "" && <StoredKey info={saved.apiKey} />}
                <InputField<ProviderEditorValues>
                  control={form.control}
                  name="apiKey"
                  rules={keyMandatory && !hasStoredKey ? { required: true } : undefined}
                  type="password"
                  autoComplete="new-password"
                  label={hasStoredKey ? t("Replace key") : t("Key")}
                  description={t("Stored encrypted and never shown again.")}
                  placeholder={keyPlaceholder(values.kind, values.baseUrl) ?? t("API key")}
                  inputClassProps={cn(aicFieldTrigger, "font-mono")}
                />
                {hasStoredKey && values.apiKey !== "" && (
                  <SwitchRow<ProviderEditorValues>
                    control={form.control}
                    name="keepPreviousKey"
                    label={t("Keep the old key working for 24 hours")}
                    note={t("So nothing fails while other systems switch over.")}
                  />
                )}
              </div>
            ),
          } satisfies EditSection,
        ]
      : []),
    {
      id: "tasks",
      label: t("What it handles"),
      keys: ["tasks"],
      note: t("Each task goes to the first provider in order that takes it."),
      content: (
        <div className={FIELD_STACK}>
          <ToggleChipsField<ProviderEditorValues, AITask>
            control={form.control}
            name="tasks"
            label={t("What it handles")}
            options={taskOptions}
          />
          <RouteImpact providerId={saved?.id ?? null} metas={metas} providers={providers} />
        </div>
      ),
    },
    {
      id: "access",
      label: t("Access"),
      note: t("What this provider may read and reach."),
      help: t(
        "Only a trusted provider takes tasks that read sensitive records, such as billing diagnosis. Private network lets Trenova call a server inside your own network, such as a self-hosted model.",
      ),
      keys: ["trusted", "allowPrivateNetwork"],
      content: (
        <div className={FIELD_STACK}>
          <SwitchField<ProviderEditorValues>
            control={form.control}
            name="trusted"
            label={t("Trusted")}
            description={t("May take tasks that read sensitive records, like billing diagnosis.")}
          />
          <SwitchField<ProviderEditorValues>
            control={form.control}
            name="allowPrivateNetwork"
            label={t("Private network")}
            description={t("May reach a server on your own network.")}
          />
          {trustNeeded && !values.trusted && (
            <Callout
              tone="w"
              action={
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => form.setValue("trusted", true, { shouldDirty: true })}
                >
                  {t("Trust it")}
                </Button>
              }
            >
              {t(
                "{0} needs a trusted provider. It's assigned here but will be skipped.",
                trustLabels.join(", "),
              )}
            </Callout>
          )}
        </div>
      ),
    },
    {
      id: "limits",
      label: t("Limits and price"),
      note: t("How hard it may be used and what each call costs."),
      help: t(
        "Prices let Trenova report spend and enforce the monthly cap. Leave them empty for a self-hosted model you don't pay for per token.",
      ),
      keys: [
        "timeoutSeconds",
        "maxConcurrent",
        "monthlyCapUsd",
        "onCap",
        "inputCostPerMillion",
        "outputCostPerMillion",
        "cacheReadCostPerMillion",
        "cacheWriteCostPerMillion",
      ],
      content: (
        <FormGroup cols={1}>
          <FormControl className="min-h-2">
            <InputField<ProviderEditorValues>
              control={form.control}
              name="timeoutSeconds"
              rules={{ required: true }}
              description={t("How long one call may run before it's abandoned.")}
              type="number"
              label={t("Timeout")}
              sideText={t("seconds")}
              inputClassProps={aicFieldTrigger}
              layout="inline"
            />
          </FormControl>
          <FormControl className="min-h-2">
            <InputField<ProviderEditorValues>
              control={form.control}
              name="maxConcurrent"
              rules={{ required: true }}
              description={t("How many calls it may handle at the same time.")}
              type="number"
              label={t("Concurrent calls")}
              sideText={t("at once")}
              inputClassProps={aicFieldTrigger}
              layout="inline"
            />
          </FormControl>
          <FormControl className="min-h-2">
            <InputField<ProviderEditorValues>
              control={form.control}
              name="monthlyCapUsd"
              type="number"
              label={t("Monthly spend cap")}
              description={
                values.monthlyCapUsd.trim() === ""
                  ? t("No cap")
                  : saved
                    ? t("${0} spent this month", Number(saved.monthSpendUsd).toFixed(2))
                    : t("Counted from the first call")
              }
              leftElement={DOLLAR}
              placeholder={t("None")}
              inputClassProps={aicFieldTrigger}
              layout="inline"
            />
          </FormControl>
          <FormControl className="min-h-2">
            <SegmentedField<ProviderEditorValues, ProviderEditorValues["onCap"]>
              control={form.control}
              name="onCap"
              description={t("What happens once this month's cap is reached.")}
              label={t("At the cap")}
              options={[
                { value: "Next", label: t("Hand to next") },
                { value: "Stop", label: t("Stop") },
              ]}
              layout="inline"
            />
          </FormControl>
          <FormControl className="min-h-2">
            <InputField<ProviderEditorValues>
              control={form.control}
              name="inputCostPerMillion"
              inputMode="decimal"
              label={t("Input price")}
              description={t("Per million tokens")}
              leftElement={DOLLAR}
              placeholder="—"
              inputClassProps={aicFieldTrigger}
              layout="inline"
            />
          </FormControl>
          <FormControl className="min-h-2">
            <InputField<ProviderEditorValues>
              control={form.control}
              name="outputCostPerMillion"
              inputMode="decimal"
              label={t("Output price")}
              description={t("Per million tokens")}
              leftElement={DOLLAR}
              placeholder="—"
              inputClassProps={aicFieldTrigger}
              layout="inline"
            />
          </FormControl>
          <FormControl className="min-h-2">
            <InputField<ProviderEditorValues>
              control={form.control}
              name="cacheReadCostPerMillion"
              inputMode="decimal"
              label={t("Cache read price")}
              description={t(
                "Per million prompt tokens served from the cache. Empty charges a tenth of the input price.",
              )}
              leftElement={DOLLAR}
              placeholder={
                cachePriceDefault(values.kind, values.inputCostPerMillion, "read") || "—"
              }
              inputClassProps={aicFieldTrigger}
              layout="inline"
            />
          </FormControl>
          {reportsCacheWrites(values.kind) && (
            <FormControl className="min-h-2">
              <InputField<ProviderEditorValues>
                control={form.control}
                name="cacheWriteCostPerMillion"
                inputMode="decimal"
                label={t("Cache write price")}
                description={t(
                  "Per million prompt tokens written to the cache. Empty charges 1.25 times the input price.",
                )}
                leftElement={DOLLAR}
                placeholder={
                  cachePriceDefault(values.kind, values.inputCostPerMillion, "write") || "—"
                }
                inputClassProps={aicFieldTrigger}
                layout="inline"
              />
            </FormControl>
          )}
        </FormGroup>
      ),
    },
    {
      id: "advanced",
      label: t("Advanced"),
      help: t(
        "Most models work with the defaults. Change these only when the endpoint's documentation asks for it.",
      ),
      keys: [
        "description",
        "structuredOutputMode",
        "reasoningEffort",
        "thinkingStyle",
        "maxTokens",
        "embeddingDimensionsChoice",
        "embeddingInputStyle",
        "extraBodyText",
      ],
      note: t("How requests are shaped for this endpoint."),
      content: <AdvancedFields catalog={catalog} />,
    },
    ...(saved && canDelete
      ? [
          {
            id: "danger",
            label: t("Remove"),
            note: t("Take it out of the chain once nothing should use it."),
            content: (
              <div className="border-danger-border flex items-center justify-between gap-4 rounded-lg border px-3 py-2.5">
                <div className="flex flex-col gap-0.5">
                  <b className="text-base font-medium">{t("Remove {0}", saved.name)}</b>
                  <span className="text-muted-foreground text-sm">
                    {saved.tasks.length === 0
                      ? t("It handles no tasks.")
                      : saved.tasks.length === 1
                        ? t("Its 1 task falls to the next provider in line.")
                        : t("Its {0} tasks fall to the next provider in line.", saved.tasks.length)}
                  </span>
                </div>
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  onClick={() => setRemoving(true)}
                >
                  {t("Remove provider")}
                </Button>
              </div>
            ),
          } satisfies EditSection,
        ]
      : []),
  ];

  const mark = {
    name: values.name || saved?.name || preset?.label || "",
    kind: values.kind,
    baseUrl: values.baseUrl,
  };
  const failed = test?.state === "done" && !test.result.success ? test.result : null;

  const testControl = (
    <div className="flex min-w-0 items-center gap-2">
      <Button
        type="button"
        size="sm"
        variant="outline"
        disabled={test?.state === "run" || values.model.trim() === ""}
        onClick={() => void runTest()}
      >
        <Ic n="plug" s={12} />
        {t("Test draft")}
      </Button>
      {test && (
        <span
          className={cn(
            "inline-flex min-w-0 items-center gap-1.5 truncate text-sm",
            test.state === "run"
              ? "text-muted-foreground"
              : test.result.success
                ? "text-success-foreground"
                : "text-danger-foreground",
          )}
          title={test.state === "done" ? test.result.detail : undefined}
        >
          {test.state === "run" ? (
            t("Testing…")
          ) : (
            <>
              <Ic n={test.result.success ? "check" : "x"} s={11} w={2.4} />
              {test.result.success
                ? t("Connected · {0} ms", test.result.latencyMs)
                : test.result.message}
            </>
          )}
        </span>
      )}
    </div>
  );

  const body = (
    <div className="flex flex-col gap-5">
      {failed && (
        <Alert variant="destructive" size="sm">
          <AlertCircleIcon />
          <AlertTitle>{failed.message}</AlertTitle>
          {(failed.detail || failed.hint) && (
            <AlertDescription>
              {failed.detail && <span className="font-mono break-all">{failed.detail}</span>}
              {failed.hint && <span>{failed.hint}</span>}
            </AlertDescription>
          )}
        </Alert>
      )}
      <div className="divide-border-subtle flex flex-col divide-y">
        {sections.map((section) => (
          <FormPanelSection
            key={section.id}
            id={`pe-${section.id}`}
            title={section.label}
            note={section.note}
            fields={section.keys}
            actions={section.actions}
            help={section.help}
          >
            {section.warning && (
              <Alert variant="warning" size="sm">
                <AlertCircleIcon />
                <AlertDescription>{section.warning}</AlertDescription>
              </Alert>
            )}
            {section.content}
          </FormPanelSection>
        ))}
      </div>
    </div>
  );

  const markIcon = <Mark provider={mark} preset={preset} s={28} />;
  const closeOnDismiss = (next: boolean) => {
    if (!next) onClose();
  };

  return (
    <>
      {saved ? (
        <FormEditPanel<ProviderEditorValues, ProviderEditorRow, ProviderEditorValues, unknown>
          open
          onOpenChange={closeOnDismiss}
          row={editRow ?? null}
          form={form}
          title={t("AI provider")}
          queryKey={EDITOR_QUERY_KEY}
          size="xl"
          titleComponent={() => (
            <span className="flex items-center gap-3">
              {markIcon}
              {t("Edit {0}", saved.name)}
            </span>
          )}
          subtitle={() => t("Priority {0} of {1}", index + 1, providers.length)}
          formComponent={body}
          changeFields={changeFields}
          footerProblem={footerProblem}
          footerLeading={testControl}
          mutationFn={(next) => save(next)}
        />
      ) : (
        <FormCreatePanel<ProviderEditorValues, ProviderEditorRow, ProviderEditorValues, unknown>
          open
          onOpenChange={closeOnDismiss}
          form={form}
          title={t("AI provider")}
          queryKey={EDITOR_QUERY_KEY}
          size="xl"
          description={
            needsKeyField ? t("Hosted · needs an API key") : t("On your network · no key")
          }
          notice={
            <div className="flex items-center gap-3">
              {markIcon}
              <span className="text-base font-medium">
                {values.name.trim() || t("New provider")}
              </span>
            </div>
          }
          formComponent={body}
          changeFields={changeFields}
          footerProblem={footerProblem}
          footerLeading={testControl}
          mutationFn={(next) => save(next)}
        />
      )}
      {saved && (
        <ConfirmDialog
          open={removing}
          onClose={() => setRemoving(false)}
          title={t("Remove {0}?", saved.name)}
          description={
            saved.tasks.length > 0
              ? t("Its tasks fall to the next provider in line. The stored key is deleted.")
              : t("The stored key is deleted.")
          }
          confirmLabel={t("Remove provider")}
          danger
          busy={removeBusy}
          onConfirm={() => {
            setRemoveBusy(true);
            onRemove(saved)
              .then(() => {
                setRemoving(false);
                onClose();
              })
              .catch(() => undefined)
              .finally(() => setRemoveBusy(false));
          }}
        />
      )}
    </>
  );
}

function useEditorField<K extends FieldPath<ProviderEditorValues>>(name: K) {
  const { control } = useFormContext<ProviderEditorValues>();
  const { field } = useController({ control, name });
  return [field.value, field.onChange] as const;
}

type EditorTextInputProps = {
  name: "baseUrl" | "model";
  label: string;
  layout?: FieldLayout;
  required?: boolean;
  description?: string;
  error?: string;
  placeholder?: string;
};

/**
 * A text box whose message the editor works out from more than the schema: whether an
 * address is reachable, or what the endpoint said when asked for its models.
 */
function EditorTextInput({
  name,
  label,
  description,
  error: told,
  placeholder,
  layout,
  required = false,
}: EditorTextInputProps) {
  const { control } = useFormContext<ProviderEditorValues>();
  const {
    field: { value, onChange, onBlur, ref },
    fieldState,
  } = useController({ control, name });
  const error = told ?? fieldState.error?.message;
  const inputId = `input-${name}`;

  return (
    <FieldWrapper
      name={name}
      layout={layout}
      required={required}
      label={label}
      description={description}
      error={error}
      descriptionId={`${inputId}-description`}
      errorId={`${inputId}-error`}
    >
      <Input
        id={inputId}
        ref={ref}
        name={name}
        value={value}
        placeholder={placeholder}
        onBlur={onBlur}
        aria-invalid={error ? true : undefined}
        aria-required={required || undefined}
        aria-describedby={
          error ? `${inputId}-error` : description ? `${inputId}-description` : undefined
        }
        onChange={(event) => onChange(event.target.value)}
        className={aicFieldTrigger}
      />
    </FieldWrapper>
  );
}

/** The current key, described: its ends, who added it, when it was last used. */
function StoredKey({ info }: { info: NonNullable<AIProviderWithLimits["apiKey"]> }) {
  const t = useT();
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const added = info.addedBy?.name
    ? t("Added {0} by {1}", formatUnixDateMedium(info.addedAt), info.addedBy.name)
    : t("Added {0}", formatUnixDateMedium(info.addedAt));
  const used = info.lastUsedAt
    ? t("last used {0}", formatRelativeTime(info.lastUsedAt - now))
    : t("not used yet");
  const previous =
    info.previousKeyExpiresAt && info.previousKeyExpiresAt > now
      ? t("old key works until {0}", formatUnixDateTimeShort(info.previousKeyExpiresAt))
      : null;
  return (
    <div className="bg-field flex flex-wrap items-center gap-x-2.5 gap-y-1.5 rounded-lg px-3 py-2.5 text-base">
      <span className="text-success">
        <Ic n="key" s={13} />
      </span>
      <span className="min-w-0 font-mono break-all">{`${info.prefix}••••••••${info.lastFour}`}</span>
      <span className="text-muted-foreground basis-full pl-5.75 text-sm">
        {[added, used, previous].filter(Boolean).join(" · ")}
      </span>
    </div>
  );
}

type EndpointModels = {
  list: AIProviderModelOption[] | undefined;
  fetching: boolean;
  error: string | null;
  fetch: () => void;
};

/**
 * The models the endpoint as typed says it serves. Nothing is asked until someone fetches,
 * or at once for a new endpoint on a person's own network, and a list fetched for an
 * address that has since changed is put away rather than offered for the new one.
 */
function useEndpointModels(
  control: Control<ProviderEditorValues>,
  providerId: string | null,
  autoFetch: boolean,
): EndpointModels {
  const t = useT();
  const [kind, baseUrl, apiKey, allowPrivateNetwork] = useWatch({
    control,
    name: ["kind", "baseUrl", "apiKey", "allowPrivateNetwork"],
  });
  const endpoint = useMemo(
    () => ({
      providerId,
      kind,
      baseUrl: baseUrl.trim(),
      apiKey: apiKey.trim() === "" ? null : apiKey.trim(),
      allowPrivateNetwork,
    }),
    [allowPrivateNetwork, apiKey, baseUrl, kind, providerId],
  );
  const [asked, setAsked] = useState<typeof endpoint | null>(autoFetch ? endpoint : null);
  const query = useQuery({
    ...queries.aiProvider.models(asked ?? endpoint),
    enabled: asked !== null,
    staleTime: 60_000,
    retry: false,
  });
  const current = asked !== null && asked === endpoint;

  return {
    list: current ? query.data : undefined,
    fetching: current && query.isFetching,
    error:
      current && query.isError
        ? query.error instanceof Error
          ? query.error.message
          : t("The endpoint did not answer")
        : null,
    fetch: () => {
      if (asked === endpoint) {
        void query.refetch();
        return;
      }
      setAsked(endpoint);
    },
  };
}

/** The model: picked from what the endpoint serves once fetched, typed otherwise. */
function ModelSection({ models, local }: { models: EndpointModels; local: boolean }) {
  const t = useT();
  const [model, setModel] = useEditorField("model");
  const list = models.list;

  if (list && list.length > 0) {
    return <ModelPicker list={list} model={model} onPick={setModel} local={local} />;
  }

  return (
    <div>
      <EditorTextInput
        name="model"
        required
        label={t("Model ID")}
        error={
          models.error ??
          (list && list.length === 0
            ? t("The endpoint lists no models. Type the ID it serves.")
            : undefined)
        }
        description={t("Or fetch the list this provider actually serves.")}
      />
    </div>
  );
}

type ModelPickerProps = {
  list: readonly AIProviderModelOption[];
  model: string;
  onPick: (id: string) => void;
  local: boolean;
};

/**
 * The endpoint's models, latest and older apart, searched and shown a page at a time.
 * It opens where the chosen model is, and a model the endpoint no longer lists stays
 * pinned above the list.
 */
function ModelPicker({ list, model, onPick, local }: ModelPickerProps) {
  const t = useT();
  const split = useMemo(() => splitModels(list), [list]);
  const [view, setView] = useState(() => initialModelView(split, model));
  const [search, setSearch] = useState("");
  const shown = useMemo(
    () => searchModels(split.split ? split[view.group] : split.latest, search),
    [search, split, view.group],
  );
  const page = modelPage(shown, view.page);
  const current = model.trim();
  const unlisted = current !== "" && !list.some((option) => option.id === current);

  const metaOf = (option: AIProviderModelOption) => {
    const parts: string[] = [];
    if (option.embedding) parts.push(t("Embedding"));
    if (option.contextWindow) {
      parts.push(t("{0} context", formatContext(option.contextWindow)));
    }
    if (option.sizeBytes) parts.push(formatFileSize(option.sizeBytes));
    if (option.loaded) parts.push(t("loaded"));
    if (option.createdAt) parts.push(formatUnixDateMedium(option.createdAt));
    return parts.join(" · ");
  };
  const priceOf = (option: AIProviderModelOption) => {
    const estimate = option.priceSource === "OpenRouter" ? "~" : "";
    if (option.inputCostPerMillion && option.outputCostPerMillion) {
      return `${estimate}$${Number(option.inputCostPerMillion)} / $${Number(option.outputCostPerMillion)}`;
    }
    if (option.inputCostPerMillion) return `${estimate}$${Number(option.inputCostPerMillion)}`;
    return local ? t("Local") : null;
  };
  const groupLabel = (group: ModelGroup, count: number) => (
    <span className="inline-flex items-center gap-1.5">
      {group === "latest" ? t("Latest") : t("Older")}
      <span className="text-muted-foreground font-mono text-xs">{count}</span>
    </span>
  );

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        {split.split && (
          <SegmentedControl<ModelGroup>
            aria-label={t("Models")}
            value={view.group}
            items={[
              { value: "latest", label: groupLabel("latest", split.latest.length) },
              { value: "older", label: groupLabel("older", split.older.length) },
            ]}
            onValueChange={(group) => setView({ group, page: 0 })}
          />
        )}
        <Input
          type="search"
          value={search}
          aria-label={t("Search models")}
          placeholder={t("Search {0} models", list.length)}
          inputContainerClassName="w-full max-w-80 flex-1"
          leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
          onChange={(event) => {
            setSearch(event.target.value);
            setView((previous) => ({ ...previous, page: 0 }));
          }}
        />
      </div>
      <ScrollArea
        className="border-border rounded-lg border"
        viewportClassName="max-h-72"
        maskVariant="background"
      >
        <div className="flex flex-col" role="radiogroup" aria-label={t("Model")}>
          {unlisted && (
            <button type="button" role="radio" aria-checked className={MODEL_ROW}>
              <span className={MODEL_DOT} />
              <span className="flex min-w-0 flex-col">
                <b className="truncate font-mono text-sm font-medium">{current}</b>
                <span className="text-muted-foreground text-xs">
                  {t("Not listed by the endpoint")}
                </span>
              </span>
            </button>
          )}
          {page.items.map((option) => (
            <button
              key={option.id}
              type="button"
              role="radio"
              aria-checked={model === option.id}
              className={MODEL_ROW}
              onClick={() => onPick(option.id)}
            >
              <span className={MODEL_DOT} />
              <span className="flex min-w-0 flex-col">
                <b className="truncate font-mono text-sm font-medium">{option.id}</b>
                <span className="text-muted-foreground text-xs">{metaOf(option)}</span>
              </span>
              {priceOf(option) && (
                <span className="text-muted-foreground font-mono text-xs">{priceOf(option)}</span>
              )}
            </button>
          ))}
          {page.total === 0 && (
            <p className="text-muted-foreground m-0 px-3 py-4.5 text-center text-sm">
              {search.trim()
                ? t("No models match “{0}”.", search.trim())
                : t("No models in this group.")}
            </p>
          )}
        </div>
      </ScrollArea>
      {page.items.some((option) => option.priceSource === "OpenRouter") && (
        <p className="text-muted-foreground m-0 text-xs">
          {t("~ Estimated list price per million tokens, from OpenRouter's public catalog.")}
        </p>
      )}
      {page.total > 0 && (
        <div className="text-muted-foreground flex items-center justify-end gap-0.5 text-xs">
          <span className="mr-1.5 font-mono">
            {t("{0}–{1} of {2}", page.from, page.to, page.total)}
          </span>
          <button
            type="button"
            className={PAGE_BUTTON}
            aria-label={t("Previous page")}
            disabled={page.page === 0}
            onClick={() => setView((previous) => ({ ...previous, page: page.page - 1 }))}
          >
            <Ic n="chevL" s={13} />
          </button>
          <button
            type="button"
            className={PAGE_BUTTON}
            aria-label={t("Next page")}
            disabled={page.page >= page.pageCount - 1}
            onClick={() => setView((previous) => ({ ...previous, page: page.page + 1 }))}
          >
            <Ic n="chevR" s={13} />
          </button>
        </div>
      )}
    </div>
  );
}

/** Where each task goes once the editor is saved, for the tasks that change. */
function RouteImpact({
  providerId,
  metas,
  providers,
}: {
  providerId: string | null;
  metas: readonly TaskMeta[];
  providers: readonly AIProviderRow[];
}) {
  const t = useT();
  const { control } = useFormContext<ProviderEditorValues>();
  const values = useWatch({ control }) as ProviderEditorValues;
  const draft = useDebounce(
    useMemo(() => routingDraft(values, providerId), [providerId, values]),
    ROUTE_PREVIEW_DEBOUNCE_MS,
  );
  const preview = useQuery({
    ...queries.aiProvider.routePreview(draft),
    placeholderData: (previous) => previous,
    staleTime: 30_000,
  });
  const labels = new Map(metas.map((meta) => [meta.task, meta.label]));
  const byId = new Map(providers.map((provider) => [provider.id, provider]));
  const rows = (preview.data ?? []).filter((route) => route.changed);

  if (!preview.data) return null;
  if (rows.length === 0) {
    return (
      <p className="text-muted-foreground m-0 flex items-center gap-1.5 text-sm">
        <span className="text-success">
          <Ic n="check" s={12} w={2.2} />
        </span>
        {t("No task changes where it goes.")}
      </p>
    );
  }

  const choice = (route: AITaskRoute["before"], strong: boolean) => {
    if (!route) return null;
    const provider = route.providerId ? byId.get(String(route.providerId)) : undefined;
    const name = route.draft ? values.name.trim() || route.name : (provider?.name ?? route.name);
    const source = route.draft
      ? { name, kind: values.kind, baseUrl: values.baseUrl }
      : { name, kind: provider?.kind, baseUrl: provider?.baseUrl };
    return (
      <>
        <Mark provider={source} s={14} />
        {strong ? <b className="text-foreground font-medium">{name}</b> : name}
      </>
    );
  };

  return (
    <div className="bg-sunken rounded-lg px-3 py-2.5">
      <div className="text-muted-foreground mb-1 text-xs font-medium">{t("When you save")}</div>
      {rows.map((route) => (
        <div
          key={route.task}
          className="grid min-h-7.5 grid-cols-[150px_minmax(0,1fr)] items-center gap-2.5 text-base"
        >
          <span className="flex items-center gap-1.75">
            {(!route.after || !route.before) && (
              <span
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  route.after ? "bg-success" : "bg-warning",
                )}
              />
            )}
            {labels.get(route.task) ?? route.task}
          </span>
          <span className="text-muted-foreground flex flex-wrap items-center gap-1.5">
            {route.before ? choice(route.before, false) : t("Nowhere")}
            <span className="text-muted-foreground/60">
              <Ic n="arrowR" s={11} />
            </span>
            {route.after ? (
              choice(route.after, true)
            ) : (
              <span className="text-warning-foreground">
                {t("Nowhere · {0}", TASK_FALLBACKS[route.task].toLowerCase())}
              </span>
            )}
          </span>
        </div>
      ))}
    </div>
  );
}

/** The request shape: output mode, reasoning, reply size, embeddings and vendor fields. */
function AdvancedFields({ catalog }: { catalog: AIProviderCatalog }) {
  const t = useT();
  const { control } = useFormContext<ProviderEditorValues>();
  const [kind, tasks] = useWatch({ control, name: ["kind", "tasks"] });
  const embeds = tasks.includes("Embedding") && kindSupportsEmbedding(kind);

  const outputModes: SelectOption[] = [
    {
      value: "JSONSchema" satisfies StructuredOutputMode,
      label: t("JSON schema"),
      description: t("The reply is held to the exact shape asked for."),
      icon: <BracketsCheckIcon className="size-4" />,
    },
    {
      value: "JSONMode" satisfies StructuredOutputMode,
      label: t("JSON mode"),
      description: t("The reply is valid JSON, but its shape isn't enforced."),
      icon: <BracketsIcon className="size-4" />,
    },
    {
      value: "Prompted" satisfies StructuredOutputMode,
      label: t("Prompted"),
      description: t("The prompt asks for JSON; nothing enforces it."),
      icon: <MessageTextSquare01Icon className="size-4" />,
    },
  ];
  const efforts: SelectOption[] = [
    {
      value: "Off" satisfies ReasoningEffort,
      label: t("Off"),
      description: t("No reasoning setting is sent; the model decides."),
      icon: <Power01Icon className="size-4" />,
    },
    {
      value: "None" satisfies ReasoningEffort,
      label: t("None"),
      description: t("The model is asked to skip reasoning."),
      icon: <SlashCircle01Icon className="size-4" />,
    },
    {
      value: "Minimal" satisfies ReasoningEffort,
      label: t("Minimal"),
      description: t("The least thinking and the fastest replies."),
      icon: <Speedometer01Icon className="size-4" />,
    },
    {
      value: "Low" satisfies ReasoningEffort,
      label: t("Low"),
      description: t("Brief thinking for simple tasks."),
      icon: <Speedometer02Icon className="size-4" />,
    },
    {
      value: "Medium" satisfies ReasoningEffort,
      label: t("Medium"),
      description: t("A balance of thinking and speed."),
      icon: <Speedometer03Icon className="size-4" />,
    },
    {
      value: "High" satisfies ReasoningEffort,
      label: t("High"),
      description: t("The most thinking and the slowest replies."),
      icon: <Speedometer04Icon className="size-4" />,
    },
  ];
  const thinking: SelectOption[] = [
    {
      value: "Auto" satisfies ThinkingStyle,
      label: t("Read from the model"),
      description: t("Uses whichever style the model supports."),
      icon: <Settings01Icon className="size-4" />,
    },
    {
      value: "Effort" satisfies ThinkingStyle,
      label: t("By effort"),
      description: t("Thinking follows the reasoning level above."),
      icon: <Speedometer03Icon className="size-4" />,
    },
    {
      value: "Budget" satisfies ThinkingStyle,
      label: t("By token budget"),
      description: t("Thinking is capped at a fixed number of tokens."),
      icon: <Coins01Icon className="size-4" />,
    },
  ];
  const inputStyleIcon: Record<EmbeddingInputStyle, ReactNode> = {
    None: <SlashCircle01Icon className="size-4" />,
    VoyageInputType: <Dataflow01Icon className="size-4" />,
    NomicPrefix: <Edit05Icon className="size-4" />,
  };
  const inputStyles: SelectOption[] = catalog.embeddingInputStyles.map((entry) => ({
    value: entry.style,
    label: entry.label,
    description: entry.description,
    icon: inputStyleIcon[entry.style],
  }));
  const dimensionNotes: Record<number, string> = {
    768: t("Smallest index and fastest search."),
    1024: t("A balance of detail and index size."),
    1536: t("The most detail and the largest index."),
  };
  const dimensions: SelectOption[] = catalog.embeddingDimensions.map((size) => ({
    value: String(size),
    label: String(size),
    description: dimensionNotes[size],
    icon: <RulerIcon className="size-4" />,
  }));

  return (
    <div className={FIELD_STACK}>
      <InputField<ProviderEditorValues>
        control={control}
        name="description"
        description={t("A note for your team about this provider.")}
        label={t("Description")}
        inputClassProps={aicFieldTrigger}
      />
      <div className={FIELD_PAIR}>
        <SelectField<ProviderEditorValues>
          control={control}
          name="structuredOutputMode"
          rules={{ required: true }}
          description={t("How the model is asked to return structured data.")}
          label={t("Structured output")}
          placeholder={t("Structured output")}
          options={outputModes}
          triggerClassName={aicFieldTrigger}
        />
        <InputField<ProviderEditorValues>
          control={control}
          name="maxTokens"
          rules={{ required: true }}
          description={t("The longest reply the model may write.")}
          type="number"
          label={t("Most tokens per reply")}
          inputClassProps={aicFieldTrigger}
        />
      </div>
      <div className={FIELD_PAIR}>
        <SelectField<ProviderEditorValues>
          control={control}
          name="reasoningEffort"
          rules={{ required: true }}
          description={t("How much the model thinks before it answers.")}
          label={t("Reasoning")}
          placeholder={t("Reasoning")}
          options={efforts}
          triggerClassName={aicFieldTrigger}
        />
        {kind === "AnthropicMessages" && (
          <SelectField<ProviderEditorValues>
            control={control}
            name="thinkingStyle"
            rules={{ required: true }}
            description={t("How extended thinking is requested from the model.")}
            label={t("Thinking")}
            placeholder={t("Thinking")}
            options={thinking}
            triggerClassName={aicFieldTrigger}
          />
        )}
      </div>
      {embeds && (
        <div className={FIELD_PAIR}>
          <SelectField<ProviderEditorValues>
            control={control}
            name="embeddingDimensionsChoice"
            description={t("The length of each vector; empty uses the model's default.")}
            label={t("Vector size")}
            placeholder={t("Choose")}
            options={dimensions}
            isClearable
            triggerClassName={aicFieldTrigger}
          />
          <SelectField<ProviderEditorValues>
            control={control}
            name="embeddingInputStyle"
            rules={{ required: true }}
            description={t("How text is labelled when it's sent for embedding.")}
            label={t("Embedding input")}
            placeholder={t("Embedding input")}
            options={inputStyles}
            triggerClassName={aicFieldTrigger}
          />
        </div>
      )}
      <JsonEditorField<ProviderEditorValues>
        control={control}
        name="extraBodyText"
        label={t("Extra request fields")}
        description={t(
          "JSON merged under the fields Trenova sets, for options the endpoint takes.",
        )}
        placeholder='{"top_k": 40}'
      />
    </div>
  );
}
