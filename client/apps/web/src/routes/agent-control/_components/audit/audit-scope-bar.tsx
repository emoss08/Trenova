import { AgentPickerList } from "@/components/assistant/agent-picker";
import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import type { SelectOption } from "@/lib/graphql/select-options";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatRange, fromUserWallClock, toUserWallClock } from "@trenova/shared/lib/date";
import { fieldTriggerClass } from "@trenova/shared/lib/variants/field";
import { cn } from "@trenova/shared/lib/utils";
import { endOfDay, startOfDay } from "date-fns";
import { BotIcon, CalendarIcon, ChevronDownIcon, XIcon } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import type { DateRange } from "react-day-picker";
import {
  AUDIT_RANGE_DAYS,
  type AuditRange,
  type AuditRangeDays,
  type AuditTrailScope,
} from "./audit-model";

const NO_RECENT: readonly string[] = [];
const USER_OPTIONS = { resource: "USER" } as const;

export function auditRangeLabel(t: TranslateFn, range: AuditRange): string {
  if (range.kind === "between") {
    return formatRange(range.from, range.to);
  }
  switch (range.days) {
    case 1:
      return t("Last 24 hours");
    case 7:
      return t("Last 7 days");
    case 30:
      return t("Last 30 days");
    case 90:
      return t("Last 90 days");
  }
}

type AuditScopeBarProps = {
  scope: AuditTrailScope;
  onChange: (scope: AuditTrailScope) => void;
  /** Whether the reader may list the organization's agents to pick one. */
  canPickAgent: boolean;
  /** Controls on the right: the export. */
  actions?: ReactNode;
};

/**
 * What the trail is narrowed to from beside the table: the range, the agent,
 * the person the work was for or who decided it, and whether evaluations are
 * shown. The table's own filters narrow it further.
 */
export function AuditScopeBar({ scope, onChange, canPickAgent, actions }: AuditScopeBarProps) {
  const t = useT();
  const evaluationsId = useId();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <AuditRangePicker range={scope.range} onChange={(range) => onChange({ ...scope, range })} />
      {canPickAgent ? (
        <AuditAgentPicker agent={scope.agent} onChange={(agent) => onChange({ ...scope, agent })} />
      ) : null}
      <div className="w-56">
        <Autocomplete<SelectOption, Record<string, never>>
          link="/users/select-options/"
          graphql={USER_OPTIONS}
          initialLimit={50}
          label={t("Person")}
          placeholder={t("Anyone")}
          value={scope.personId}
          onChange={(value: string | null | undefined) =>
            onChange({ ...scope, personId: value ? String(value) : null })
          }
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => option.label || ""}
          renderOption={(option) => option.label || ""}
          clearable
          triggerClassName="h-8 text-xs"
        />
      </div>
      <label htmlFor={evaluationsId} className="flex items-center gap-2 text-xs">
        <Switch
          id={evaluationsId}
          size="sm"
          checked={scope.includeEvaluations}
          onCheckedChange={(checked) => onChange({ ...scope, includeEvaluations: checked })}
        />
        {t("Include evaluations")}
      </label>
      {actions ? <div className="ml-auto flex items-center gap-2">{actions}</div> : null}
    </div>
  );
}

function AuditRangePicker({
  range,
  onChange,
}: {
  range: AuditRange;
  onChange: (range: AuditRange) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<DateRange | undefined>(undefined);

  const choose = (days: AuditRangeDays) => {
    onChange({ kind: "last", days });
    setOpen(false);
  };

  const apply = () => {
    const from = fromUserWallClock(picked?.from ? startOfDay(picked.from) : undefined);
    const to = fromUserWallClock(picked?.to ? endOfDay(picked.to) : undefined);
    if (from === undefined || to === undefined) {
      return;
    }
    onChange({ kind: "between", from, to });
    setOpen(false);
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) {
          setPicked(
            range.kind === "between"
              ? { from: toUserWallClock(range.from), to: toUserWallClock(range.to) }
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
            aria-label={t("Range")}
            className={cn(fieldTriggerClass, "h-8 justify-start gap-1.5 text-xs font-normal")}
          />
        }
      >
        <CalendarIcon className="size-3.5" />
        {auditRangeLabel(t, range)}
        <ChevronDownIcon className="text-foreground-muted size-3.5" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-0">
        <div className="flex">
          <div className="flex flex-col gap-0.5 border-r p-2">
            {AUDIT_RANGE_DAYS.map((days) => (
              <Button
                key={days}
                type="button"
                size="sm"
                variant="ghost"
                aria-pressed={range.kind === "last" && range.days === days}
                className="justify-start"
                onClick={() => choose(days)}
              >
                {auditRangeLabel(t, { kind: "last", days })}
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

function AuditAgentPicker({
  agent,
  onChange,
}: {
  agent: AuditTrailScope["agent"];
  onChange: (agent: AuditTrailScope["agent"]) => void;
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
          <XIcon className="size-3.5" />
        </Button>
      ) : null}
    </div>
  );
}
