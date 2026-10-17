import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { ExternalLink } from "@/components/link";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { agentExtension } from "@/lib/queries/agent-extension";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import {
  AgentExtensionCapability,
  type AgentExtensionAvailability,
  type AgentExtensionCatalogItem,
  type AgentExtensionConfigResponse,
} from "@/types/agent-extension";
import { lazyComponent } from "@/lib/lazy-component";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { AnalysisRunsSkeleton } from "./analysis-runs-skeleton";
import { PortalRunsSkeleton } from "./portal-runs-skeleton";
import { PortalSitesSkeleton } from "./portal-sites-skeleton";
import type { ConfigFieldSpec } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toAgentPanelRow, toSaveRequest } from "../agents/agent-form-schema";
import { aicFieldTrigger } from "../edit/field-trigger";
import { Ic } from "../kit/ic";
import { Tile } from "../kit/marks";
import {
  capabilityHolder,
  extensionState,
  holdsExtension,
  withExtensionTools,
} from "./extension-roster";
import { Button } from "@trenova/shared/components/ui/button";

const portalSites = lazyComponent(
  () => import("./portal-sites"),
  (module) => module.PortalSites,
);
const portalRuns = lazyComponent(
  () => import("./portal-runs"),
  (module) => module.PortalRuns,
);
const analysisRuns = lazyComponent(
  () => import("./analysis-runs"),
  (module) => module.AnalysisRuns,
);

/** What each choice a setting offers is called; one a vendor adds unnamed shows as sent. */
const SETTING_OPTION_LABELS: Record<string, string> = defineLabels({
  auto: "Auto",
  fast: "Fast",
  deep: "Deep",
  basic: "Basic",
  advanced: "Advanced",
  "browser-use-2.0": "Browser Use 2.0",
  "gpt-5.6-luna": "GPT-5.6 Luna",
  "claude-sonnet-5": "Claude Sonnet 5",
  "gemini-3-flash-preview": "Gemini 3 Flash",
  us: "United States",
  ca: "Canada",
  mx: "Mexico",
  none: "No proxy",
});

/** The daily limits offered as steps; a saved limit outside them is offered too. */
const DAILY_LIMITS = [100, 500, 1000, 5000] as const;

/** Fields drawn by their own controls rather than as a plain box. */
const OWN_CONTROLS = new Set(["searchType", "dailyRequestLimit"]);

type ExtensionDetailProps = {
  item: AgentExtensionCatalogItem;
  control: ExtensionControl;
  /** Desk agents the extension can be given to. */
  agents: readonly AgentDefinitionRow[];
  canUpdate: boolean;
  canUpdateAgents: boolean;
};

export type ConfigPatch = {
  enabled?: boolean;
  availability?: AgentExtensionAvailability;
  configuration?: Record<string, string>;
};

type TestState = { state: "run" } | { state: "ok" | "fail"; message: string; latencyMs: number };

export type ExtensionControl = {
  config: AgentExtensionConfigResponse | undefined;
  saving: boolean;
  save: (patch: ConfigPatch, onSaved?: () => void) => void;
};

/** An extension's saved settings and the one way to change them, shared by its sheet. */
export function useExtensionControl(item: AgentExtensionCatalogItem | null): ExtensionControl {
  const t = useT();
  const queryClient = useQueryClient();
  const type = item?.type ?? "";
  const configQuery = useQuery({ ...agentExtension.config(type), enabled: type !== "" });
  const config = configQuery.data;

  const mutation = useApiMutation({
    mutationFn: ({ patch }: { patch: ConfigPatch; onSaved?: () => void }) => {
      if (!config) throw new Error(t("The settings have not loaded yet"));
      return apiService.agentExtensionService.updateConfig(type, {
        enabled: patch.enabled ?? config.enabled,
        availability: patch.availability ?? config.availability,
        configuration: { ...currentConfiguration(config), ...patch.configuration },
        version: config.version,
      });
    },
    onSuccess: async (_saved, { onSaved }) => {
      onSaved?.();
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: agentExtension._def }),
        queryClient.invalidateQueries({ queryKey: assistant._def }),
      ]);
    },
    resourceName: t("Extension"),
  });

  return {
    config,
    saving: mutation.isPending,
    save: (patch, onSaved) => mutation.mutate({ patch, onSaved }),
  };
}

/**
 * Everything decided about one extension, each change saved as it is made: what it
 * can do and what leaves Trenova, its key and limits, and which agents get its tools.
 */
