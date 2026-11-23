import { AgentPickerList } from "@/components/assistant/agent-picker";
import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import type { SelectOption } from "@/lib/graphql/select-options";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { fieldTriggerClass } from "@trenova/shared/lib/variants/field";
import { cn } from "@trenova/shared/lib/utils";
import {
  BotIcon,
  CalendarIcon,
  ChevronDownIcon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { format, parse } from "date-fns";
import { useState, type ReactNode } from "react";
import type { DateRange } from "react-day-picker";
import {
  CONVERSATION_ORIGINS,
  TIMEFRAME_PRESETS,
  originLabel,
  timeframeLabel,
  type ConversationScope,
  type ConversationTimeframe,
} from "./conversation-scope";

const NO_RECENT: readonly string[] = [];
const USER_OPTIONS = { resource: "USER" } as const;
const ANY_ORIGIN = "any";
const DAY_FORMAT = "yyyy-MM-dd";

function toDay(value: string): Date {
  return parse(value, DAY_FORMAT, new Date());
}

type ConversationScopeBarProps = {
  scope: ConversationScope;
  onChange: (scope: ConversationScope) => void;
  /** Whether the reader may list the organization's agents to pick one. */
  canPickAgent: boolean;
  /** Controls on the right: the export. */
  actions?: ReactNode;
};

/**
 * What the list of everyone's conversations is narrowed to: the timeframe of
 * their last message, the person, the agent and where they began. The export
 * dialog opens with the same choices.
 */
export function ConversationScopeBar({
  scope,
  onChange,
  canPickAgent,
  actions,
}: ConversationScopeBarProps) {
  const t = useT();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <TimeframePicker
        timeframe={scope.timeframe}
        onChange={(timeframe) => onChange({ ...scope, timeframe })}
      />
      <div className="w-56">
        <PersonPicker
          personId={scope.personId}
          onChange={(personId) => onChange({ ...scope, personId })}
        />
      </div>
      {canPickAgent ? (
        <AgentScopePicker agent={scope.agent} onChange={(agent) => onChange({ ...scope, agent })} />
      ) : null}
      <OriginPicker origin={scope.origin} onChange={(origin) => onChange({ ...scope, origin })} />
      {actions ? <div className="ml-auto flex items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export function OriginPicker({
  origin,
  onChange,
  className,
}: {
  origin: ConversationScope["origin"];
  onChange: (origin: ConversationScope["origin"]) => void;
  className?: string;
}) {
  const t = useT();

  return (
    <Select
      value={origin ?? ANY_ORIGIN}
      onValueChange={(value) =>
        onChange(
          value === ANY_ORIGIN || value === null ? null : (value as ConversationScope["origin"]),
        )
      }
    >
      <SelectTrigger aria-label={t("Where it began")} className={cn("h-8 w-44 text-xs", className)}>
        <SelectValue>{origin ? originLabel(t, origin) : t("Began anywhere")}</SelectValue>
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ANY_ORIGIN}>{t("Began anywhere")}</SelectItem>
        {CONVERSATION_ORIGINS.map((option) => (
          <SelectItem key={option} value={option}>
            {originLabel(t, option)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function PersonPicker({
  personId,
  onChange,
}: {
  personId: string | null;
  onChange: (personId: string | null) => void;
}) {
  const t = useT();

  return (
    <Autocomplete<SelectOption, Record<string, never>>
      graphql={USER_OPTIONS}
      initialLimit={50}
      label={t("Person")}
      placeholder={t("Everyone")}
      value={personId}
      onChange={(value: string | null | undefined) => onChange(value ? String(value) : null)}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      clearable
      triggerClassName="h-8 text-xs"
    />
  );
}

export function TimeframePicker({
  timeframe,
  onChange,
  className,
}: {
  timeframe: ConversationTimeframe;
  onChange: (timeframe: ConversationTimeframe) => void;
  className?: string;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<DateRange | undefined>(undefined);

  const choose = (preset: (typeof TIMEFRAME_PRESETS)[number]) => {
    onChange({ preset });
    setOpen(false);
  };

  const apply = () => {
    if (!picked?.from || !picked?.to) {
      return;
    }
    onChange({
      preset: "Custom",
      fromDate: format(picked.from, DAY_FORMAT),
      toDate: format(picked.to, DAY_FORMAT),
    });
    setOpen(false);
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) {
          setPicked(
            timeframe.preset === "Custom"
              ? { from: toDay(timeframe.fromDate), to: toDay(timeframe.toDate) }
              : undefined,
          );
        }
      }}
    >
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            aria-label={t("Timeframe")}
            className={cn(
              fieldTriggerClass,
              "h-8 justify-start gap-1.5 text-xs font-normal",
              className,
            )}
          />
        }
      >
        <CalendarIcon className="size-3.5" />
        {timeframeLabel(t, timeframe)}
        <ChevronDownIcon className="text-foreground-muted ml-auto size-3.5" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-0">
        <div className="flex">
          <div className="flex flex-col gap-0.5 border-r p-2">
            {TIMEFRAME_PRESETS.map((preset) => (
              <Button
                key={preset}
                type="button"
                size="sm"
                variant="ghost"
                aria-pressed={timeframe.preset === preset}
                className="justify-start"
                onClick={() => choose(preset)}
              >
                {timeframeLabel(t, { preset })}
              </Button>
            ))}
          </div>
          <div className="flex flex-col gap-2 p-2">
            <Calendar
              mode="range"
              numberOfMonths={1}
              selected={picked}
              defaultMonth={picked?.from}
              onSelect={setPicked}
            />
            <div className="flex justify-end gap-2">
              <Button type="button" size="sm" variant="outline" onClick={() => setOpen(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="button"
                size="sm"
                disabled={!picked?.from || !picked?.to}
                onClick={apply}
              >
                {t("Apply")}
              </Button>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function AgentScopePicker({
  agent,
  onChange,
}: {
  agent: ConversationScope["agent"];
  onChange: (agent: ConversationScope["agent"]) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <div className="flex items-center">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              variant="outline"
              size="sm"
              aria-label={t("Agent")}
              className={cn(
                fieldTriggerClass,
                "h-8 max-w-56 justify-start gap-1.5 text-xs font-normal",
                agent && "rounded-r-none",
              )}
            />
          }
        >
          <BotIcon className="size-3.5 shrink-0" />
          <span className="truncate">{agent ? agent.name : t("Every agent")}</span>
          <ChevronDownIcon className="text-foreground-muted size-3.5 shrink-0" />
        </PopoverTrigger>
        <PopoverContent
          align="start"
          side="bottom"
          sideOffset={6}
          className="ui-lift-float w-80 gap-0 overflow-hidden p-0"
        >
          {open ? (
            <AgentPickerList
              selectedId={agent?.id ?? null}
              recentIds={NO_RECENT}
              source="grantable"
              onSelect={(choice) => {
                onChange({ id: choice.id, name: choice.name });
                setOpen(false);
              }}
              emptyMessage={t("No agent matches.")}
            />
          ) : null}
        </PopoverContent>
      </Popover>
      {agent ? (
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-label={t("Show every agent")}
          className={cn(fieldTriggerClass, "h-8 rounded-l-none border-l-0 px-2")}
          onClick={() => onChange(null)}
        >
          <XCloseIcon className="size-3.5" />
        </Button>
      ) : null}
    </div>
  );
}
