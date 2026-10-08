import { ExternalLink } from "@/components/link";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  AgentExtensionAvailability,
  AgentExtensionCatalogItem,
  AgentExtensionConfigResponse,
} from "@/types/agent-extension";
import type { ConfigFieldSpec } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { toast } from "sonner";
import { toAgentPanelRow, toSaveRequest } from "../agents/agent-form-schema";
import { Ic } from "../kit/ic";
import { Seg } from "../kit/layout";
import { Tile } from "../kit/marks";
import { extensionState, holdsExtension, withExtensionTools } from "./extension-roster";

const SEARCH_DEPTH_LABELS: Record<string, string> = defineLabels({
  auto: "Auto",
  fast: "Fast",
  deep: "Deep",
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
  save: (patch: ConfigPatch, message?: string, onSaved?: () => void) => void;
};

/** An extension's saved settings and the one way to change them, shared by its sheet. */
export function useExtensionControl(item: AgentExtensionCatalogItem | null): ExtensionControl {
  const t = useT();
  const queryClient = useQueryClient();
  const type = item?.type ?? "";
  const configQuery = useQuery({ ...queries.agentExtension.config(type), enabled: type !== "" });
  const config = configQuery.data;

  const mutation = useApiMutation({
    mutationFn: ({ patch }: { patch: ConfigPatch; message?: string; onSaved?: () => void }) => {
      if (!config) throw new Error(t("The settings have not loaded yet"));
      return apiService.agentExtensionService.updateConfig(type, {
        enabled: patch.enabled ?? config.enabled,
        availability: patch.availability ?? config.availability,
        configuration: { ...currentConfiguration(config), ...patch.configuration },
        version: config.version,
      });
    },
    onSuccess: async (_saved, { message, onSaved }) => {
      if (message) toast.success(message);
      onSaved?.();
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.agentExtension._def }),
        queryClient.invalidateQueries({ queryKey: queries.assistant._def }),
      ]);
    },
    resourceName: t("Extension"),
  });

  return {
    config,
    saving: mutation.isPending,
    save: (patch, message, onSaved) => mutation.mutate({ patch, message, onSaved }),
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
  const [key, setKey] = useState("");
  const [replacing, setReplacing] = useState(false);
  const [test, setTest] = useState<TestState | null>(null);
  const keySaved = () => {
    setKey("");
    setReplacing(false);
  };

  const keyField = config?.spec.find((field) => field.sensitive) ?? null;
  const stored = keyField
    ? (config?.fields.find((field) => field.key === keyField.key)?.hasValue ?? false)
    : false;
  const on = extensionState(item) === "on";
  const canTurnOn = !keyField || stored || key.trim() !== "";
  const availability = config?.availability ?? item.availability;

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: queries.assistant._def });
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
    onSuccess: async (_saved, { agent, holds }) => {
      toast.success(
        holds
          ? t("{0} no longer has {1}", agent.name, item.name)
          : t("{0} now has {1}", agent.name, item.name),
      );
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

  const holders = agents.filter((agent) => holdsExtension(agent.toolNames, item.tools));
  const turnOn = () => {
    const configuration = keyField && key.trim() ? { [keyField.key]: key.trim() } : undefined;
    const count = availability === "AllAgents" ? null : holders.length;
    control.save(
      { enabled: true, configuration },
      count === null
        ? t("{0} is on for every agent", item.name)
        : count === 1
          ? t("{0} is on for 1 agent", item.name)
          : t("{0} is on for {1} agents", item.name, count),
      keySaved,
    );
  };

  const depthField = config?.spec.find((field) => field.key === "searchType") ?? null;
  const limitField = config?.spec.find((field) => field.key === "dailyRequestLimit") ?? null;
  const limit = Number(valueOf(config, "dailyRequestLimit") || item.dailyRequestLimit);
  const limits = [...new Set([...DAILY_LIMITS, limit])].filter((value) => value > 0).sort((a, b) => a - b);
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
              className="pd-kf"
              onSubmit={(event) => {
                event.preventDefault();
                if (!key.trim() || busy) return;
                if (on || stored) {
                  control.save(
                    { configuration: { [keyField.key]: key.trim() } },
                    t("{0} key saved", item.vendor),
                    keySaved,
                  );
                } else {
                  turnOn();
                }
              }}
            >
              <Ic n="key" s={13} />
              <input
                type="password"
                autoFocus
                autoComplete="new-password"
                aria-label={keyField.label}
                placeholder={keyField.placeholder || t("API key")}
                value={key}
                disabled={!canUpdate || busy}
                onChange={(event) => setKey(event.target.value)}
              />
              {(on || stored) && (
                <button type="submit" className="btn sm ink" disabled={!key.trim() || busy}>
                  {t("Save key")}
                </button>
              )}
            </form>
          )}
          {depthField && (
            <>
              <h4 className="mt">{t("Search depth")}</h4>
              <Seg
                className="sm"
                label={t("Search depth")}
                v={valueOf(config, "searchType") || depthField.default || "auto"}
                opts={(depthField.options ?? []).map(
                  (option) => [option, SEARCH_DEPTH_LABELS[option] ?? option] as const,
                )}
                onChange={(value) =>
                  canUpdate && control.save({ configuration: { searchType: value } })
                }
              />
            </>
          )}
          {limitField && (
            <>
              <h4 className="mt">{t("Daily request limit")}</h4>
              <Seg
                className="sm"
                label={t("Daily request limit")}
                v={limit}
                opts={limits.map((value) => [value, value.toLocaleString()] as const)}
                onChange={(value) =>
                  canUpdate && control.save({ configuration: { dailyRequestLimit: String(value) } })
                }
              />
            </>
          )}
          {config &&
            others.map((field) => (
              <OtherField
                key={`${field.key}:${config.version}`}
                field={field}
                value={valueOf(config, field.key)}
                disabled={!canUpdate || busy}
                onSave={(value) => control.save({ configuration: { [field.key]: value } })}
              />
            ))}
        </div>
        <div className="pd-s">
          <h4>{t("Available to")}</h4>
          <Seg
            className="sm"
            label={t("Available to")}
            v={availability}
            opts={[
              ["SelectedAgents", t("Agents you choose")],
              ["AllAgents", t("Every agent")],
            ]}
            onChange={(value) =>
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
      <div className="ad-bar sh-f">
        {canUpdate &&
          (on ? (
            <button
              type="button"
              className="xa d"
              disabled={busy}
              onClick={() => control.save({ enabled: false }, t("{0} is off", item.name))}
            >
              <Ic n="ban" s={13} />
              {t("Turn off")}
            </button>
          ) : (
            <button
              type="button"
              className="btn sm ink"
              disabled={!canTurnOn || busy}
              onClick={turnOn}
            >
              {control.saving && <i className="spn" />}
              {t("Turn on")}
            </button>
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
      field.sensitive ? "" : (config.fields.find((value) => value.key === field.key)?.value ?? field.default ?? ""),
    ]),
  );
}

/** A setting the extension takes beyond its key, depth and limit, saved when it is left. */
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
  const [text, setText] = useState(value || field.default || "");
  const commit = () => {
    const next = text.trim();
    if (field.type === "number" && next !== "" && !/^\d+$/.test(next)) {
      setText(value);
      return;
    }
    if (next !== (value || field.default || "")) onSave(next);
  };
  return (
    <>
      <h4 className="mt">{t(field.label)}</h4>
      <label className="inx">
        <input
          aria-label={t(field.label)}
          inputMode={field.type === "number" ? "numeric" : undefined}
          placeholder={field.placeholder}
          value={text}
          disabled={disabled}
          onChange={(event) => setText(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => event.key === "Enter" && event.currentTarget.blur()}
        />
      </label>
      {field.helpText && <p className="ad-h">{t(field.helpText)}</p>}
    </>
  );
}
