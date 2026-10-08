import {
  fetchAIProvider,
  testAIProviderDraft,
  type AIProviderDraftTestResult,
  type AIProviderModelOption,
  type AIProviderRow,
  type AITaskRoute,
} from "@/lib/graphql/ai-provider";
import { queries } from "@/lib/queries";
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
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  FormProvider,
  useController,
  useForm,
  useFormContext,
  useWatch,
  type Control,
  type FieldPath,
} from "react-hook-form";
import { toast } from "sonner";
import type { EditFields } from "../edit/change-review";
import { EditSheet, type EditSection } from "../edit/edit-sheet";
import { Callout, Chips, F, Sel, SwRow, Txt } from "../edit/fields";
import { useEditFlow } from "../edit/use-edit-flow";
import { Ic } from "../kit/ic";
import { Seg } from "../kit/layout";
import { Mark } from "../kit/marks";
import { ConfirmDialog } from "../kit/modal";
import {
  draftTestInput,
  editorBlocker,
  editorValuesFromPreset,
  editorValuesFromProvider,
  providerEditorSchema,
  routingDraft,
  toSaveRequest,
  type KeyRule,
  type ProviderEditorValues,
} from "./provider-editor-model";
import { baseUrlProblem, formatTokens, keyPlaceholder, type TaskMeta } from "./provider-model";
import { kindSupportsEmbedding } from "./provider-form-schema";
import { TASK_FALLBACKS } from "./task-fallbacks";

const ROUTE_PREVIEW_DEBOUNCE_MS = 250;

