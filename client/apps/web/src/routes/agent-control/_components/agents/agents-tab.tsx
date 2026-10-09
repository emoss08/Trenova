import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  AGENT_CONTROL_QUERY_KEY,
  agentControlQueryOptions,
  updateAgentControl,
} from "@/lib/graphql/agent-control";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryState, useQueryStates } from "nuqs";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import {
  AGENT_EDIT_PARAM,
  AGENT_FILTER_PARAM,
  AGENT_NEW_PARAM,
  AGENT_OPEN_PARAM,
  AGENT_TRY_PARAM,
  TOOL_PICKER_PARAM,
  agentEditParser,
  agentFilterParser,
  agentNewParser,
  agentOpenParser,
  agentTryParser,
  toolPickerParser,
  type AgentFilter,
} from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { controlInput } from "../agent-control-options";
import { Search, useSlashFocus } from "../kit/controls";
import { Ic } from "../kit/ic";
import { Switch } from "../kit/layout";
import { Tile } from "../kit/marks";
import { ConfirmDialog } from "../kit/modal";
import { ReadSheet } from "../kit/read-sheet";
import { NovaSummary } from "../nova/nova-summary";
import { useNovaTargets } from "../nova/use-nova-targets";
import { workingAgents } from "../overview/agents-at-work";
import { usePauseAgents } from "../overview/use-pause-agents";
import { toAgentPanelRow, toSaveRequest } from "./agent-form-schema";
import { AgentDetail } from "./agent-detail";
import { AgentRow } from "./agent-row";
import { groupAgentsByTrigger } from "./agent-roster";
import { AgentBuilder, type BuilderStart } from "./builder/agent-builder";
import {
  filterRoster,
  modeFlags,
  rosterCounts,
  rowFact,
  tierSplit,
  type AgentMode,
} from "./roster-model";
import { TRIGGER_LABELS, TRIGGER_NOTES } from "./trigger-meta";

const DESK_DECISIONS_PATH = "/desk/decisions";
const SUMMARY_STALE_MS = 30_000;
const PENDING_REFRESH_MS = 3_000;
const WORKING_REFRESH_MS = 15_000;

const SHELF_ICON = {
  Chat: "chat",
  Scheduled: "calendar",
  Event: "bolt",
  Continuous: "refresh",
} as const;

const builderParsers = {
  [AGENT_EDIT_PARAM]: agentEditParser,
  [AGENT_NEW_PARAM]: agentNewParser,
  [AGENT_TRY_PARAM]: agentTryParser,
  [TOOL_PICKER_PARAM]: toolPickerParser,
};

/** The builder closed, with the panels it opened over the agent it was editing. */
const BUILDER_CLOSED = {
  [AGENT_EDIT_PARAM]: null,
  [AGENT_NEW_PARAM]: null,
  [AGENT_TRY_PARAM]: null,
  [TOOL_PICKER_PARAM]: null,
} as const;

/**
 * Every agent in the organization, shelved by what starts it: what each has done in the
 * last two weeks, how often people approve it, what waits on a person, and how its tools
 * split between reading, proposing and acting. A row opens the agent to read; Edit opens
 * the builder.
 */
