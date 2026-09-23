import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AgentTemplate } from "@/types/assistant";
import {
  EyeOffIcon,
  FlaskConicalIcon,
  ForwardIcon,
  InboxIcon,
  LockIcon,
  PencilIcon,
  PlayIcon,
  Trash2Icon,
  WrenchIcon,
} from "lucide-react";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentShelf } from "./agent-roster";
import { canDelegate, delegatesLine, savedDelegates } from "./delegates";
import { TRIGGER_ICONS, TRIGGER_LABELS, TRIGGER_NOTES } from "./trigger-meta";

const RUN_TIME_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

export type AgentRowActions = {
  canUpdate: boolean;
  canDelete: boolean;
  canRun: boolean;
  isToggling: (agent: AgentDefinitionRow) => boolean;
  isRunning: (agent: AgentDefinitionRow) => boolean;
  onEdit: (agent: AgentDefinitionRow) => void;
  onToggleEnabled: (agent: AgentDefinitionRow, enabled: boolean) => void;
  onRunNow: (agent: AgentDefinitionRow) => void;
  onDelete: (agent: AgentDefinitionRow) => void;
};

/**
 * The roster: one shelf per thing that starts an agent, one ruled row per
 * agent. A row reads left to right the way a person checks on an agent —
 * who it is, whether it is on, what it may do, what it did last, and what
 * is waiting — and the controls sit at the end where every row's are.
 */
export function AgentShelves({
  shelves,
  templates,
  actions,
}: {
  shelves: AgentShelf[];
  templates: readonly AgentTemplate[];
  actions: AgentRowActions;
}) {
  const t = useT();

  return (
    <div className="flex flex-col gap-5">
      {shelves.map((shelf) => {
        const Icon = TRIGGER_ICONS[shelf.trigger];
        return (
          <section key={shelf.trigger} className="flex flex-col gap-2">
            <header className="flex items-baseline gap-2 px-1">
              <Icon className="text-muted-foreground size-3.5 self-center" aria-hidden />
              <h3 className="text-sm font-semibold">{t(TRIGGER_LABELS[shelf.trigger])}</h3>
              <span className="text-muted-foreground text-xs">
                {t(TRIGGER_NOTES[shelf.trigger])}
              </span>
            </header>
            <ul className="bg-card divide-border divide-y overflow-hidden rounded-lg border">
              {shelf.agents.map((agent) => (
                <AgentRow key={agent.id} agent={agent} templates={templates} actions={actions} />
              ))}
            </ul>
          </section>
        );
      })}
    </div>
  );
}

