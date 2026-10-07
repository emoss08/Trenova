import { Dialog } from "@base-ui/react/dialog";
import { agentAccessPreviewRequest } from "@/lib/graphql/agent-access";
import {
  fetchAgentDefinition,
  fetchAgentVersionDraft,
} from "@/lib/graphql/agent-builder";
import { agentControlQueryOptions } from "@/lib/graphql/agent-control";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AutonomyTier, SaveAgentDefinitionRequest } from "@/types/assistant";
import { zodResolver } from "@hookform/resolvers/zod";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { downloadJsonFile, slugify } from "@trenova/shared/lib/utils";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useRef, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { ConflictBar } from "../../edit/conflict-bar";
import { ChangeReview, type EditFields } from "../../edit/change-review";
import { SaveBar } from "../../edit/save-bar";
import { useEditFlow } from "../../edit/use-edit-flow";
import { Menu } from "../../kit/controls";
import { Ic } from "../../kit/ic";
import { Switch } from "../../kit/layout";
import { Tile } from "../../kit/marks";
import { ConfirmDialog } from "../../kit/modal";
import { Pop } from "../../kit/pop";
import { accessToSave, NEW_AGENT_ACCESS } from "../agent-access-save";
import {
  accessOf,
  agentFormSchema,
  toAgentPanelRow,
  toSaveRequest,
  type AgentFormValues,
} from "../agent-form-schema";
import { toolTitle } from "../tool-catalog";
import { Blk, type BlockId } from "./block";
import { BuilderRail, type RailEntry } from "./builder-rail";
import {
  changeTiers,
  checklist,
  chosenTools,
  readiness,
  startValues,
  triggerProblem,
  type BuilderStart,
} from "./builder-model";
import { CreateIntro } from "./create-intro";
import { IdentityBlock } from "./identity-block";
import { InstructionsBlock } from "./instructions-block";
import { LimitsBlock } from "./limits-block";
import { RecordBlock } from "./record-block";
import { SummarySentence } from "./summary-sentence";
import { TeamBlock } from "./team-block";
import { ToolBench } from "./tool-bench";
import { TriggerBlock } from "./trigger-block";
import { TryPanel } from "./try-panel";
import { schedulePreset } from "./schedule-presets";
import { useAgentDrafting } from "./use-agent-drafting";

export { BUILDER_STARTS, type BuilderStart } from "./builder-model";

/** How long the sections Nova filled glow. */
const FRESH_MS = 2_200;
/** Instructions are checked against the tools this long after the last keystroke. */
const LINT_DEBOUNCE_MS = 600;
const SHADOW_REPORT_DAYS = 30;

type Mode = "live" | "shadow" | "sim";

const BLOCKS: BlockId[] = ["who", "instr", "trig", "tools", "limits", "team", "record"];

/** The draft's own values, without what the roster row adds for drawing it. */
function formValuesOf(agent: AgentDefinitionRow): AgentFormValues {
  const {
    id: _id,
    updatedAt: _updatedAt,
    systemKey: _systemKey,
    delegates: _delegates,
    accessRoles: _accessRoles,
    ...values
  } = toAgentPanelRow(agent);
  return values;
}

function modeOf(values: Pick<AgentFormValues, "shadowMode" | "simulationMode">): Mode {
  return values.simulationMode ? "sim" : values.shadowMode ? "shadow" : "live";
}

type AgentBuilderProps = {
  /** The agent being edited, or null for a new one. */
  agent: AgentDefinitionRow | null;
  start: BuilderStart;
  onClose: () => void;
  onOpenActivity?: () => void;
  onOpened?: (agentId: string) => void;
};

/**
 * The whole viewport for building one agent: a checklist down the left, the agent on a
 * canvas section by section, and a panel to try the unsaved draft. A new agent opens on
 * "What should it do?" and starts in shadow.
 */
