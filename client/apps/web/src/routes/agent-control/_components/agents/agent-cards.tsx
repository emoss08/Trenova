import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Card } from "@trenova/shared/components/ui/card";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AgentTemplate, TriggerMode } from "@/types/assistant";
import {
  BoltIcon,
  BotIcon,
  CalendarClockIcon,
  EyeOffIcon,
  InboxIcon,
  LockIcon,
  MessageSquareIcon,
  PencilIcon,
  PlayIcon,
  RepeatIcon,
  Trash2Icon,
  WrenchIcon,
} from "lucide-react";
import { TEMPLATE_ICONS } from "./template-picker";

const TRIGGER_META: Record<
  TriggerMode,
  { label: string; icon: typeof BoltIcon; variant: "info" | "teal" | "orange" | "pink" }
> = {
  Chat: { label: "Chat", icon: MessageSquareIcon, variant: "info" },
  Scheduled: { label: "Scheduled", icon: CalendarClockIcon, variant: "teal" },
  Event: { label: "Event", icon: BoltIcon, variant: "orange" },
  Continuous: { label: "Continuous", icon: RepeatIcon, variant: "pink" },
};

const RUN_TIME_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

type AgentCardProps = {
  agent: AgentDefinitionRow;
  templates: readonly AgentTemplate[];
  canUpdate: boolean;
  canDelete: boolean;
  canRun: boolean;
  isToggling: boolean;
  isRunning: boolean;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onRunNow: () => void;
  onDelete: () => void;
};

export function AgentCard({
  agent,
  templates,
  canUpdate,
  canDelete,
  canRun,
  isToggling,
  isRunning,
  onEdit,
  onToggleEnabled,
  onRunNow,
  onDelete,
}: AgentCardProps) {
  const t = useT();
  const trigger = TRIGGER_META[agent.triggerMode];
  const TriggerIcon = trigger.icon;
  const Icon = agent.template ? (TEMPLATE_ICONS[agent.template] ?? BotIcon) : BotIcon;
  const templateLabel = templates.find((entry) => entry.template === agent.template)?.label;
  const isSystem = agent.systemKey !== "";

  return (
    <Card
      size="sm"
      className={cn(
        "hover:border-primary/40 gap-3 px-3 transition-colors",
        !agent.enabled && "opacity-75",
      )}
    >
      <div className="flex items-start gap-3">
        <span
          className={cn(
            "flex size-10 shrink-0 items-center justify-center rounded-lg",
            agent.enabled
              ? "from-primary bg-gradient-to-br to-violet-500 text-white"
              : "bg-muted text-muted-foreground",
          )}
        >
          <Icon className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <button
              type="button"
              onClick={canUpdate ? onEdit : undefined}
              className={cn("truncate text-sm font-semibold", canUpdate && "hover:underline")}
            >
              {agent.name}
            </button>
            <Badge variant={trigger.variant} className="gap-1">
              <TriggerIcon className="size-3" />
              {t(trigger.label)}
            </Badge>
            {agent.shadowMode && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Badge variant="purple" className="gap-1">
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
            {isSystem && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Badge variant="outline" className="gap-1">
                      <LockIcon className="size-3" />
                      {t("System")}
                    </Badge>
                  }
                />
                <TooltipContent>{t("Started by Trenova itself; cannot be removed")}</TooltipContent>
              </Tooltip>
            )}
          </div>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {templateLabel ?? t("Custom agent")}
          </p>
        </div>
        {canUpdate && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="flex items-center">
                  <Switch
                    size="sm"
                    checked={agent.enabled}
                    disabled={isToggling}
                    onCheckedChange={(checked) => onToggleEnabled(checked)}
                    aria-label={agent.enabled ? t("Disable agent") : t("Enable agent")}
                  />
                </span>
              }
            />
            <TooltipContent>{agent.enabled ? t("Enabled") : t("Disabled")}</TooltipContent>
          </Tooltip>
        )}
      </div>

      {agent.description && (
        <p className="text-muted-foreground line-clamp-2 text-xs">{agent.description}</p>
      )}

      <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px]">
        <span className="inline-flex items-center gap-1">
          <WrenchIcon className="size-3" />
          {agent.toolNames.length === 0
            ? t("Answers only")
            : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
        </span>
        <ScheduleSummary agent={agent} />
      </div>

      <div className="border-border mt-auto flex items-center justify-between gap-2 border-t pt-3">
        <div className="flex min-w-0 items-center gap-2 text-[11px]">
          {agent.pendingProposals > 0 ? (
            <Badge variant="warning" className="gap-1">
              <InboxIcon className="size-3" />
              {t(
                "{0, plural, one {# awaiting decision} other {# awaiting decision}}",
                agent.pendingProposals,
              )}
            </Badge>
          ) : (
            <span className="text-muted-foreground">
              {agent.lastRunAt
                ? t("Last ran {0}", formatUnixInUserTimezone(agent.lastRunAt, RUN_TIME_FORMAT))
                : t("Has not run yet")}
            </span>
          )}
          {agent.openRuns > 0 && (
            <span className="text-muted-foreground">
              {t("{0, plural, one {# run open} other {# runs open}}", agent.openRuns)}
            </span>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {canRun && agent.triggerMode !== "Chat" && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="inline-flex">
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      aria-label={t("Run now")}
                      disabled={!agent.enabled || isRunning}
                      onClick={onRunNow}
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
          {canUpdate && (
            <Button size="icon-xs" variant="ghost" aria-label={t("Edit agent")} onClick={onEdit}>
              <PencilIcon className="size-3.5" />
            </Button>
          )}
          {canDelete && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="inline-flex">
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      aria-label={t("Remove agent")}
                      className="text-muted-foreground hover:text-destructive"
                      disabled={isSystem}
                      onClick={onDelete}
                    />
                  </span>
                }
              >
                <Trash2Icon className="size-3.5" />
              </TooltipTrigger>
              <TooltipContent>
                {isSystem ? t("System agents cannot be removed") : t("Remove agent")}
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>
    </Card>
  );
}

function ScheduleSummary({ agent }: { agent: AgentDefinitionRow }) {
  const t = useT();

  switch (agent.triggerMode) {
    case "Scheduled":
      return (
        <span className="inline-flex items-center gap-1">
          <CalendarClockIcon className="size-3" />
          <span className="font-mono">{agent.cronExpression}</span>
          {agent.cronTimezone && <span>· {agent.cronTimezone}</span>}
          {agent.nextRunAt && agent.enabled && (
            <span>
              · {t("next {0}", formatUnixInUserTimezone(agent.nextRunAt, RUN_TIME_FORMAT))}
            </span>
          )}
        </span>
      );
    case "Continuous":
      return (
        <span className="inline-flex items-center gap-1">
          <RepeatIcon className="size-3" />
          {t("every {0}s", agent.intervalSeconds)}
          {agent.nextRunAt && agent.enabled && (
            <span>
              · {t("next {0}", formatUnixInUserTimezone(agent.nextRunAt, RUN_TIME_FORMAT))}
            </span>
          )}
        </span>
      );
    case "Event":
      return (
        <span className="inline-flex items-center gap-1">
          <BoltIcon className="size-3" />
          {t("{0, plural, one {# event} other {# events}}", agent.eventKinds.length)}
        </span>
      );
    default:
      return null;
  }
}