/** What the editor is open on: a saved provider, or a new one from a preset. */
export type ProviderEditorTarget =
  | { kind: "edit"; provider: AIProviderRow }
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
}: ProviderEditorProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const saved = target.kind === "edit" ? target.provider : null;
  const create = saved === null;
  const [loaded] = useState(() =>
    target.kind === "edit"
      ? editorValuesFromProvider(target.provider)
      : editorValuesFromPreset(target.preset, providers),
  );
  const form = useForm<ProviderEditorValues>({
    resolver: zodResolver(providerEditorSchema),
    defaultValues: loaded,
    mode: "onChange",
  });
  const values = useWatch({ control: form.control }) as ProviderEditorValues;
  const [test, setTest] = useState<DraftTest | null>(null);
  const [removing, setRemoving] = useState(false);
  const [removeBusy, setRemoveBusy] = useState(false);

  const kindLabel = useCallback(
    (kind: string) => catalog.kinds.find((entry) => entry.kind === kind)?.label ?? kind,
    [catalog.kinds],
  );
  const needsKeyField = keyRequired(values.kind);
  const hasStoredKey = Boolean(saved?.hasApiKey);
  const requiresBaseUrl =
    catalog.kinds.find((entry) => entry.kind === values.kind)?.requiresBaseUrl ?? false;
  const testOk = test?.state === "done" && test.result.success;

  const connection = `${values.kind}|${values.baseUrl}|${values.model}|${values.apiKey}|${values.allowPrivateNetwork}`;
  useEffect(() => {
    setTest(null);
  }, [connection]);

  const blocker =
    requiresBaseUrl && values.baseUrl.trim() === ""
      ? t("Add the base URL")
      : editorBlocker(values, {
          create,
          keyRequired: needsKeyField,
          hasStoredKey,
        });

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
      const fresh = await fetchAIProvider(result.id);
      if (saved) {
        toast.success(t("Saved"));
      } else {
        toast.success(
          result.enabled ? t("{0} is on", result.name) : t("{0} added", result.name),
        );
      }
      return fresh ? editorValuesFromProvider(fresh) : next;
    },
    [create, queryClient, saved, t, testOk],
  );

  const flow = useEditFlow({
    form,
    onSave: save,
    onClose,
    create,
    invalid: blocker,
    resourceName: t("AI provider"),
    loadLatest: saved
      ? async () => {
          const latest = await fetchAIProvider(saved.id);
          if (!latest) throw new Error(t("This provider was removed"));
          return editorValuesFromProvider(latest);
        }
      : undefined,
  });

  const runTest = async () => {
    setTest({ state: "run" });
    try {
      const result = await testAIProviderDraft(draftTestInput(values, saved?.id ?? null));
      setTest({ state: "done", result });
    } catch (error: unknown) {
      setTest({
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
      });
    }
  };

  const models = useEndpointModels(form.control, saved?.id ?? null, create && !needsKeyField);

  const index = saved ? providers.findIndex((provider) => provider.id === saved.id) : -1;
  const taskLabel = useCallback(
    (task: string | number) => metas.find((meta) => meta.task === task)?.label ?? String(task),
    [metas],
  );
  const fields: EditFields = {
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
    description: { label: t("Description") },
    structuredOutputMode: { label: t("Structured output") },
    reasoningEffort: { label: t("Reasoning") },
    thinkingStyle: { label: t("Thinking") },
    maxTokens: { label: t("Most tokens per reply") },
    embeddingDimensionsChoice: { label: t("Vector size") },
    embeddingInputStyle: { label: t("Embedding input") },
    extraBodyText: { label: t("Extra request fields") },
  };

  const urlProblem = baseUrlProblem(values.baseUrl, values.allowPrivateNetwork);
  const trustNeeded = metas.some((meta) => meta.trust && values.tasks.includes(meta.task));
  const trustLabels = metas
    .filter((meta) => meta.trust && values.tasks.includes(meta.task))
    .map((meta) => meta.label);

  const sections: EditSection[] = [
    {
      id: "conn",
      label: t("Connection"),
      keys: ["name", "kind", "baseUrl"],
      warning:
        urlProblem === "private"
          ? t("This address is on your own network. Turn on Private network so Trenova can reach it.")
          : undefined,
      content: (
        <>
          <div className="f-grid">
            <F label={t("Name")} error={form.formState.errors.name?.message}>
              <FieldText name="name" label={t("Name")} autoFocus={create} />
            </F>
            <F label={t("Kind")}>
              <FieldSelect
                name="kind"
                label={t("Kind")}
                options={catalog.kinds.map((entry) => [entry.kind, entry.label] as const)}
              />
            </F>
          </div>
          <F
            label={t("Base URL")}
            error={
              urlProblem === "private"
                ? t(
                    "This address is on your own network. Turn on Private network so Trenova can reach it.",
                  )
                : urlProblem === "scheme"
                  ? t("Start with http:// or https://")
                  : undefined
            }
            hint={
              values.baseUrl.trim() === ""
                ? requiresBaseUrl
                  ? t("This kind of endpoint has no default; give its address.")
                  : t("Leave empty to use {0}'s default endpoint.", kindLabel(values.kind))
                : undefined
            }
            aside={
              urlProblem === "private" ? (
                <button
                  type="button"
                  className="lnk"
                  onClick={() =>
                    form.setValue("allowPrivateNetwork", true, {
                      shouldDirty: true,
                      shouldValidate: true,
                    })
                  }
                >
                  {t("Turn on Private network")}
                </button>
              ) : undefined
            }
          >
            <FieldText
              name="baseUrl"
              label={t("Base URL")}
              mono
              placeholder="https://api.example.com/v1"
            />
          </F>
        </>
      ),
    },
    {
      id: "model",
      label: t("Model"),
      keys: ["model"],
      actions: (
        <button
          type="button"
          className="btn sm"
          disabled={models.fetching}
          onClick={models.fetch}
        >
          {models.fetching ? (
            <>
              <i className="spn" />
              {t("Asking {0}…", kindLabel(values.kind))}
            </>
          ) : (
            <>
              <Ic n="refresh" s={12} />
              {models.list ? t("Refresh") : t("Fetch models")}
            </>
          )}
        </button>
      ),
      content: <ModelSection models={models} local={!needsKeyField} />,
    },
    ...(needsKeyField
      ? [
          {
            id: "key",
            label: t("API key"),
            keys: ["apiKey", "keepPreviousKey"],
            content: (
              <>
                {saved?.apiKey && values.apiKey === "" && <StoredKey info={saved.apiKey} />}
                <F
                  label={hasStoredKey ? t("Replace key") : t("Key")}
                  hint={t("Stored encrypted. Nobody, including you, can read it back.")}
                >
                  <FieldText
                    name="apiKey"
                    type="password"
                    label={hasStoredKey ? t("Replace key") : t("Key")}
                    mono
                    placeholder={keyPlaceholder(values.kind, values.baseUrl) ?? t("API key")}
                  />
                </F>
                {hasStoredKey && values.apiKey !== "" && (
                  <FieldSwitch
                    name="keepPreviousKey"
                    label={t("Keep the old key working for 24 hours")}
                    note={t("So nothing fails while other systems switch over.")}
                  />
                )}
              </>
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
        <>
          <FieldChips metas={metas} />
          {form.formState.errors.tasks?.message && (
            <p className="f-h t-d">{form.formState.errors.tasks.message}</p>
          )}
          <RouteImpact providerId={saved?.id ?? null} metas={metas} providers={providers} />
        </>
      ),
    },
    {
      id: "access",
      label: t("Access"),
      keys: ["trusted", "allowPrivateNetwork"],
      content: (
        <>
          <FieldSwitch
            name="trusted"
            label={t("Trusted")}
            note={t("May take tasks that read sensitive records, like billing diagnosis.")}
          />
          <FieldSwitch
            name="allowPrivateNetwork"
            label={t("Private network")}
            note={t("May reach a server on your own network.")}
          />
          {trustNeeded && !values.trusted && (
            <Callout
              tone="w"
              action={
                <button
                  type="button"
                  className="btn sm"
                  onClick={() => form.setValue("trusted", true, { shouldDirty: true })}
                >
                  {t("Trust it")}
                </button>
              }
            >
              {t(
                "{0} needs a trusted provider. It's assigned here but will be skipped.",
                trustLabels.join(", "),
              )}
            </Callout>
          )}
        </>
      ),
    },
    {
      id: "limits",
      label: t("Limits and price"),
      keys: [
        "timeoutSeconds",
        "maxConcurrent",
        "monthlyCapUsd",
        "onCap",
        "inputCostPerMillion",
        "outputCostPerMillion",
      ],
      content: (
        <>
          <div className="f-grid">
            <F label={t("Timeout")} error={form.formState.errors.timeoutSeconds?.message}>
              <FieldText
                name="timeoutSeconds"
                type="number"
                label={t("Timeout")}
                suffix={t("seconds")}
                mono
              />
            </F>
            <F label={t("Concurrent calls")} error={form.formState.errors.maxConcurrent?.message}>
              <FieldText
                name="maxConcurrent"
                type="number"
                label={t("Concurrent calls")}
                suffix={t("at once")}
                mono
              />
            </F>
          </div>
          <div className="f-grid">
            <F
              label={t("Monthly spend cap")}
              error={form.formState.errors.monthlyCapUsd?.message}
              hint={
                values.monthlyCapUsd.trim() === ""
                  ? t("No cap")
                  : saved
                    ? t("${0} spent this month", Number(saved.monthSpendUsd).toFixed(2))
                    : t("Counted from the first call")
              }
            >
              <FieldText
                name="monthlyCapUsd"
                type="number"
                label={t("Monthly spend cap")}
                prefix="$"
                mono
                placeholder={t("None")}
              />
            </F>
            <F label={t("At the cap")}>
              <FieldSeg
                name="onCap"
                label={t("At the cap")}
                options={[
                  ["Next", t("Hand to next")],
                  ["Stop", t("Stop")],
                ]}
              />
            </F>
          </div>
          <div className="f-grid">
            <F
              label={t("Input price")}
              hint={t("Per million tokens")}
              error={form.formState.errors.inputCostPerMillion?.message}
            >
              <FieldText
                name="inputCostPerMillion"
                label={t("Input price")}
                prefix="$"
                mono
                placeholder="—"
              />
            </F>
            <F
              label={t("Output price")}
              hint={t("Per million tokens")}
              error={form.formState.errors.outputCostPerMillion?.message}
            >
              <FieldText
                name="outputCostPerMillion"
                label={t("Output price")}
                prefix="$"
                mono
                placeholder="—"
              />
            </F>
          </div>
        </>
      ),
    },
    {
      id: "advanced",
      label: t("Advanced"),
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
      note: t("How requests are shaped for this endpoint. The defaults suit most models."),
      content: <AdvancedFields catalog={catalog} />,
    },
    ...(saved && canDelete
      ? [
          {
            id: "danger",
            label: t("Remove"),
            content: (
              <div className="dz">
                <div>
                  <b>{t("Remove {0}", saved.name)}</b>
                  <span>
                    {saved.tasks.length === 0
                      ? t("It handles no tasks.")
                      : saved.tasks.length === 1
                        ? t("Its 1 task falls to the next provider in line.")
                        : t("Its {0} tasks fall to the next provider in line.", saved.tasks.length)}
                  </span>
                </div>
                <button type="button" className="btn sm dng" onClick={() => setRemoving(true)}>
                  {t("Remove provider")}
                </button>
              </div>
            ),
          } satisfies EditSection,
        ]
      : []),
  ];

  const mark = saved ?? { name: values.name || (target.kind === "create" ? target.preset.label : "") };
  const failed = test?.state === "done" && !test.result.success ? test.result : null;

  return (
    <FormProvider {...form}>
      <EditSheet
        open
        form={form}
        flow={flow}
        fields={fields}
        sections={sections}
        icon={<Mark provider={mark} s={36} />}
        title={saved ? t("Edit {0}", saved.name) : t("Add {0}", values.name.trim() || t("provider"))}
        subtitle={
          saved
            ? t("Priority {0} of {1}", index + 1, providers.length)
            : needsKeyField
              ? t("Hosted · needs an API key")
              : t("On your network · no key")
        }
        banner={
          failed ? (
            <div className="es-bn d" role="alert">
              <Ic n="alert" s={13} />
              <div>
                <b>{failed.message}</b>
                {failed.detail && <span className="mono">{failed.detail}</span>}
                {failed.hint && <span>{failed.hint}</span>}
              </div>
            </div>
          ) : undefined
        }
        footerLeading={
          <>
            <button
              type="button"
              className="btn sm"
              disabled={test?.state === "run" || values.model.trim() === ""}
              onClick={() => void runTest()}
            >
              <Ic n="plug" s={12} />
              {t("Test draft")}
            </button>
            {test && (
              <span
                className={cn(
                  "es-tr",
                  test.state === "run" ? "run" : test.result.success ? "ok" : "fail",
                )}
                title={test.state === "done" ? test.result.detail : undefined}
              >
                {test.state === "run" ? (
                  <>
                    <i className="spn" />
                    {t("Testing…")}
                  </>
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
          </>
        }
        saveLabel={saved ? t("Save changes") : testOk ? t("Add and turn on") : t("Add provider")}
      />
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
                flow.close();
              })
              .catch(() => undefined)
              .finally(() => setRemoveBusy(false));
          }}
        />
      )}
    </FormProvider>
  );
}