export default function AgentsTab() {
  const t = useT();
  const navigate = useNavigate();
  const go = useAIControlNavigation();
  const onTarget = useNovaTargets();
  const queryClient = useQueryClient();
  const searchRef = useRef<HTMLInputElement>(null);
  useSlashFocus(searchRef);

  const [filter, setFilter] = useQueryState(AGENT_FILTER_PARAM, agentFilterParser);
  const [{ [AGENT_EDIT_PARAM]: editId, [AGENT_NEW_PARAM]: start }, setBuilder] =
    useQueryStates(builderParsers);
  const [query, setQuery] = useState("");
  const [openId, setOpenId] = useQueryState(AGENT_OPEN_PARAM, agentOpenParser);
  const [menuOpen, setMenuOpen] = useState(false);
  const [removing, setRemoving] = useState<AgentDefinitionRow | null>(null);

  const { allowed: canCreate } = usePermission(Resource.AgentDefinition, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.AgentDefinition, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.AgentDefinition, Operation.Delete);
  const { allowed: canRun } = usePermission(Resource.AgentRun, Operation.Create);
  const { allowed: canUpdateControl } = usePermission(Resource.AgentControl, Operation.Update);

  const summaryQuery = useQuery({
    ...queries.aiControl.summary("Agents"),
    staleTime: SUMMARY_STALE_MS,
    refetchInterval: (state) => (state.state.data?.pending ? PENDING_REFRESH_MS : false),
  });
  const agentsQuery = useQuery(queries.assistant.agents(false));
  const rosterQuery = useQuery(queries.aiControl.roster());
  const catalogQuery = useQuery(queries.assistant.toolCatalog());
  const eventsQuery = useQuery(queries.assistant.eventKinds());
  const controlQuery = useQuery(agentControlQueryOptions());
  const facts = summaryQuery.data?.facts;
  const noProvider = facts ? facts.providersOn === 0 : false;
  const paused = facts?.paused ?? false;
  const runsQuery = useQuery({
    ...queries.aiControl.workingRuns(),
    refetchInterval: noProvider || paused ? false : WORKING_REFRESH_MS,
  });

  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const catalog = useMemo(() => catalogQuery.data?.tools ?? [], [catalogQuery.data?.tools]);
  const eventLabels = useMemo(
    () => new Map((eventsQuery.data?.events ?? []).map((event) => [event.kind, event.label])),
    [eventsQuery.data?.events],
  );
  const eventLabel = useCallback((kind: string) => eventLabels.get(kind) ?? kind, [eventLabels]);
  const working = useMemo(
    () => (noProvider || paused ? [] : workingAgents(agents, runsQuery.data ?? [])),
    [agents, noProvider, paused, runsQuery.data],
  );
  const workingById = useMemo(
    () => new Map(working.map((entry) => [entry.agent.id, entry.run])),
    [working],
  );
  const counts = useMemo(() => rosterCounts(agents), [agents]);
  const shelves = useMemo(
    () => groupAgentsByTrigger(filterRoster(agents, filter, query)),
    [agents, filter, query],
  );
  const splits = useMemo(
    () => new Map(agents.map((agent) => [agent.id, tierSplit(agent, catalog)])),
    [agents, catalog],
  );
  const on = agents.filter((agent) => agent.enabled);
  const pending = on.reduce((total, agent) => total + agent.pendingProposals, 0);
  const waitingNames = on.filter((agent) => agent.pendingProposals > 0).map((agent) => agent.name);
  const openAgent = agents.find((agent) => agent.id === openId) ?? null;
  const openMissing = openId !== null && agentsQuery.isSuccess && openAgent === null;
  const editing =
    editId && canUpdate ? (agents.find((agent) => agent.id === editId) ?? null) : null;
  const creating = editId === null && start !== null && canCreate;
  const builderOpen = editing !== null || creating;

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.assistant._def }),
      queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
    ]);
  }, [queryClient]);

  const save = useApiMutation({
    mutationFn: ({
      agent,
      patch,
    }: {
      agent: AgentDefinitionRow;
      patch: { enabled?: boolean; shadowMode?: boolean; simulationMode?: boolean };
      message: string;
    }) =>
      apiService.agentDefinitionService.update(agent.id, {
        ...toSaveRequest(toAgentPanelRow(agent)),
        ...patch,
      }),
    onSuccess: async (_saved, { message }) => {
      toast.success(message);
      await refresh();
    },
    resourceName: t("Agent"),
  });

  const run = useApiMutation({
    mutationFn: (agent: AgentDefinitionRow) =>
      apiService.agentRunService.start({ agentDefinitionId: agent.id }),
    onSuccess: async (_run, agent) => {
      toast.success(t("{0} is running", agent.name));
      await refresh();
    },
    resourceName: t("Agent run"),
  });

  const remove = useApiMutation({
    mutationFn: (agent: AgentDefinitionRow) => apiService.agentDefinitionService.remove(agent.id),
    onSuccess: async (_result, agent) => {
      toast.success(t("{0} removed", agent.name));
      setRemoving(null);
      void setOpenId(null);
      await refresh();
    },
    resourceName: t("Agent"),
  });

  const earnedOn = useApiMutation({
    mutationFn: () => {
      if (!controlQuery.data) {
        throw new Error("The organization's settings have not loaded");
      }
      return updateAgentControl(controlInput(controlQuery.data, { earnedAutonomy: true }));
    },
    onSuccess: async (saved) => {
      queryClient.setQueryData(AGENT_CONTROL_QUERY_KEY, saved);
      toast.success(t("Earned autonomy is on"));
      await refresh();
    },
    resourceName: t("Organization-wide settings"),
  });

  const pause = usePauseAgents(controlQuery.data);

  const toggle = useCallback(
    (agent: AgentDefinitionRow, enabled: boolean) =>
      save.mutate({
        agent,
        patch: { enabled },
        message: enabled ? t("{0} is on", agent.name) : t("{0} is off", agent.name),
      }),
    [save, t],
  );
  const setMode = useCallback(
    (agent: AgentDefinitionRow, mode: AgentMode) =>
      save.mutate({
        agent,
        patch: modeFlags(mode),
        message:
          mode === "live"
            ? t("{0} is live", agent.name)
            : mode === "shadow"
              ? t("{0} runs in shadow", agent.name)
              : t("{0} runs in simulation", agent.name),
      }),
    [save, t],
  );
  const ask = useCallback(
    (agent: AgentDefinitionRow) => void navigate(`/desk/agents/${encodeURIComponent(agent.id)}`),
    [navigate],
  );
  const reviewInDesk = useCallback(() => void navigate(DESK_DECISIONS_PATH), [navigate]);
  const edit = useCallback(
    (agent: AgentDefinitionRow) => {
      void setOpenId(null);
      void setBuilder({ ...BUILDER_CLOSED, [AGENT_EDIT_PARAM]: agent.id });
    },
    [setOpenId, setBuilder],
  );
  const create = useCallback(
    (from: BuilderStart) => void setBuilder({ ...BUILDER_CLOSED, [AGENT_NEW_PARAM]: from }),
    [setBuilder],
  );
  const closeBuilder = useCallback(() => void setBuilder(BUILDER_CLOSED), [setBuilder]);
  const activity = useCallback(
    (agent: AgentDefinitionRow) =>
      go({
        tab: "activity",
        view: "runs",
        fieldFilters: [{ field: "agentDefinitionId", operator: "eq", value: agent.id }],
      }),
    [go],
  );

  // An address that names an agent removed since, or one never in this organization, opens
  // nothing; it is cleared rather than left to open a later agent that reuses the key.
  useEffect(() => {
    if (openMissing) {
      void setOpenId(null, { history: "replace" });
    }
  }, [openMissing, setOpenId]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        builderOpen ||
        (target && (target.isContentEditable || /INPUT|TEXTAREA|SELECT/.test(target.tagName)))
      ) {
        return;
      }
      const key = event.key.toLowerCase();
      if (key === "n" && canCreate && !openAgent) {
        event.preventDefault();
        setMenuOpen(true);
        return;
      }
      if (!openAgent) {
        return;
      }
      if (key === "e" && canUpdate) {
        event.preventDefault();
        edit(openAgent);
      } else if (key === "r" && openAgent.triggerMode !== "Chat" && openAgent.enabled && canRun) {
        event.preventDefault();
        run.mutate(openAgent);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [builderOpen, canCreate, canRun, canUpdate, edit, openAgent, run]);

  const startItems: { icon: ReactNode; label: string; note: string; start: BuilderStart }[] = [
    {
      icon: <Ic n="chat" s={14} />,
      label: t("Desk agent"),
      note: t("Answers people in the assistant"),
      start: "chat",
    },
    {
      icon: <Ic n="calendar" s={14} />,
      label: t("Scheduled report"),
      note: t("Runs on a timetable and sends a summary"),
      start: "scheduled",
    },
    {
      icon: <Ic n="bolt" s={14} />,
      label: t("Event watcher"),
      note: t("Wakes when something happens"),
      start: "event",
    },
    {
      icon: <Ic n="sparkle" s={14} />,
      label: t("Start blank"),
      note: t("Describe it for Nova to draft, or set it up yourself"),
      start: "blank",
    },
  ];

  const filters: [AgentFilter, string][] = [
    ["all", t("All")],
    ["waiting", t("Waiting")],
    ["shadow", t("Shadow")],
    ["off", t("Off")],
  ];

  return (
    <div className="tabp">
      {!noProvider && (
        <>
          <NovaSummary
            context={t("Agents")}
            segments={summaryQuery.data?.segments}
            loading={summaryQuery.isLoading}
            working={working.length > 0}
            onTarget={onTarget}
            control={
              pending > 0 ? (
                <>
                  <Button type="button" variant="default" size="lg" onClick={reviewInDesk}>
                    <Ic n="inbox" s={13} />
                    {t("Review {0} in Desk", pending.toLocaleString())}
                  </Button>
                  <span>{formatList(waitingNames)}</span>
                </>
              ) : undefined
            }
          />
          {working.length > 0 && (
            <div className="lv">
              {working.map(({ agent, run: going }) => (
                <button
                  key={going.id}
                  type="button"
                  className="lv-c"
                  onClick={() => void setOpenId(agent.id)}
                >
                  <span className="pc-mk">
                    <Tile agent={agent} s={22} />
                    <i className="pc-dot run sm" />
                  </span>
                  <span className="lv-t">
                    <b>{agent.name}</b>
                    <span className="shm">{going.summary.trim() || t("Working")}</span>
                  </span>
                </button>
              ))}
            </div>
          )}
        </>
      )}
      {noProvider && (
        <div className="bnr">
          <Ic n="plug" s={14} />
          <span>
            <b>{t("Agents can't answer until a provider is connected.")}</b>{" "}
            {t("Their settings are kept; they start as soon as one is on.")}
          </span>
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={() => go({ tab: "providers", panel: { mode: "create" } })}
          >
            {t("Connect a provider")}
          </Button>
        </div>
      )}
      {paused && !noProvider && (
        <div className="bnr w">
          <Ic n="pause" s={13} w={2.4} />
          <span>
            <b>{t("All agents are paused organization-wide.")}</b>{" "}
            {t("Each one below keeps its own switch for when you resume.")}
          </span>
          {canUpdateControl && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={pause.isPending}
              onClick={() => pause.mutate(false)}
            >
              {t("Resume")}
            </Button>
          )}
        </div>
      )}
      <div className="tb">
        <Search
          value={query}
          onChange={setQuery}
          placeholder={t("Search agents")}
          inputRef={searchRef}
        />
        <div className="seg" role="radiogroup" aria-label={t("Show")}>
          {filters.map(([key, label]) => (
            <button
              key={key}
              type="button"
              role="radio"
              aria-checked={filter === key}
              className={filter === key ? "on" : undefined}
              onClick={() => void setFilter(key === "all" ? null : key)}
            >
              {label}
              <em className="mono">{counts[key]}</em>
            </button>
          ))}
        </div>
        <span className="sp" />
        <span className="tb-ct mono">{t("{0} of {1} on", on.length, agents.length)}</span>
        {canCreate && (
          <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
            <DropdownMenuTrigger render={<Button type="button" variant="default" shortcut="N" />}>
              <Ic n="plus" s={13} />
              {t("New agent")}
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-72">
              <DropdownMenuGroup>
                <DropdownMenuLabel>{t("Start from")}</DropdownMenuLabel>
                {startItems.map((item) => (
                  <DropdownMenuItem
                    key={item.start}
                    title={item.label}
                    description={item.note}
                    startContent={item.icon}
                    onClick={() => create(item.start)}
                  />
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      {shelves.map((shelf) => (
        <section key={shelf.trigger} className="shelf">
          <header className="shelf-h">
            <Ic n={SHELF_ICON[shelf.trigger]} s={13} />
            <b>{t(TRIGGER_LABELS[shelf.trigger])}</b>
            <span>{t(TRIGGER_NOTES[shelf.trigger])}</span>
            <em className="mono">{shelf.agents.length}</em>
          </header>
          <div className="al-hd" aria-hidden>
            <span />
            <span>{t("Agent")}</span>
            <span>{t("Last 14 days")}</span>
            <span>{t("Approved")}</span>
            <span>{t("Waiting")}</span>
            <span>{t("Tools")}</span>
            <span />
          </div>
          <div className="rows">
            {shelf.agents.map((agent) => (
              <AgentRow
                key={agent.id}
                agent={agent}
                stat={rosterQuery.data?.get(agent.id)}
                working={workingById.get(agent.id)}
                fact={rowFact(agent, eventLabel)}
                split={splits.get(agent.id) ?? EMPTY_SPLIT}
                open={openId === agent.id}
                noProvider={noProvider}
                canUpdate={canUpdate}
                canRun={canRun}
                onOpen={() => void setOpenId((current) => (current === agent.id ? null : agent.id))}
                onToggle={(enabled) => toggle(agent, enabled)}
                onRun={() => run.mutate(agent)}
                onAsk={() => ask(agent)}
                onReviewWaiting={reviewInDesk}
              />
            ))}
          </div>
        </section>
      ))}
      {!agentsQuery.isLoading && shelves.length === 0 && (
        <div className="empty">
          <b>{agents.length ? t("No agents match") : t("No agents yet")}</b>
          <span>
            {agents.length
              ? t("Try another word, or clear the filter.")
              : t("Describe a job and Nova drafts the agent, or start from a template.")}
          </span>
          {agents.length ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                setQuery("");
                void setFilter(null);
              }}
            >
              {t("Clear")}
            </Button>
          ) : (
            canCreate && (
              <Button type="button" variant="default" size="sm" onClick={() => create("blank")}>
                <Ic n="plus" s={12} />
                {t("New agent")}
              </Button>
            )
          )}
        </div>
      )}
      <ReadSheet
        open={openAgent !== null}
        onClose={() => void setOpenId(null)}
        label={openAgent?.name ?? t("Agent")}
        head={
          openAgent && (
            <>
              <Tile agent={openAgent} s={36} />
              <div className="sh-t">
                <b>{openAgent.name}</b>
                <span>
                  {[
                    t(TRIGGER_LABELS[openAgent.triggerMode]),
                    agentToolsLabel(splits.get(openAgent.id) ?? EMPTY_SPLIT, t),
                    ...(openAgent.systemKey ? [t("system")] : []),
                  ].join(" · ")}
                </span>
              </div>
              {canUpdate && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  shortcut="E"
                  onClick={() => edit(openAgent)}
                >
                  <Ic n="edit" s={12} />
                  {t("Edit")}
                </Button>
              )}
              <Switch
                on={openAgent.enabled}
                label={openAgent.enabled ? t("Disable agent") : t("Enable agent")}
                disabled={!canUpdate}
                onChange={(enabled) => toggle(openAgent, enabled)}
              />
            </>
          )
        }
      >
        {openAgent && (
          <>
            <p className="sh-d">{openAgent.description}</p>
            <AgentDetail
              agent={openAgent}
              stat={rosterQuery.data?.get(openAgent.id)}
              split={splits.get(openAgent.id) ?? EMPTY_SPLIT}
              control={controlQuery.data}
              eventLabel={eventLabel}
              canUpdate={canUpdate}
              canUpdateControl={canUpdateControl}
              canRun={canRun}
              canDelete={canDelete}
              onMode={(mode) => setMode(openAgent, mode)}
              onEdit={() => edit(openAgent)}
              onRun={() => run.mutate(openAgent)}
              onAsk={() => ask(openAgent)}
              onActivity={() => activity(openAgent)}
              onRemove={() => setRemoving(openAgent)}
              onEarnedAutonomy={() => earnedOn.mutate()}
            />
          </>
        )}
      </ReadSheet>
      <ConfirmDialog
        open={removing !== null}
        onClose={() => setRemoving(null)}
        title={t("Remove {0}?", removing?.name ?? "")}
        description={t(
          "Its runs and audit trail are kept. Open proposals are withdrawn and people can no longer pick it in Desk.",
        )}
        typed={removing?.name}
        danger
        busy={remove.isPending}
        confirmLabel={t("Remove agent")}
        onConfirm={() => removing && remove.mutate(removing)}
      />
      {(editing || creating) && (
        <AgentBuilder
          key={editing?.id ?? `new:${start ?? "blank"}`}
          agent={editing}
          start={start ?? "blank"}
          onClose={closeBuilder}
          onOpenActivity={editing ? () => activity(editing) : undefined}
        />
      )}
    </div>
  );
}

const EMPTY_SPLIT: readonly [number, number, number] = [0, 0, 0];

function agentToolsLabel(
  split: readonly [number, number, number],
  t: ReturnType<typeof useT>,
): string {
  const total = split[0] + split[1] + split[2];
  if (total === 0) return t("no task tools");
  return total === 1 ? t("1 tool") : t("{0} tools", total);
}
