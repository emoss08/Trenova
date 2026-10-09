import { RoleAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDatePicker } from "@/components/fields/date-field/date-picker";
import { FieldWrapper } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { TimezoneField } from "@/components/fields/timezone-field";
import type { TriggerMode } from "@/types/assistant";
import { intlLocale } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatTimezoneLabel } from "@trenova/shared/lib/timezones";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { Controller, useFormContext } from "react-hook-form";
import { Callout } from "../../edit/callout";
import { aicFieldTrigger, aicMultiFieldTrigger } from "../../edit/field-trigger";
import { Ic, type IcName } from "../../kit/ic";
import type { AgentFormValues } from "../agent-form-schema";
import { Blk, useDraftField } from "./block";
import { schedulePresets } from "./schedule-presets";
import {
  DEFAULT_INTERVAL_SECONDS,
  DEFAULT_SCHEDULE,
  calendarDateOf,
  dateIn,
  endOfDay,
  nextRun,
  pickerDateOf,
  weekStrip,
} from "./builder-model";
import { Button } from "@trenova/shared/components/ui/button";

/** A part of the block set off from the one above it. */
const SECTION = "mt-5 border-t border-border-subtle pt-4";

/** Runs at most this many at once. */
const MAX_CONCURRENT = 10;

const INTERVALS = [60, 300, 900, 3600] as const;

type TriggerBlockProps = {
  fresh: boolean;
  /** Every event that may start an agent, by kind, with its label. */
  events: readonly { kind: string; label: string }[];
  /** While it is open to everyone, the chosen tools that reach outside or restricted data. */
  sensitiveTools: readonly string[];
  toolTitle: (name: string) => string;
};