function useEditorField<K extends FieldPath<ProviderEditorValues>>(name: K) {
  const { control } = useFormContext<ProviderEditorValues>();
  const { field } = useController({ control, name });
  return [field.value, field.onChange] as const;
}

type TextFieldName =
  | "name"
  | "baseUrl"
  | "model"
  | "apiKey"
  | "timeoutSeconds"
  | "maxConcurrent"
  | "monthlyCapUsd"
  | "inputCostPerMillion"
  | "outputCostPerMillion"
  | "description"
  | "maxTokens";

function FieldText({
  name,
  label,
  ...rest
}: {
  name: TextFieldName;
  label: string;
  mono?: boolean;
  placeholder?: string;
  prefix?: string;
  suffix?: string;
  type?: "text" | "password" | "number";
  autoFocus?: boolean;
}) {
  const [value, onChange] = useEditorField(name);
  return <Txt value={value} onChange={onChange} label={label} {...rest} />;
}

function FieldSelect<K extends "kind" | "structuredOutputMode" | "reasoningEffort" | "thinkingStyle" | "embeddingInputStyle" | "embeddingDimensionsChoice">({
  name,
  label,
  options,
}: {
  name: K;
  label: string;
  options: readonly (readonly [ProviderEditorValues[K], string])[];
}) {
  const [value, onChange] = useEditorField(name);
  return (
    <Sel
      value={value as ProviderEditorValues[K]}
      onChange={(next) => onChange(next)}
      options={options}
      label={label}
    />
  );
}