export function ExtensionDetail({
  item,
  control,
  agents,
  canUpdate,
  canUpdateAgents,
}: ExtensionDetailProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const { config } = control;
  const keyForm = useForm<{ key: string }>({ defaultValues: { key: "" } });
  const key = useWatch({ control: keyForm.control, name: "key" });
  const [replacing, setReplacing] = useState(false);
  const [test, setTest] = useState<TestState | null>(null);
  const keySaved = () => {
    keyForm.reset({ key: "" });
    setReplacing(false);
  };

  const keyField = config?.spec.find((field) => field.sensitive) ?? null;
  const stored = keyField
    ? (config?.fields.find((field) => field.key === keyField.key)?.hasValue ?? false)
    : false;
  const on = extensionState(item) === "on";
  const heldBy = capabilityHolder(item);
  const canTurnOn = heldBy === "" && (!keyField || stored || key.trim() !== "");
  const availability = config?.availability ?? item.availability;

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: assistant._def });
  };

  const give = useApiMutation({
    mutationFn: ({ agent, holds }: { agent: AgentDefinitionRow; holds: boolean }) =>
      apiService.agentDefinitionService.update(
        agent.id,
        toSaveRequest({
          ...toAgentPanelRow(agent),
          toolNames: withExtensionTools(agent.toolNames, item.tools, !holds),
        }),
      ),
    onSuccess: async () => {
      await refresh();
    },
    resourceName: t("Agent"),
  });

  const runTest = async () => {
    setTest({ state: "run" });
    try {
      const result = await apiService.agentExtensionService.test(item.type);
      setTest({
        state: result.success ? "ok" : "fail",
        message: result.message,
        latencyMs: result.latencyMs,
      });
    } catch (error: unknown) {
      setTest({
        state: "fail",
        message: error instanceof Error ? error.message : t("The test could not run"),
        latencyMs: 0,
      });
    }
  };

  const turnOn = () => {
    const configuration = keyField && key.trim() ? { [keyField.key]: key.trim() } : undefined;
    control.save({ enabled: true, configuration }, keySaved);
  };

  const depthField = config?.spec.find((field) => field.key === "searchType") ?? null;
  const limitField = config?.spec.find((field) => field.key === "dailyRequestLimit") ?? null;
  const limit = Number(valueOf(config, "dailyRequestLimit") || item.dailyRequestLimit);
  const limits = [...new Set([...DAILY_LIMITS, limit])]
    .filter((value) => value > 0)
    .sort((a, b) => a - b);
  const others = (config?.spec ?? []).filter(
    (field) => !field.sensitive && !OWN_CONTROLS.has(field.key),
  );
  const busy = control.saving || !config;

  return (
    <div className="pd">
      <div className="pd-g">
        <div className="pd-s">
          <h4>{t("What it can do")}</h4>
          <ul className="caps">
            {item.capabilities.map((capability) => (
              <li key={capability}>
                <Ic n="check" s={12} w={2.2} />
                {capability}
              </li>
            ))}
          </ul>
          <div className="pv-tk mt8">
            {item.tools.map((tool) => (
              <span key={tool.name} className="tk mono" title={tool.description}>
                <Ic n="tool" s={10} />
                {tool.name}
              </span>
            ))}
          </div>
          <p className="ex-n">
            <Ic n="shield" s={12} />
            {item.dataNotice}
          </p>
          {heldBy !== "" && (
            <Alert size="sm" variant="info" className="mt8">
              <AlertDescription>
                {t(
                  "{0} is already on and gives agents the same tools. Turn it off before turning this one on.",
                  heldBy,
                )}
              </AlertDescription>
            </Alert>
          )}
        </div>
        <div className="pd-s">
          <h4>{t("Connection")}</h4>
          {!keyField ? (
            <div className="kst">
              <Ic n="check" s={12} w={2.2} />
              <span>{t("No key needed — it's a public source")}</span>
            </div>
          ) : stored && !replacing ? (
            <div className="kst">
              <Ic n="key" s={12} />
              <span>{t("{0} key stored", item.vendor)}</span>
              {canUpdate && (
                <button type="button" className="lnk" onClick={() => setReplacing(true)}>
                  {t("Replace")}
                </button>
              )}
            </div>
          ) : (
            <form
              className="flex w-full items-start gap-2"
              onSubmit={(event) => {
                event.preventDefault();
                if (!key.trim() || busy) return;
                if (on || stored) {
                  control.save({ configuration: { [keyField.key]: key.trim() } }, keySaved);
                } else {
                  turnOn();
                }
              }}
            >
              <InputField
                control={keyForm.control}
                name="key"
                rules={{ required: true }}
                type="password"
                autoFocus
                autoComplete="new-password"
                aria-label={keyField.label}
                placeholder={keyField.placeholder || t("API key")}
                disabled={!canUpdate || busy}
                leftElement={<Ic n="key" s={13} />}
                className="min-w-0 flex-1"
                inputClassProps={aicFieldTrigger}
              />
              {(on || stored) && (
                <Button type="submit" variant="default" size="sm" disabled={!key.trim() || busy}>
                  {t("Save key")}
                </Button>
              )}
            </form>
          )}
          {depthField && (
            <>
              <h4 className="mt">{t("Search depth")}</h4>
              <SegmentedControl<string>
                aria-label={t("Search depth")}
                value={valueOf(config, "searchType") || depthField.default || "auto"}
                items={(depthField.options ?? []).map((option) => ({
                  value: option,
                  label: SETTING_OPTION_LABELS[option] ?? option,
                  disabled: !canUpdate,
                }))}
                onValueChange={(value) =>
                  canUpdate && control.save({ configuration: { searchType: value } })
                }
              />
            </>
          )}
          {limitField && (
            <>
              <h4 className="mt">{t("Daily request limit")}</h4>
              <SegmentedControl<string>
                aria-label={t("Daily request limit")}
                value={String(limit)}
                items={limits.map((value) => ({
                  value: String(value),
                  label: value.toLocaleString(),
                  disabled: !canUpdate,
                }))}
                onValueChange={(value) =>
                  canUpdate && control.save({ configuration: { dailyRequestLimit: value } })
                }
              />
            </>
          )}
          {config &&
            others.map((field) => {
              const save = (value: string) =>
                control.save({ configuration: { [field.key]: value } });
              const value = valueOf(config, field.key) || field.default || "";
              if (field.type === "select" && (field.options?.length ?? 0) > 0) {
                return (
                  <ChoiceSetting
                    key={`${field.key}:${config.version}`}
                    field={field}
                    value={value}
                    disabled={!canUpdate || busy}
                    onSave={save}
                  />
                );
              }
              if (field.type === "boolean") {
                return (
                  <FlagSetting
                    key={`${field.key}:${config.version}`}
                    field={field}
                    value={value}
                    disabled={!canUpdate || busy}
                    onSave={save}
                  />
                );
              }
              return (
                <OtherField
                  key={`${field.key}:${config.version}`}
                  field={field}
                  value={valueOf(config, field.key)}
                  disabled={!canUpdate || busy}
                  onSave={save}
                />
              );
            })}
        </div>
        <div className="pd-s">
          <h4>{t("Available to")}</h4>
          <SegmentedControl<typeof availability>
            aria-label={t("Available to")}
            value={availability}
            items={[
              { value: "SelectedAgents", label: t("Agents you choose"), disabled: !canUpdate },
              { value: "AllAgents", label: t("Every agent"), disabled: !canUpdate },
            ]}
            onValueChange={(value) =>
              canUpdate && value !== availability && control.save({ availability: value })
            }
          />
          {availability === "SelectedAgents" ? (
            <div className="ex-ag">
              {agents.map((agent) => {
                const holds = holdsExtension(agent.toolNames, item.tools);
                return (
                  <button
                    key={agent.id}
                    type="button"
                    className={cn("ex-a", holds && "on")}
                    aria-pressed={holds}
                    disabled={!canUpdateAgents || give.isPending}
                    onClick={() => give.mutate({ agent, holds })}
                  >
                    <Tile agent={agent} s={18} />
                    <span>{agent.name}</span>
                    {holds && <Ic n="check" s={11} w={2.4} />}
                  </button>
                );
              })}
            </div>
          ) : (
            <p className="ad-h">
              {t(
                "Every agent, including the assistant, gets {0}.",
                item.tools.map((tool) => tool.name).join(", "),
              )}
            </p>
          )}
        </div>
      </div>
      {item.capability === AgentExtensionCapability.PortalAutomation && (
        <div className="flex flex-col gap-6 px-5 pb-5">
          <LazyComponent fallback={<PortalSitesSkeleton />}>
            <portalSites.Component canUpdate={canUpdate} />
          </LazyComponent>
          <LazyComponent fallback={<PortalRunsSkeleton />}>
            <portalRuns.Component />
          </LazyComponent>
        </div>
      )}
      {item.capability === AgentExtensionCapability.DataAnalysis && (
        <div className="flex flex-col gap-6 px-5 pb-5">
          <LazyComponent fallback={<AnalysisRunsSkeleton />}>
            <analysisRuns.Component />
          </LazyComponent>
        </div>
      )}
      <div className="ad-bar sh-f">
        {canUpdate &&
          (on ? (
            <button
              type="button"
              className="xa d"
              disabled={busy}
              onClick={() => control.save({ enabled: false })}
            >
              <Ic n="ban" s={13} />
              {t("Turn off")}
            </button>
          ) : (
            <Button
              type="button"
              variant="default"
              size="sm"
              disabled={!canTurnOn || busy}
              isLoading={control.saving}
              loadingText={t("Turn on")}
              onClick={turnOn}
            >
              {t("Turn on")}
            </Button>
          ))}
        {keyField && item.supportsTestConnect && canUpdate && (
          <button
            type="button"
            className="xa"
            disabled={!stored || test?.state === "run"}
            onClick={() => void runTest()}
          >
            <Ic n="plug" s={13} />
            {test?.state === "run" ? t("Testing…") : t("Test connection")}
          </button>
        )}
        {test && test.state !== "run" && (
          <span className={cn("tr", test.state === "ok" ? "t-k" : "t-d")} title={test.message}>
            <Ic n={test.state === "ok" ? "check" : "x"} s={11} w={2.4} />
            {test.state === "ok" ? t("Works · {0} ms", test.latencyMs) : test.message}
          </span>
        )}
        <span className="sp" />
        {item.docsUrl && (
          <ExternalLink href={item.docsUrl} className="xa">
            {t("{0} docs", item.vendor)}
          </ExternalLink>
        )}
      </div>
    </div>
  );
}

