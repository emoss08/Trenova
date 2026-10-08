import { useApiMutation } from "@/hooks/use-api-mutation";
import { conversationPath } from "@/lib/conversation-path";
import {
  businessHoursLabel,
  capabilityOptions,
  isLocked,
  limitHigh,
  limitShare,
  updateAgentCapabilities,
  withToolMode,
  type AgentCapabilities,
  type AgentCapabilityMode,
  type AgentCapabilityTool,
  type UpdateAgentCapabilitiesInput,
} from "@/lib/graphql/agent-capabilities";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigate, useSearchParams } from "react-router";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { defineLabels } from "@trenova/shared/i18n/labels";

const MODE_LABEL: Record<AgentCapabilityMode, string> = defineLabels({
  Allowed: "Allowed",
  AskFirst: "Ask first",
  Off: "Off",
});

const wholeDollars = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 0,
});

function dollars(amount: string | null | undefined): string {
  const value = Number(amount ?? 0);
  return wholeDollars.format(Number.isFinite(value) ? value : 0);
}

function monthName(at: number, timeZone: string): string {
  try {
    return new Intl.DateTimeFormat("en-US", { month: "long", timeZone }).format(at * 1000);
  } catch {
    return new Intl.DateTimeFormat("en-US", { month: "long" }).format(at * 1000);
  }
}

function shortDate(at: number, timeZone: string): string {
  try {
    return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", timeZone }).format(
      at * 1000,
    );
  } catch {
    return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }).format(at * 1000);
  }
}

/**
 * What an agent can do: what it looks up and changes, and how far each change
 * goes on its own, who it hands work to and the limits it runs under. Anyone
 * who may use the agent reads it; only people who manage agents can change it,
 * and for everyone else every control is drawn but disabled.
 */