export function AgentRow({
  agent,
  templates,
  actions,
}: {
  agent: AgentDefinitionRow;
  templates: readonly AgentTemplate[];
  actions: AgentRowActions;
}) {
  const t = useT();
  const templateLabel = templates.find((entry) => entry.template === agent.template)?.label;
  const isSystem = agent.systemKey !== "";
  const toggling = actions.isToggling(agent);
  const running = actions.isRunning(agent);

  return (
    <li
      className={cn(
        "hover:bg-surface-hover grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-3 px-3 py-2.5 transition-colors sm:grid-cols-[auto_minmax(0,2fr)_minmax(0,1.6fr)_auto]",
        !agent.enabled && "text-muted-foreground",
      )}
    >
      <AgentTile agent={agent} size="lg" className={cn(!agent.enabled && "opacity-60 grayscale")} />

      <div className="flex min-w-0 flex-col">
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
          <button
            type="button"
            onClick={actions.canUpdate ? () => actions.onEdit(agent) : undefined}
            className={cn(
              "ui-focus-ring truncate rounded-control text-sm font-medium",
              actions.canUpdate && "hover:underline",
              agent.enabled ? "text-foreground" : "text-muted-foreground",
            )}
          >
            {agent.name}
          </button>
          {agent.shadowMode && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Badge variant="neutral" appearance="outline" className="gap-1">
                    <EyeOffIcon className="size-3" />
                    {t("Shadow")}
                  </Badge>
                }
              />
              <TooltipContent>
                {t("Runs, but its proposals are recorded rather than offered")}
              </TooltipContent>
            </Tooltip>
          )}
          {agent.simulationMode && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Badge variant="accent-violet" appearance="outline" className="gap-1">
                    <FlaskConicalIcon className="size-3" />
                    {t("Simulation")}
                  </Badge>
                }
              />
              <TooltipContent>
                {t("Its writes are previewed and recorded, never made")}
              </TooltipContent>
            </Tooltip>
          )}
          {isSystem && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Badge variant="neutral" appearance="outline" className="gap-1">
                    <LockIcon className="size-3" />
                    {t("System")}
                  </Badge>
                }
              />
              <TooltipContent>{t("Started by Trenova itself; cannot be removed")}</TooltipContent>
            </Tooltip>
          )}
        </div>
        <p className="text-muted-foreground truncate text-xs">
          {agent.description || templateLabel || t("Custom agent")}
        </p>
      </div>

      <div className="text-muted-foreground col-span-3 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 pt-1 text-xs sm:col-span-1 sm:pt-0">
        <span className="inline-flex items-center gap-1">
          <WrenchIcon className="size-3" />
          {agent.toolNames.length === 0
            ? t("No task tools")
            : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
        </span>
        <ScheduleSummary agent={agent} />
        <DelegatesSummary agent={agent} />
        {agent.pendingProposals > 0 ? (
          <Badge variant="warning" className="gap-1">
            <InboxIcon className="size-3" />
            {t(
              "{0, plural, one {# awaiting decision} other {# awaiting decision}}",
              agent.pendingProposals,
            )}
          </Badge>
        ) : (
          <span className="truncate">
            {agent.lastRunAt
              ? t("Last ran {0}", formatUnixInUserTimezone(agent.lastRunAt, RUN_TIME_FORMAT))
              : t("Has not run yet")}
          </span>
        )}
        {agent.openRuns > 0 && (
          <span>{t("{0, plural, one {# run open} other {# runs open}}", agent.openRuns)}</span>
        )}
      </div>

      <div className="col-start-3 row-start-1 flex shrink-0 items-center gap-1 sm:col-start-4">
        {actions.canRun && agent.triggerMode !== "Chat" && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="inline-flex">
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    aria-label={t("Run now")}
                    disabled={!agent.enabled || running}
                    onClick={() => actions.onRunNow(agent)}
                  >
                    <PlayIcon className="size-3.5" />
                  </Button>
                </span>
              }
            />
            <TooltipContent>
              {agent.enabled
                ? t("Start a run now without waiting for its trigger")
                : t("Enable the agent before running it")}
            </TooltipContent>
          </Tooltip>
        )}
        {actions.canUpdate && (
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={t("Edit agent")}
            onClick={() => actions.onEdit(agent)}
          >
            <PencilIcon className="size-3.5" />
          </Button>
        )}
        {actions.canDelete && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="inline-flex">
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    aria-label={t("Remove agent")}
                    className="text-muted-foreground hover:text-destructive"
                    disabled={isSystem}
                    onClick={() => actions.onDelete(agent)}
                  >
                    <Trash2Icon className="size-3.5" />
                  </Button>
                </span>
              }
            />
            <TooltipContent>
              {isSystem ? t("System agents cannot be removed") : t("Remove agent")}
            </TooltipContent>
          </Tooltip>
        )}
        {actions.canUpdate && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="ml-1 flex items-center">
                  <Switch
                    size="sm"
                    checked={agent.enabled}
                    disabled={toggling}
                    onCheckedChange={(checked) => actions.onToggleEnabled(agent, checked)}
                    aria-label={agent.enabled ? t("Disable agent") : t("Enable agent")}
                  />
                </span>
              }
            />
            <TooltipContent>{agent.enabled ? t("Enabled") : t("Disabled")}</TooltipContent>
          </Tooltip>
        )}
      </div>
    </li>
  );
}

/**
 * Who the agent may hand work to, in one line: "Can ask Report Builder,
 * Dispatch desk +1". The whole list, with any agent disabled since it was
 * added flagged, is behind a hover.
 */
function DelegatesSummary({ agent }: { agent: AgentDefinitionRow }) {
  const t = useT();
  const delegates = savedDelegates(agent);
  const line = delegatesLine(delegates, t);
  if (!canDelegate(agent.triggerMode) || line === "") {
    return null;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="inline-flex min-w-0 cursor-default items-center gap-1">
            <ForwardIcon className="size-3 shrink-0" />
            <span className="truncate">{line}</span>
          </span>
        }
      />
      <TooltipContent>
        <ul className="flex flex-col gap-0.5">
          {delegates.map((delegate) => (
            <li key={delegate.id}>
              {delegate.enabled ? delegate.name : t("{0} (disabled)", delegate.name)}
            </li>
          ))}
        </ul>
      </TooltipContent>
    </Tooltip>
  );
}

function ScheduleSummary({ agent }: { agent: AgentDefinitionRow }) {
  const t = useT();
  const next =
    agent.nextRunAt && agent.enabled
      ? t("next {0}", formatUnixInUserTimezone(agent.nextRunAt, RUN_TIME_FORMAT))
      : null;

  switch (agent.triggerMode) {
    case "Scheduled":
      return (
        <span className="inline-flex min-w-0 items-center gap-1">
          <span className="truncate font-mono">{agent.cronExpression}</span>
          {agent.cronTimezone && <span className="truncate">· {agent.cronTimezone}</span>}
          {next && <span>· {next}</span>}
        </span>
      );
    case "Continuous":
      return (
        <span className="inline-flex items-center gap-1">
          {t("every {0}s", agent.intervalSeconds)}
          {next && <span>· {next}</span>}
        </span>
      );
    case "Event":
      return (
        <span>{t("{0, plural, one {# event} other {# events}}", agent.eventKinds.length)}</span>
      );
    default:
      return null;
  }
}