function valueOf(config: AgentExtensionConfigResponse | undefined, key: string): string {
  return config?.fields.find((field) => field.key === key)?.value ?? "";
}

/**
 * The configuration as saved, for the fields the client can see. A key is left blank,
 * which the server reads as "keep the stored one".
 */
function currentConfiguration(config: AgentExtensionConfigResponse): Record<string, string> {
  return Object.fromEntries(
    config.spec.map((field) => [
      field.key,
      field.sensitive
        ? ""
        : (config.fields.find((value) => value.key === field.key)?.value ?? field.default ?? ""),
    ]),
  );
}

/** A setting the extension takes beyond its key, depth and limit, saved when it is left. */
type SettingProps = {
  field: ConfigFieldSpec;
  value: string;
  disabled: boolean;
  onSave: (value: string) => void;
};

/** A setting with a few named choices, saved as soon as one is picked. */
function ChoiceSetting({ field, value, disabled, onSave }: SettingProps) {
  const t = useT();

  return (
    <div className="mt-3 flex flex-col gap-1.5">
      <h4 className="mt">{t(field.label)}</h4>
      <SegmentedControl<string>
        aria-label={t(field.label)}
        value={value}
        items={(field.options ?? []).map((option) => ({
          value: option,
          label: SETTING_OPTION_LABELS[option] ?? option,
          disabled,
        }))}
        onValueChange={(next) => !disabled && next !== value && onSave(next)}
      />
      {field.helpText && <p className="text-muted-foreground text-xs">{t(field.helpText)}</p>}
    </div>
  );
}