export function DeskAgentPage({ agentId }: { agentId: string }) {
  const t = useT();
  const navigate = useNavigate();
  const [search] = useSearchParams();
  const from = search.get("from");
  const queryClient = useQueryClient();
  const query = queries.assistant.agentCapabilities(agentId);
  const capsQuery = useQuery(query);
  const caps = capsQuery.data;

  const save = useApiMutation({
    mutationFn: (input: UpdateAgentCapabilitiesInput) => updateAgentCapabilities(agentId, input),
    onSuccess: (saved) => {
      queryClient.setQueryData(query.queryKey, saved);
      void queryClient.invalidateQueries({ queryKey: queries.assistant.myAgents().queryKey });
    },
    onError: () => {
      void queryClient.invalidateQueries({ queryKey: query.queryKey });
    },
    resourceName: "Agent",
  });

  const back = () => void navigate(from ? conversationPath(from) : "/desk");

  // One change at a time: each is saved against the version the page holds,
  // and the next waits for the one before it to come back.
  const editable = caps?.canEdit === true && !save.isPending;
  const change = (
    patch: Omit<UpdateAgentCapabilitiesInput, "version">,
    next?: AgentCapabilities,
  ) => {
    if (!caps || !editable) return;
    if (next) {
      queryClient.setQueryData(query.queryKey, next);
    }
    save.mutate({ ...patch, version: caps.version });
  };
  const setMode = (tool: AgentCapabilityTool, mode: AgentCapabilityMode) => {
    if (!caps || tool.mode === mode) return;
    change({ tools: [{ key: tool.key, mode }] }, withToolMode(caps, tool.key, mode));
  };

  return (
    <div className="dk-pg">
      <div className="dk-pg-in dk-narrow">
        <button type="button" className="dk-pg-back" onClick={back}>
          <DeskIcon name="chevL" size={13} />
          {from ? t("Back to conversation") : t("Back to Today")}
        </button>
        {capsQuery.isError ? (
          <p className="dk-ag-empty">
            {t("This agent's page could not be opened. You may not have access to it.")}
          </p>
        ) : !caps ? (
          <p className="dk-ag-empty">{t("Loading…")}</p>
        ) : (
          <>
            <header className="dk-ag-h">
              <DeskAgentTile
                agent={{
                  id: caps.agentId,
                  name: caps.name,
                  icon: caps.icon,
                  accent: caps.accent,
                  template: caps.template,
                }}
                size="lg"
              />
              <div>
                <h1>{caps.name}</h1>
                <p>{agentByline(caps, t)}</p>
              </div>
              <button
                type="button"
                className={cn("dk-ag-on", caps.enabled && "dk-on")}
                aria-pressed={caps.enabled}
                disabled={!editable}
                title={
                  caps.canEdit ? undefined : t("Only people who manage agents can change this")
                }
                onClick={() =>
                  change({ enabled: !caps.enabled }, { ...caps, enabled: !caps.enabled })
                }
              >
                <i />
                {caps.enabled ? t("On") : t("Off")}
              </button>
            </header>
            {!caps.canEdit && (
              <p className="dk-pgc-tip">
                {t(
                  "You can see what this agent can do. Changing it needs permission to manage agents.",
                )}
              </p>
            )}

            <ToolSection
              title={t("Look things up")}
              sub={t("What it can read")}
              tools={caps.readTools}
              canEdit={editable}
              onMode={setMode}
            />
            <ToolSection
              title={t("Make changes")}
              sub={t("What it can change, and when it asks you")}
              tools={caps.writeTools}
              canEdit={editable}
              onMode={setMode}
            />

            <section className="dk-pgc">
              <div className="dk-pgc-h">
                <h2>{t("Hands off to")}</h2>
                <span>{t("Questions outside its work go to these agents")}</span>
              </div>
              <div className="dk-ag-l">
                {caps.handoffs.length === 0 ? (
                  <p className="dk-ag-empty">{t("It doesn't hand work to other agents.")}</p>
                ) : (
                  caps.handoffs.map((handoff) => (
                    <div key={handoff.agentId} className="dk-ag-r">
                      <span className="dk-ag-t">
                        <b>{handoff.topic}</b>
                      </span>
                      <span className="dk-ag-to">
                        <DeskIcon name="handoff" size={13} />
                        {handoff.name}
                      </span>
                    </div>
                  ))
                )}
              </div>
            </section>

            <Limits caps={caps} canEdit={editable} onChange={change} />
          </>
        )}
      </div>
    </div>
  );
}

/** "{desc} · {model} · set up by {owner}", leaving out what is not known. */
export function agentByline(caps: AgentCapabilities, t: TranslateFn): string {
  return [caps.description, caps.model, caps.setUpBy ? t("set up by {0}", caps.setUpBy) : ""]
    .filter((part) => part.trim() !== "")
    .join(" · ");
}