function FieldSwitch({
  name,
  label,
  note,
}: {
  name: "keepPreviousKey" | "trusted" | "allowPrivateNetwork";
  label: string;
  note: string;
}) {
  const [value, onChange] = useEditorField(name);
  return <SwRow label={label} note={note} on={value} onChange={onChange} />;
}

function FieldSeg({
  name,
  label,
  options,
}: {
  name: "onCap";
  label: string;
  options: readonly (readonly [ProviderEditorValues["onCap"], string])[];
}) {
  const [value, onChange] = useEditorField(name);
  return <Seg v={value} opts={options} label={label} onChange={onChange} />;
}

function FieldChips({ metas }: { metas: readonly TaskMeta[] }) {
  const t = useT();
  const [value, onChange] = useEditorField("tasks");
  return (
    <Chips<AITask>
      value={value}
      onChange={onChange}
      label={t("What it handles")}
      options={metas.map(
        (meta) =>
          [meta.task, meta.label, meta.trust ? <Ic key="t" n="shield" s={10} /> : null] as const,
      )}
    />
  );
}

/** The current key, described: its ends, who added it, when it was last used. */
function StoredKey({ info }: { info: NonNullable<AIProviderRow["apiKey"]> }) {
  const t = useT();
  const now = Math.floor(Date.now() / 1000);
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
    <div className="kst">
      <Ic n="key" s={13} />
      <span className="mono">{`${info.prefix}••••••••${info.lastFour}`}</span>
      <span className="kst-m">{[added, used, previous].filter(Boolean).join(" · ")}</span>
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

  const metaOf = (option: AIProviderModelOption) => {
    const parts: string[] = [];
    if (option.embedding) parts.push(t("Embedding"));
    if (option.contextWindow) {
      parts.push(t("{0} context", formatTokens(option.contextWindow).toLowerCase()));
    }
    if (option.sizeBytes) parts.push(formatBytes(option.sizeBytes));
    if (option.loaded) parts.push(t("loaded"));
    return parts.join(" · ");
  };
  const priceOf = (option: AIProviderModelOption) => {
    if (option.inputCostPerMillion && option.outputCostPerMillion) {
      return `$${Number(option.inputCostPerMillion)} / $${Number(option.outputCostPerMillion)}`;
    }
    if (option.inputCostPerMillion) return `$${Number(option.inputCostPerMillion)}`;
    return local ? t("Local") : "—";
  };

  if (list && list.length > 0) {
    return (
      <div className="mdl" role="radiogroup" aria-label={t("Model")}>
        {list.map((option) => (
          <button
            key={option.id}
            type="button"
            role="radio"
            aria-checked={model === option.id}
            className={cn("mdl-r", model === option.id && "on")}
            onClick={() => setModel(option.id)}
          >
            <span className="mdl-o" />
            <span className="mdl-t">
              <b className="mono">{option.id}</b>
              <em>{metaOf(option)}</em>
            </span>
            <span className="mdl-p mono">{priceOf(option)}</span>
          </button>
        ))}
      </div>
    );
  }

  return (
    <F
      label={t("Model ID")}
      error={
        models.error ??
        (list && list.length === 0
          ? t("The endpoint lists no models. Type the ID it serves.")
          : undefined)
      }
      hint={t("Or fetch the list this provider actually serves.")}
    >
      <Txt value={model} onChange={setModel} mono label={t("Model ID")} />
    </F>
  );
}