/** A setting that is on or off, saved as soon as it is flipped. */
function FlagSetting({ field, value, disabled, onSave }: SettingProps) {
  const t = useT();
  const form = useForm<{ on: boolean }>({ values: { on: value === "true" } });

  useEffect(() => {
    const subscription = form.watch((next, { type }) => {
      if (type === "change") onSave(next.on ? "true" : "false");
    });
    return () => subscription.unsubscribe();
  }, [form, onSave]);

  return (
    <SwitchField
      control={form.control}
      name="on"
      label={t(field.label)}
      description={field.helpText ? t(field.helpText) : undefined}
      disabled={disabled}
      className="mt-3"
    />
  );
}

function OtherField({
  field,
  value,
  disabled,
  onSave,
}: {
  field: ConfigFieldSpec;
  value: string;
  disabled: boolean;
  onSave: (value: string) => void;
}) {
  const t = useT();
  const form = useForm<{ text: string }>({
    defaultValues: { text: value || field.default || "" },
  });
  const commit = () => {
    const next = form.getValues("text").trim();
    if (field.type === "number" && next !== "" && !/^\d+$/.test(next)) {
      form.setValue("text", value);
      return;
    }
    if (next !== (value || field.default || "")) onSave(next);
  };
  return (
    <InputField
      control={form.control}
      name="text"
      label={t(field.label)}
      description={field.helpText ? t(field.helpText) : undefined}
      inputMode={field.type === "number" ? "numeric" : undefined}
      placeholder={field.placeholder}
      disabled={disabled}
      className="mt-3"
      inputClassProps={aicFieldTrigger}
      onBlur={commit}
      onKeyDown={(event) => event.key === "Enter" && event.currentTarget.blur()}
    />
  );
}