function ToolSection({
  title,
  sub,
  tools,
  canEdit,
  onMode,
}: {
  title: string;
  sub: string;
  tools: readonly AgentCapabilityTool[];
  canEdit: boolean;
  onMode: (tool: AgentCapabilityTool, mode: AgentCapabilityMode) => void;
}) {
  const t = useT();

  return (
    <section className="dk-pgc">
      <div className="dk-pgc-h">
        <h2>{title}</h2>
        <span>{sub}</span>
      </div>
      <div className="dk-ag-l">
        {tools.length === 0 ? (
          <p className="dk-ag-empty">{t("Nothing here yet.")}</p>
        ) : (
          tools.map((tool) => (
            <div key={tool.key} className="dk-ag-r">
              <span className="dk-ag-t">
                <b>{tool.label}</b>
                <code>{tool.key}</code>
                {isLocked(tool) && (
                  <em>
                    <DeskIcon name="lock" size={11} />
                    {tool.lockReason}
                  </em>
                )}
              </span>
              <div
                className={cn("dk-mm-seg dk-sm", isLocked(tool) && "dk-locked")}
                role="radiogroup"
                aria-label={tool.label}
              >
                {capabilityOptions(tool, canEdit).map((option) => (
                  <button
                    key={option.mode}
                    type="button"
                    role="radio"
                    aria-checked={option.selected}
                    disabled={option.disabled}
                    className={cn(option.selected && "dk-on")}
                    onClick={() => onMode(tool, option.mode)}
                  >
                    {MODE_LABEL[option.mode]}
                  </button>
                ))}
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}

function Limits({
  caps,
  canEdit,
  onChange,
}: {
  caps: AgentCapabilities;
  canEdit: boolean;
  onChange: (
    patch: Omit<UpdateAgentCapabilitiesInput, "version">,
    next?: AgentCapabilities,
  ) => void;
}) {
  const t = useT();
  const limits = caps.limits;
  const zone = limits.businessHoursTimezone || "UTC";
  const spent = Number(limits.monthlySpentUsd);
  const budget = limits.monthlyBudgetUsd === null ? 0 : Number(limits.monthlyBudgetUsd);
  const hoursOn = limits.businessHoursOnly;

  return (
    <section className="dk-pgc">
      <div className="dk-pgc-h">
        <h2>{t("Limits")}</h2>
      </div>
      <div className="dk-ag-lim">
        <LimitBar
          label={t("Requests today")}
          sub={limits.dailyRequestLimit > 0 ? t("Resets at midnight") : t("No daily limit")}
          value={String(limits.requestsToday)}
          limit={limits.dailyRequestLimit > 0 ? String(limits.dailyRequestLimit) : null}
          used={limits.requestsToday}
          cap={limits.dailyRequestLimit}
        />
        <LimitBar
          label={t("{0} budget", monthName(limits.monthStart, zone))}
          sub={
            budget > 0
              ? t(
                  "{0} of {1} · resets {2}",
                  dollars(limits.monthlySpentUsd),
                  dollars(limits.monthlyBudgetUsd),
                  shortDate(limits.monthResetsAt, zone),
                )
              : t("No monthly budget")
          }
          value={dollars(limits.monthlySpentUsd)}
          limit={budget > 0 ? dollars(limits.monthlyBudgetUsd) : null}
          used={spent}
          cap={budget}
        />
        <div className="dk-ag-r">
          <span className="dk-ag-t">
            <b>{t("Largest single change")}</b>
            <em>{t("Bigger batches are split and approved separately")}</em>
          </span>
          <span className="dk-ag-n">
            {limits.maxChangeItems}
            <i> {t("{0, plural, one {item} other {items}}", limits.maxChangeItems)}</i>
          </span>
        </div>
        <div className="dk-ag-r">
          <span className="dk-ag-t">
            <b>{t("Only change things during business hours")}</b>
            <em>{businessHoursLabel(limits.businessHoursStart, limits.businessHoursEnd, zone)}</em>
          </span>
          <button
            type="button"
            role="switch"
            aria-checked={hoursOn}
            aria-label={t("Only change things during business hours")}
            className={cn("dk-ag-on dk-sm", hoursOn && "dk-on")}
            disabled={!canEdit}
            onClick={() =>
              onChange(
                { businessHoursOnly: !hoursOn },
                { ...caps, limits: { ...limits, businessHoursOnly: !hoursOn } },
              )
            }
          >
            <i />
          </button>
        </div>
      </div>
    </section>
  );
}

function LimitBar({
  label,
  sub,
  value,
  limit,
  used,
  cap,
}: {
  label: string;
  sub: string;
  value: string;
  limit: string | null;
  used: number;
  cap: number;
}) {
  return (
    <div>
      <span>
        <b>{label}</b>
        <em>{sub}</em>
      </span>
      <span className="dk-ag-n">
        {value}
        {limit !== null && <i>/ {limit}</i>}
      </span>
      {limit !== null && (
        <span className={cn("dk-ag-bar", limitHigh(used, cap) && "dk-hi")}>
          <i style={{ width: `${limitShare(used, cap) * 100}%` }} />
        </span>
      )}
    </div>
  );
}