function formatBytes(bytes: number): string {
  if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
  if (bytes >= 1e6) return `${Math.round(bytes / 1e6)} MB`;
  return `${Math.round(bytes / 1e3)} KB`;
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
      <p className="imp-n">
        <Ic n="check" s={12} w={2.2} />
        {t("No task changes where it goes.")}
      </p>
    );
  }

  const choice = (route: AITaskRoute["before"], strong: boolean) => {
    if (!route) return null;
    const provider = route.providerId ? byId.get(String(route.providerId)) : undefined;
    const name = route.draft ? values.name.trim() || route.name : (provider?.name ?? route.name);
    return (
      <>
        <Mark provider={{ name }} s={14} />
        {strong ? <b>{name}</b> : name}
      </>
    );
  };

  return (
    <div className="imp">
      <div className="imp-h">{t("When you save")}</div>
      {rows.map((route) => (
        <div
          key={route.task}
          className={cn("imp-r", !route.after && "lost", !route.before && "won")}
        >
          <span>{labels.get(route.task) ?? route.task}</span>
          <span className="imp-v">
            {route.before ? choice(route.before, false) : <em>{t("Nowhere")}</em>}
            <Ic n="arrowR" s={11} />
            {route.after ? (
              choice(route.after, true)
            ) : (
              <em className="t-w">
                {t("Nowhere · {0}", TASK_FALLBACKS[route.task].toLowerCase())}
              </em>
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
  const { control, formState } = useFormContext<ProviderEditorValues>();
  const [kind, tasks] = useWatch({ control, name: ["kind", "tasks"] });
  const [extraBody, setExtraBody] = useEditorField("extraBodyText");
  const embeds = tasks.includes("Embedding") && kindSupportsEmbedding(kind);

  const outputModes: [StructuredOutputMode, string][] = [
    ["JSONSchema", t("JSON schema")],
    ["JSONMode", t("JSON mode")],
    ["Prompted", t("Prompted")],
  ];
  const efforts: [ReasoningEffort, string][] = [
    ["Off", t("Off")],
    ["None", t("None")],
    ["Minimal", t("Minimal")],
    ["Low", t("Low")],
    ["Medium", t("Medium")],
    ["High", t("High")],
  ];
  const thinking: [ThinkingStyle, string][] = [
    ["Auto", t("Read from the model")],
    ["Effort", t("By effort")],
    ["Budget", t("By token budget")],
  ];
  const inputStyles: [EmbeddingInputStyle, string][] = catalog.embeddingInputStyles.map(
    (entry) => [entry.style, entry.label],
  );
  const dimensions: [string, string][] = catalog.embeddingDimensions.map((size) => [
    String(size),
    String(size),
  ]);

  return (
    <>
      <F label={t("Description")}>
        <FieldText name="description" label={t("Description")} />
      </F>
      <div className="f-grid">
        <F label={t("Structured output")}>
          <FieldSelect name="structuredOutputMode" label={t("Structured output")} options={outputModes} />
        </F>
        <F label={t("Most tokens per reply")} error={formState.errors.maxTokens?.message}>
          <FieldText name="maxTokens" type="number" label={t("Most tokens per reply")} mono />
        </F>
      </div>
      <div className="f-grid">
        <F label={t("Reasoning")}>
          <FieldSelect name="reasoningEffort" label={t("Reasoning")} options={efforts} />
        </F>
        {kind === "AnthropicMessages" && (
          <F label={t("Thinking")}>
            <FieldSelect name="thinkingStyle" label={t("Thinking")} options={thinking} />
          </F>
        )}
      </div>
      {embeds && (
        <div className="f-grid">
          <F
            label={t("Vector size")}
            error={formState.errors.embeddingDimensionsChoice?.message}
          >
            <FieldSelect
              name="embeddingDimensionsChoice"
              label={t("Vector size")}
              options={[["", t("Choose")], ...dimensions]}
            />
          </F>
          <F label={t("Embedding input")}>
            <FieldSelect name="embeddingInputStyle" label={t("Embedding input")} options={inputStyles} />
          </F>
        </div>
      )}
      <F
        label={t("Extra request fields")}
        hint={t("JSON merged under the fields Trenova sets, for options the endpoint takes.")}
        error={formState.errors.extraBodyText?.message}
      >
        <div className="ara">
          <textarea
            className="mono"
            rows={4}
            value={extraBody}
            aria-label={t("Extra request fields")}
            placeholder='{"top_k": 40}'
            onChange={(event) => setExtraBody(event.target.value)}
          />
        </div>
      </F>
    </>
  );
}