/** What wakes the agent up, when it stops, and who is allowed to start it. */
export function TriggerBlock({ fresh, events, sensitiveTools, toolTitle }: TriggerBlockProps) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const [trigger, setTrigger] = useDraftField("triggerMode");
  const [cron, setCron] = useDraftField("cronExpression");
  const [timezone, setTimezone] = useDraftField("cronTimezone");
  const [interval, setInterval] = useDraftField("intervalSeconds");
  const [concurrent] = useDraftField("maxConcurrentRuns");
  const [endsAt, setEndsAt] = useDraftField("endsAt");
  const [accessMode, setAccessMode] = useDraftField("accessMode");
  const [roleIds] = useDraftField("accessRoleIds");
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const browserZone = useMemo(() => Intl.DateTimeFormat().resolvedOptions().timeZone, []);
  const zone = timezone || browserZone;

  const triggers: { mode: TriggerMode; icon: IcName; label: string; note: string }[] = [
    { mode: "Chat", icon: "chat", label: t("Someone asks"), note: t("In Desk or the assistant") },
    { mode: "Scheduled", icon: "calendar", label: t("On a schedule"), note: t("On a timetable") },
    { mode: "Event", icon: "bolt", label: t("Something happens"), note: t("On events in Trenova") },
    { mode: "Continuous", icon: "refresh", label: t("Keeps watch"), note: t("Every few minutes") },
  ];
  const presets = schedulePresets(t);

  const intervalLabel: Record<(typeof INTERVALS)[number], string> = {
    60: t("1 min"),
    300: t("5 min"),
    900: t("15 min"),
    3600: t("1 hour"),
  };
  const preset = presets.find((entry) => entry.cron === cron.trim());
  const next = useMemo(
    () => (trigger === "Scheduled" ? nextRun(cron, zone, now) : null),
    [cron, now, trigger, zone],
  );

  const pickTrigger = (mode: TriggerMode) => {
    setTrigger(mode);
    if (mode === "Scheduled" && !cron.trim()) {
      setCron(DEFAULT_SCHEDULE);
      if (!timezone) setTimezone(zone);
    }
    if (mode === "Continuous" && interval < 60) {
      setInterval(DEFAULT_INTERVAL_SECONDS);
    }
  };

  const restricted = accessMode === "Roles";
  const accessLabel = trigger === "Chat" ? t("Who can ask it") : t("Who can run it by hand");
  const endsOn = endsAt ? dateIn(endsAt, zone) : "";

  return (
    <Blk
      id="trig"
      fresh={fresh}
      title={t("When it runs")}
      note={t("What wakes it up, and who's allowed to.")}
    >
      <div className="tg3" role="radiogroup" aria-label={t("Starts when")}>
        {triggers.map((entry) => (
          <button
            key={entry.mode}
            type="button"
            role="radio"
            aria-checked={trigger === entry.mode}
            className={cn("tgc", trigger === entry.mode && "on")}
            onClick={() => pickTrigger(entry.mode)}
          >
            <span className="tgc-i">
              <Ic n={entry.icon} s={16} />
            </span>
            <b>{entry.label}</b>
            <span>{entry.note}</span>
            <i className="tgc-ck">
              <Ic n="check" s={10} w={3} />
            </i>
          </button>
        ))}
      </div>
      {trigger === "Scheduled" && (
        <div className="sch">
          <div className="tks">
            {presets.map((entry) => (
              <button
                key={entry.cron}
                type="button"
                className={cn("tkb", cron.trim() === entry.cron && "on first")}
                aria-pressed={cron.trim() === entry.cron}
                onClick={() => setCron(entry.cron)}
              >
                {entry.label}
              </button>
            ))}
          </div>
          <WeekStrip cron={cron} timezone={zone} now={now} />
          <div className="sch-r">
            <InputField<AgentFormValues>
              control={control}
              name="cronExpression"
              rules={{ required: true }}
              aria-label={t("Schedule")}
              className="w-40"
              inputClassProps={aicFieldTrigger}
            />
            <div className="sch-tz">
              <TimezoneField<AgentFormValues>
                control={control}
                name="cronTimezone"
                placeholder={formatTimezoneLabel(browserZone)}
                triggerClassName={aicFieldTrigger}
              />
            </div>
            <span className="sch-h">
              {preset
                ? next
                  ? t(
                      "{0} · next {1}",
                      preset.says,
                      formatUnixDateTimeShort(next, { timezone: zone }),
                    )
                  : preset.says
                : next
                  ? t("Custom · next {0}", formatUnixDateTimeShort(next, { timezone: zone }))
                  : t("Custom · checked when you save")}
            </span>
          </div>
        </div>
      )}
      {trigger === "Event" && (
        <div className="sch">
          <MultiCheckboxField<AgentFormValues, string>
            control={control}
            name="eventKinds"
            rules={{ required: true }}
            label={t("Wakes on")}
            description={t(
              "Each event starts a run about the record it concerns. Repeats inside five minutes are merged.",
            )}
            options={events.map((event) => ({ value: event.kind, label: event.label }))}
            emptyAs="array"
          />
        </div>
      )}
      {trigger === "Continuous" && (
        <div className="sch">
          <div className="grid grid-cols-1 items-start gap-5 md:grid-cols-[minmax(0,1fr)_12rem]">
            <SegmentedField<AgentFormValues, (typeof INTERVALS)[number]>
              control={control}
              name="intervalSeconds"
              label={t("Every")}
              description={t(
                "About {0} checks a week. Each one counts against its runs per day.",
                Math.round((86_400 / Math.max(60, interval)) * 7).toLocaleString(),
              )}
              options={INTERVALS.map((seconds) => ({
                value: seconds,
                label: intervalLabel[seconds],
              }))}
            />
            <NumberField<AgentFormValues, "maxConcurrentRuns">
              control={control}
              name="maxConcurrentRuns"
              rules={{ required: true }}
              label={t("At most")}
              sideText={concurrent === 1 ? t("run at once") : t("runs at once")}
              min={1}
              max={MAX_CONCURRENT}
              inputClassName={aicFieldTrigger}
            />
          </div>
        </div>
      )}
      {(trigger === "Scheduled" || trigger === "Continuous") && (
        <FieldWrapper
          label={t("Stop after")}
          description={
            endsAt
              ? t("Switches itself off on {0}", endsOn)
              : t("Optional. It switches itself off after this date.")
          }
          className={SECTION}
        >
          <div className="w-45">
            <AutoCompleteDatePicker
              date={pickerDateOf(endsOn)}
              setDate={(date) => setEndsAt(date ? endOfDay(calendarDateOf(date), zone) : null)}
              aria-label={t("Stop after")}
              placeholder={t("No end date")}
              className={aicFieldTrigger}
              clearable
            />
          </div>
        </FieldWrapper>
      )}
      <div className="border-border-subtle mt-5 flex items-center justify-between gap-4 border-t pt-4">
        <div className="flex flex-col">
          <b className="text-base font-medium">{accessLabel}</b>
          <span className="text-muted-foreground text-sm">
            {restricted
              ? roleIds.length === 1
                ? t("1 role")
                : t("{0} roles", roleIds.length)
              : t("Everyone who can use the assistant")}
          </span>
        </div>
        <Controller
          control={control}
          name="accessMode"
          render={({ field: { value, onChange } }) => (
            <SegmentedControl<AgentFormValues["accessMode"]>
              aria-label={accessLabel}
              value={value}
              onValueChange={onChange}
              items={[
                { value: "Everyone", label: t("Everyone") },
                { value: "Roles", label: t("Specific roles") },
              ]}
            />
          )}
        />
      </div>
      {restricted && (
        <div className="mt-3">
          <RoleAutocompleteField<AgentFormValues>
            control={control}
            name="accessRoleIds"
            placeholder={t("Pick the roles that can use it")}
            triggerClassName={aicMultiFieldTrigger}
          />
        </div>
      )}
      {!restricted && sensitiveTools.length > 0 && (
        <Callout
          tone="w"
          action={
            <Button type="button" variant="outline" size="sm" onClick={() => setAccessMode("Roles")}>
              {t("Limit to roles")}
            </Button>
          }
        >
          {sensitiveTools.length === 1
            ? t(
                "Anyone can ask it, and it holds a tool that reaches outside the company or into restricted records — {0}.",
                toolTitle(sensitiveTools[0]!),
              )
            : t(
                "Anyone can ask it, and it holds {0} tools that reach outside the company or into restricted records — {1}.",
                sensitiveTools.length,
                sensitiveTools.slice(0, 2).map(toolTitle).join(", ") +
                  (sensitiveTools.length > 2 ? "…" : ""),
              )}
        </Callout>
      )}
    </Blk>
  );
}