export function AgentBuilder({ agent, start, onClose, onOpenActivity, onOpened }: AgentBuilderProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const timezone = useMemo(() => Intl.DateTimeFormat().resolvedOptions().timeZone, []);
  const catalogQuery = useQuery(queries.assistant.toolCatalog());
  const rulesQuery = useQuery(queries.agentSafety.toolRules());
  const eventsQuery = useQuery(queries.assistant.eventKinds());
  const agentsQuery = useQuery(queries.assistant.agents(false));
  const providersQuery = useQuery(queries.aiProvider.list());
  const controlQuery = useQuery(agentControlQueryOptions());
  const budgetQuery = useQuery({
    ...queries.assistant.agentBudget(agent?.id ?? ""),
    enabled: agent !== null,
  });
  const versionsQuery = useQuery({
    ...queries.agentBuilder.versions(agent?.id ?? ""),
    enabled: agent !== null,
  });
  const drafting = useAgentDrafting();
  const rosterQuery = useQuery({ ...queries.aiControl.roster(), enabled: agent !== null });
  const stat = agent ? rosterQuery.data?.get(agent.id) : undefined;

  const catalog = useMemo(() => catalogQuery.data?.tools ?? [], [catalogQuery.data?.tools]);
  const rules = useMemo(() => rulesQuery.data ?? new Map(), [rulesQuery.data]);
  const events = useMemo(() => eventsQuery.data?.events ?? [], [eventsQuery.data?.events]);
  const eventLabel = useCallback(
    (kind: string) => events.find((event) => event.kind === kind)?.label ?? kind,
    [events],
  );

  const form = useForm<AgentFormValues>({
    resolver: zodResolver(agentFormSchema) as Resolver<AgentFormValues>,
    defaultValues: agent ? formValuesOf(agent) : startValues(start, { timezone, catalog }),
  });
  const values = useWatch({ control: form.control }) as AgentFormValues;
  const savedAccess = useRef(agent ? accessOf(formValuesOf(agent)) : NEW_AGENT_ACCESS);

  const [intro, setIntro] = useState(agent === null);
  const [fresh, setFresh] = useState<BlockId[]>([]);
  const [tryOpen, setTryOpen] = useState(false);
  const [tried, setTried] = useState<string | null>(null);
  const [active, setActive] = useState<BlockId>("who");
  const [pop, setPop] = useState<"mode" | "more" | null>(null);
  const [removing, setRemoving] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const [tightening, setTightening] = useState(false);
  const canvas = useRef<HTMLDivElement>(null);
  const popup = useRef<HTMLDivElement>(null);

  const draftRequest: SaveAgentDefinitionRequest = useMemo(() => toSaveRequest(values), [values]);
  const snapshot = useMemo(() => JSON.stringify(draftRequest), [draftRequest]);

  const lintRequest = useDebounce(
    { instructions: values.instructions, toolNames: values.toolNames },
    LINT_DEBOUNCE_MS,
  );
  const lintQuery = useQuery({
    ...queries.agentBuilder.lint(lintRequest),
    enabled: !intro && lintRequest.instructions.trim() !== "",
    staleTime: 60_000,
  });
  const findings = lintRequest.instructions.trim() ? (lintQuery.data ?? []) : [];

  const accessRequest = useDebounce(
    agentAccessPreviewRequest({
      agentId: agent?.id ?? "",
      toolNames: values.toolNames,
      accessMode: values.accessMode,
    }),
    LINT_DEBOUNCE_MS,
  );
  const accessQuery = useQuery({
    ...queries.assistant.agentAccessPreview(accessRequest),
    enabled: !intro,
  });
  const sensitiveTools = accessQuery.data?.sensitiveTools ?? [];
  const roles = accessQuery.data?.roles ?? [];

  const shadowQuery = useQuery({
    ...queries.agentBuilder.shadow(agent?.id ?? "", SHADOW_REPORT_DAYS),
    enabled:
      agent !== null &&
      Boolean(form.formState.defaultValues?.shadowMode) &&
      modeOf(values) === "live",
  });

  const chosen = chosenTools(values, catalog);
  const { reads, byTier } = changeTiers(values, catalog);
  const status = checklist({
    values,
    findings: findings.length,
    openRisk: values.accessMode === "Everyone" ? sensitiveTools.length : 0,
    chosen: chosen.length,
    tried,
    draft: snapshot,
  });
  const ready = readiness(status);
  const mode = modeOf(values);
  const problem = triggerProblem(values);
  const invalid = intro
    ? t("Describe it or pick a start")
    : !values.name.trim()
      ? t("Give it a name")
      : problem === "events"
        ? t("Pick at least one event")
        : problem === "schedule"
          ? t("Set a schedule")
          : problem === "badSchedule"
            ? t("The schedule isn't one Trenova can read")
            : problem === "interval"
              ? t("Run at most once a minute")
              : null;

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.assistant._def }),
      queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
      queryClient.invalidateQueries({ queryKey: queries.agentBuilder._def }),
    ]);
  }, [queryClient]);

  const save = useCallback(
    async (next: AgentFormValues): Promise<AgentFormValues> => {
      const access = accessOf(next);
      if (agent) {
        await apiService.agentDefinitionService.update(
          agent.id,
          toSaveRequest(next, accessToSave(access, savedAccess.current)),
        );
        savedAccess.current = access;
        const reloaded = await fetchAgentDefinition(agent.id);
        await refresh();
        toast.success(t("{0} saved", next.name.trim()));
        return reloaded ? formValuesOf(reloaded) : next;
      }
      const created = await apiService.agentDefinitionService.create(
        toSaveRequest(next, accessToSave(access, NEW_AGENT_ACCESS)),
      );
      await refresh();
      toast.success(
        next.simulationMode
          ? t("{0} created in simulation", created.name)
          : next.shadowMode
            ? t("{0} created in shadow", created.name)
            : t("{0} created", created.name),
      );
      return next;
    },
    [agent, refresh, t],
  );

  const flow = useEditFlow({
    form,
    onSave: save,
    onClose,
    create: agent === null,
    invalid,
    enabled: true,
    resourceName: t("Agent"),
    loadLatest: agent
      ? async () => {
          const latest = await fetchAgentDefinition(agent.id);
          if (!latest) throw new Error(t("This agent was removed"));
          return formValuesOf(latest);
        }
      : undefined,
  });

  const set = useCallback(
    <K extends keyof AgentFormValues>(key: K, value: AgentFormValues[K]) =>
      form.setValue(key as never, value as never, { shouldDirty: true, shouldValidate: false }),
    [form],
  );

  const jump = useCallback((id: BlockId) => {
    const container = canvas.current;
    const target = container?.querySelector<HTMLElement>(`#ab-${id}`);
    if (container && target) {
      container.scrollTo({ top: target.offsetTop - 24, behavior: "smooth" });
    }
    setActive(id);
  }, []);

  const onScroll = () => {
    const container = canvas.current;
    if (!container) return;
    let current: BlockId = "who";
    for (const id of BLOCKS) {
      const target = container.querySelector<HTMLElement>(`#ab-${id}`);
      if (target && target.offsetTop - container.scrollTop < 140) current = id;
    }
    if (container.scrollTop + container.clientHeight >= container.scrollHeight - 4) {
      current = agent ? "record" : "team";
    }
    setActive(current);
  };

  const begin = useCallback(
    (from: BuilderStart, description: string) => {
      const base = startValues(from, { timezone, catalog });
      form.reset({ ...base, description: from === "blank" ? "" : description.slice(0, 120) });
      setIntro(false);
    },
    [catalog, form, timezone],
  );

  const draftFrom = useCallback(
    async (description: string) => {
      const { notes, ...drafted } = await drafting.draft(description);
      const base = startValues("blank", { timezone, catalog });
      form.reset({ ...base, ...drafted, cronTimezone: drafted.cronTimezone || timezone });
      if (notes.length) {
        toast(
          notes.length === 1
            ? t("Nova left one thing out: {0}", notes[0]!.reason)
            : t("Nova left {0} things out, such as: {1}", notes.length, notes[0]!.reason),
        );
      }
      setIntro(false);
      setFresh(["who", "instr", "trig", "tools", "limits"]);
      window.setTimeout(() => setFresh([]), FRESH_MS);
    },
    [catalog, drafting, form, t, timezone],
  );

  const tighten = useCallback(async () => {
    setTightening(true);
    try {
      const tightened = await drafting.tighten(values.instructions);
      if (tightened.changed) {
        set("instructions", tightened.instructions);
        toast.success(t("Instructions tightened · undo it from the change list"));
      } else {
        toast(t("Nothing to tighten"));
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("Nova could not tighten them."));
    } finally {
      setTightening(false);
    }
  }, [drafting, set, t, values.instructions]);

  const giveTool = useCallback(
    (name: string) => {
      if (values.toolNames.includes(name)) return;
      const rule = rules.get(name);
      const top = (rule?.promotableTier as AutonomyTier | undefined) ?? values.autonomyCeiling;
      set("toolNames", [...values.toolNames, name]);
      if (catalog.find((tool) => tool.name === name)?.kind === "action") {
        set("toolTiers", { ...values.toolTiers, [name]: top });
      }
    },
    [catalog, rules, set, values.autonomyCeiling, values.toolNames, values.toolTiers],
  );

  const restore = useCallback(
    async (version: number) => {
      if (!agent) return;
      setRestoring(true);
      try {
        const draft = formValuesOf(await fetchAgentVersionDraft(agent.id, version));
        for (const [key, value] of Object.entries(draft)) {
          if (key === "version") continue;
          set(key as keyof AgentFormValues, value as never);
        }
        toast.success(t("Loaded v{0} into the draft · save to restore it", version));
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t("That version could not be loaded"));
      } finally {
        setRestoring(false);
      }
    },
    [agent, set, t],
  );

  const duplicate = useCallback(async () => {
    const copy = await apiService.agentDefinitionService.create(
      toSaveRequest(
        { ...values, name: t("{0} copy", values.name.trim()), shadowMode: true },
        accessToSave(accessOf(values), NEW_AGENT_ACCESS),
      ),
    );
    await refresh();
    toast.success(t("Duplicated as “{0}” · opening it", copy.name));
    onOpened?.(copy.id);
  }, [onOpened, refresh, t, values]);

  const remove = async () => {
    if (!agent) return;
    await apiService.agentDefinitionService.remove(agent.id);
    await refresh();
    toast.success(t("{0} removed", agent.name));
    setRemoving(false);
    onClose();
  };

  const fields: EditFields = {
    name: { label: t("Name") },
    description: { label: t("Description") },
    icon: { label: t("Icon") },
    accent: { label: t("Color") },
    instructions: {
      label: t("Instructions"),
      format: (value) =>
        t("{0} words", (typeof value === "string" ? value : "").split(/\s+/).filter(Boolean).length),
    },
    guardrails: { label: t("Never") },
    triggerMode: { label: t("Starts when") },
    cronExpression: { label: t("Schedule") },
    cronTimezone: { label: t("Time zone") },
    eventKinds: { label: t("Wakes on"), item: (kind) => eventLabel(String(kind)) },
    intervalSeconds: { label: t("Every"), format: (value) => t("{0}s", Number(value)) },
    maxConcurrentRuns: { label: t("At most") },
    endsAt: { label: t("Stop after") },
    accessMode: { label: t("Who can use it") },
    accessRoleIds: {
      label: t("Roles"),
      item: (id) => roles.find((entry) => entry.role.id === id)?.role.name ?? String(id),
    },
    toolNames: {
      label: t("Tools"),
      item: (name) => {
        const tool = catalog.find((entry) => entry.name === name);
        return tool ? toolTitle(tool) : String(name);
      },
    },
    toolTiers: { label: t("Tool freedom"), format: (value) => t("{0} set", Object.keys(value ?? {}).length) },
    toolDailyLimits: {
      label: t("Daily tool limits"),
      format: (value) =>
        t("{0} set", Object.values((value ?? {}) as Record<string, number | null>).filter(Boolean).length),
    },
    autonomyCeiling: { label: t("Ceiling") },
    dataAccessCeiling: { label: t("Data access") },
    decisionTimeoutSeconds: {
      label: t("Proposals expire"),
      format: (value) => t("{0} hours", Math.round(Number(value) / 3600)),
    },
    shadowMode: { label: t("Shadow") },
    simulationMode: { label: t("Simulation") },
    enabled: { label: t("Enabled") },
    monthlyBudgetUsd: {
      label: t("Monthly budget"),
      format: (value) => (typeof value === "number" ? `$${value.toFixed(2)}` : t("No cap")),
    },
    dailyRunLimit: {
      label: t("Runs per day"),
      format: (value) => (Number(value) ? String(value) : t("No cap")),
    },
    runTimeoutSeconds: {
      label: t("Run timeout"),
      format: (value) => t("{0} min", Math.round(Number(value) / 60)),
    },
    maxToolCalls: { label: t("Tool calls per run") },
    preferredProviderId: {
      label: t("Preferred provider"),
      format: (value) =>
        providersQuery.data?.find((provider) => provider.id === value)?.name ?? t("Automatic"),
    },
    outputMode: { label: t("Replies as") },
    learnsFromWork: { label: t("Learns") },
    memoryTokenBudget: {
      label: t("Memory in prompt"),
      format: (value) => (value ? t("{0} tokens", Number(value).toLocaleString()) : t("Default")),
    },
    contextProviders: { label: t("Tell it about") },
    delegateIds: {
      label: t("Hands work to"),
      item: (id) => agentsQuery.data?.find((entry) => entry.id === id)?.name ?? String(id),
    },
  };

  const words = values.instructions.split(/\s+/).filter(Boolean).length;
  const presetSchedule = schedulePreset(values.cronExpression, t);
  const roleNames = roles
    .filter((entry) => values.accessRoleIds.includes(entry.role.id))
    .map((entry) => entry.role.name);
  const triggerSummary =
    values.triggerMode === "Chat"
      ? values.accessMode === "Roles"
        ? values.accessRoleIds.length === 1
          ? t("1 role can ask")
          : t("{0} roles can ask", values.accessRoleIds.length)
        : t("Anyone can ask")
      : values.triggerMode === "Scheduled"
        ? (presetSchedule?.label ?? t("Custom schedule"))
        : values.triggerMode === "Continuous"
          ? t("Every {0} min", Math.max(1, Math.round(values.intervalSeconds / 60)))
          : values.eventKinds.length
            ? values.eventKinds.length === 1
              ? t("1 event")
              : t("{0} events", values.eventKinds.length)
            : t("No events yet");
  const learningOffForAll = controlQuery.data?.learningOff ?? false;
  const entries: RailEntry[] = [
    { id: "who", label: t("Identity"), summary: values.name.trim() || t("Unnamed"), status: status.who },
    {
      id: "instr",
      label: t("Instructions"),
      summary: words
        ? findings.length
          ? t("{0} words · {1} gap", words, findings.length)
          : t("{0} words", words)
        : t("Not written"),
      status: status.instr,
    },
    { id: "trig", label: t("When it runs"), summary: triggerSummary, status: status.trig },
    {
      id: "tools",
      label: t("Tools and autonomy"),
      summary: t("{0} tools · {1} ask first", chosen.length, byTier.ActWithApproval),
      status: status.tools,
    },
    {
      id: "limits",
      label: t("Budget and model"),
      summary: `${
        values.monthlyBudgetUsd === null
          ? t("No cap")
          : t("${0}/mo", values.monthlyBudgetUsd)
      } · ${values.outputMode === "Report" ? t("reports") : t("conversational")}`,
      status: status.limits,
    },
    {
      id: "team",
      label: t("Memory and handoffs"),
      summary:
        (values.learnsFromWork && !learningOffForAll ? t("Learns") : t("Doesn't learn")) +
        (values.delegateIds.length ? ` · ${t("asks {0}", values.delegateIds.length)}` : ""),
      status: status.team,
    },
    ...(agent
      ? [
          {
            id: "record" as const,
            label: t("How it's doing"),
            summary: stat?.approved
              ? t("{0} approved", stat.approved.toLocaleString())
              : t("No decisions yet"),
            status: "todo" as const,
          },
        ]
      : []),
  ];
  const modeLabels: Record<Mode, { label: string; note: string }> = {
    live: { label: t("Live"), note: t("Offers proposals; automatic tools run") },
    shadow: {
      label: t("Shadow"),
      note: t("Runs for real, records proposals instead of offering them"),
    },
    sim: { label: t("Simulation"), note: t("Previews every write, never makes one") },
  };
  const setMode = (next: Mode) => {
    set("shadowMode", next === "shadow");
    set("simulationMode", next === "sim");
    setPop(null);
  };
  const saveLabel = agent
    ? t("Save")
    : mode === "live"
      ? t("Create")
      : mode === "shadow"
        ? t("Create in shadow")
        : t("Create in simulation");
  const providers = (providersQuery.data ?? [])
    .filter((provider) => provider.enabled && provider.tasks.includes("AssistantChat"))
    .map((provider) => ({ id: provider.id, name: provider.name, model: provider.model }));
  const shadow = shadowQuery.data;

  return (
    <Dialog.Root open onOpenChange={(next) => !next && flow.back()}>
      <Dialog.Portal>
        <div className="aic aic-layer">
          <Dialog.Popup
            ref={popup}
            initialFocus={intro ? undefined : popup}
            className={cn("ab", tryOpen && !intro && "ab-try", intro && "intro")}
            aria-label={agent ? t("Edit {0}", agent.name) : t("New agent")}
          >
            <Dialog.Title className="sr-only">
              {agent ? t("Edit {0}", agent.name) : t("New agent")}
            </Dialog.Title>
            <header className={cn("ab-h", flow.confirmingClose && "cf")}>
              <button type="button" className="ab-bk" onClick={flow.tryClose}>
                <Ic n="chevR" s={13} />
                {t("Agents")}
              </button>
              <span className="ab-sl">/</span>
              <Tile agent={{ name: values.name, icon: values.icon, accent: values.accent }} s={22} />
              <b className="ab-n">{values.name.trim() || t("New agent")}</b>
              {!intro && (
                <div className="rel">
                  <button
                    type="button"
                    className={cn("mp", mode)}
                    aria-expanded={pop === "mode"}
                    onClick={() => setPop(pop === "mode" ? null : "mode")}
                  >
                    <i />
                    {modeLabels[mode].label}
                    <Ic n="chevD" s={11} />
                  </button>
                  {pop === "mode" && (
                    <Pop className="mp-m" label={t("Mode")} onClose={() => setPop(null)}>
                      {(["live", "shadow", "sim"] as const).map((key) => (
                        <button
                          key={key}
                          type="button"
                          className={cn("mp-o", mode === key && "on")}
                          onClick={() => setMode(key)}
                        >
                          <span className={cn("mp-d", key)} />
                          <span>
                            <b>{modeLabels[key].label}</b>
                            <em>{modeLabels[key].note}</em>
                          </span>
                          {mode === key && <Ic n="check" s={13} w={2.2} />}
                        </button>
                      ))}
                    </Pop>
                  )}
                </div>
              )}
              {agent?.systemKey && (
                <span className="tg">
                  <Ic n="lock" s={10} />
                  {t("System")}
                </span>
              )}
              {!intro && (
                <label
                  className="ab-en"
                  title={t("Disabled agents can't be talked to, scheduled or started by events")}
                >
                  <Switch
                    on={values.enabled}
                    label={t("Enabled")}
                    onChange={(on) => set("enabled", on)}
                  />
                  <span>{values.enabled ? t("Enabled") : t("Disabled")}</span>
                </label>
              )}
              <span className="sp" />
              {!intro && (
                <>
                  <button
                    type="button"
                    className={cn("ab-tb", tryOpen && "on")}
                    aria-pressed={tryOpen}
                    onClick={() => setTryOpen((open) => !open)}
                  >
                    <Ic n="flask" s={13} />
                    {t("Try it")}
                  </button>
                  {agent && (
                    <div className="rel">
                      <button
                        type="button"
                        className="ib"
                        aria-label={t("More")}
                        aria-haspopup="menu"
                        aria-expanded={pop === "more"}
                        onClick={() => setPop(pop === "more" ? null : "more")}
                      >
                        <Ic n="more" s={15} />
                      </button>
                      {pop === "more" && (
                        <Menu
                          right
                          label={t("More")}
                          onClose={() => setPop(null)}
                          items={[
                            {
                              kind: "item",
                              icon: <Ic n="copy" s={14} />,
                              label: t("Duplicate"),
                              onSelect: () => void duplicate().catch(() => toast.error(t("It could not be duplicated"))),
                            },
                            {
                              kind: "item",
                              icon: <Ic n="download" s={14} />,
                              label: t("Export as JSON"),
                              note: t("Import it into another organization"),
                              onSelect: () =>
                                downloadJsonFile(
                                  `${slugify(values.name || "agent")}.agent.json`,
                                  toSaveRequest(values),
                                ),
                            },
                            ...(onOpenActivity
                              ? [
                                  {
                                    kind: "item" as const,
                                    icon: <Ic n="timeline" s={14} />,
                                    label: t("Runs and proposals"),
                                    onSelect: () => {
                                      onOpenActivity();
                                      flow.tryClose();
                                    },
                                  },
                                ]
                              : []),
                            { kind: "separator" },
                            {
                              kind: "item",
                              icon: <Ic n={agent.systemKey ? "lock" : "trash"} s={14} />,
                              label: agent.systemKey ? t("System agent") : t("Remove agent"),
                              note: agent.systemKey ? t("Started by Trenova; turn it off instead") : undefined,
                              danger: !agent.systemKey,
                              disabled: Boolean(agent.systemKey),
                              onSelect: () => setRemoving(true),
                            },
                          ]}
                        />
                      )}
                    </div>
                  )}
                  <span className="ab-vr" />
                </>
              )}
              <div className="ab-sv">
                <SaveBar
                  flow={{ ...flow, invalid: intro ? null : flow.invalid }}
                  saveLabel={saveLabel}
                  review={<ChangeReview form={form} flow={flow} fields={fields} />}
                />
              </div>
              <button
                type="button"
                className="ib"
                title={t("Close (Esc)")}
                aria-label={t("Close (Esc)")}
                onClick={flow.tryClose}
              >
                <Ic n="x" s={14} />
              </button>
            </header>
            {flow.conflict && (
              <ConflictBar
                conflict={flow.conflict}
                onLoadTheirs={() => void flow.loadTheirs()}
                onKeepMine={flow.keepMine}
              />
            )}
            <FormProvider {...form}>
              {intro ? (
                <CreateIntro
                  draftingAvailable={drafting.available}
                  onDraft={draftFrom}
                  onStart={begin}
                />
              ) : (
                <div className="ab-m">
                  <BuilderRail
                    entries={entries}
                    active={active}
                    test={{
                      status: status.test,
                      summary:
                        status.test === "ok"
                          ? t("Tried with this draft")
                          : status.test === "warn"
                            ? t("Draft changed since")
                            : t("Not tried yet"),
                    }}
                    ready={ready}
                    modeNote={
                      mode === "live"
                        ? t("It's set to go live when you save")
                        : t("It stays in {0} until you switch it", modeLabels[mode].label.toLowerCase())
                    }
                    versions={versionsQuery.data ?? null}
                    currentVersion={agent?.version ?? 0}
                    restoring={restoring}
                    onJump={jump}
                    onTry={() => setTryOpen(true)}
                    onRestore={(version) => void restore(version)}
                  />
                  <div className="ab-c" ref={canvas} onScroll={onScroll}>
                    <div className="ab-in">
                      {shadow && shadow.recorded > 0 && (
                        <div className="srb">
                          <Ic n="eyeOff" s={14} />
                          <div>
                            <b>
                              {t("Going live after {0} shadow proposals", shadow.recorded.toLocaleString())}
                            </b>
                            <span>
                              {[
                                shadow.matchRate !== null && shadow.matchRate !== undefined
                                  ? t("{0}% matched what people did", Math.round(shadow.matchRate * 100))
                                  : t("No one has answered one yet"),
                                shadow.wouldReject === 1
                                  ? t("1 would have been rejected")
                                  : t("{0} would have been rejected", shadow.wouldReject),
                                shadow.wouldFail
                                  ? shadow.wouldFail === 1
                                    ? t("1 would have failed")
                                    : t("{0} would have failed", shadow.wouldFail)
                                  : t("none would have failed"),
                              ].join(" · ")}
                            </span>
                          </div>
                          {onOpenActivity && (
                            <button type="button" className="btn sm" onClick={onOpenActivity}>
                              {t("Read them")}
                            </button>
                          )}
                        </div>
                      )}
                      <IdentityBlock fresh={fresh.includes("who")} />
                      <SummarySentence
                        values={values}
                        reads={reads}
                        byTier={byTier}
                        schedule={presetSchedule?.says ?? null}
                        roleNames={roleNames}
                        eventLabel={eventLabel}
                        onJump={jump}
                        onMode={() => setPop("mode")}
                      />
                      <InstructionsBlock
                        fresh={fresh.includes("instr")}
                        findings={findings}
                        onGiveTool={giveTool}
                        onTighten={drafting.available ? () => void tighten() : undefined}
                        tightening={tightening}
                      />
                      <TriggerBlock
                        fresh={fresh.includes("trig")}
                        events={events}
                        roles={roles}
                        sensitiveTools={sensitiveTools}
                        toolTitle={(name) => {
                          const tool = catalog.find((entry) => entry.name === name);
                          return tool ? toolTitle(tool) : name;
                        }}
                      />
                      <Blk
                        id="tools"
                        fresh={fresh.includes("tools")}
                        title={t("Tools and autonomy")}
                        note={t(
                          "What it can read and change, and how much freedom each change gets.",
                        )}
                      >
                        <ToolBench catalog={catalog} rules={rules} />
                      </Blk>
                      <LimitsBlock
                        fresh={fresh.includes("limits")}
                        providers={providers}
                        budget={budgetQuery.data}
                      />
                      <TeamBlock
                        agents={(agentsQuery.data ?? []).filter((entry) => entry.id !== agent?.id)}
                        learningOffForAll={learningOffForAll}
                      />
                      {agent && (
                        <RecordBlock
                          agentId={agent.id}
                          toolNames={values.toolNames}
                          earnedAutonomy={controlQuery.data?.earnedAutonomy ?? false}
                        />
                      )}
                    </div>
                  </div>
                  {tryOpen && (
                    <TryPanel
                      agentId={agent?.id ?? null}
                      draft={draftRequest}
                      snapshot={snapshot}
                      starter={agent?.starters[0]?.prompt ?? null}
                      available={drafting.available}
                      onTried={setTried}
                      onClose={() => setTryOpen(false)}
                    />
                  )}
                </div>
              )}
            </FormProvider>
          </Dialog.Popup>
        </div>
      </Dialog.Portal>
      {agent && (
        <ConfirmDialog
          open={removing}
          onClose={() => setRemoving(false)}
          title={t("Remove {0}?", agent.name)}
          description={t(
            "Its runs and audit trail are kept. Open proposals are withdrawn and people can no longer pick it in Desk.",
          )}
          typed={agent.name}
          danger
          confirmLabel={t("Remove agent")}
          onConfirm={() => void remove().catch(() => toast.error(t("It could not be removed")))}
        />
      )}
    </Dialog.Root>
  );
}