/** The next seven days, one track a day, a dot for each run. */
function WeekStrip({ cron, timezone, now }: { cron: string; timezone: string; now: number }) {
  const t = useT();
  const strip = useMemo(() => weekStrip(cron, timezone, now), [cron, now, timezone]);
  const dayName = useMemo(
    () => new Intl.DateTimeFormat(intlLocale(), { weekday: "short", timeZone: timezone }),
    [timezone],
  );

  if (strip.kind === "unreadable") {
    return <div className="wks nil">{t("Not a schedule Trenova can read yet.")}</div>;
  }
  if (strip.kind === "quiet") {
    return (
      <div className="wks nil">
        <Ic n="calendar" s={13} />
        {t("Nothing runs in the next seven days.")}
      </div>
    );
  }

  return (
    <div className="wks" role="img" aria-label={t("{0} runs in the next 7 days", strip.upcoming)}>
      {strip.days.map((day) => (
        <div key={day.start} className={cn("wks-d", day.today && "today")}>
          <div className="wks-t">
            {day.today && (
              <span className="wks-now" style={{ top: `${strip.nowFraction * 100}%` }} />
            )}
            {day.runs.map((run) => (
              <i
                key={`${run.hour}:${run.minute}`}
                className={cn(run.past && "past", day.runs.length > 6 && "tick")}
                style={{ top: `${((run.hour * 60 + run.minute) / 1440) * 100}%` }}
              />
            ))}
          </div>
          <span>
            {day.today ? t("Today") : dayName.format(new Date((day.start + 43_200) * 1000))}
          </span>
        </div>
      ))}
      <div className="wks-s">
        <b className="mono">{strip.upcoming}</b>
        <span>{t("runs in the next 7 days")}</span>
      </div>
    </div>
  );
}
