import { UserMultiSelectAutocompleteField } from "@/components/autocomplete-fields";
import { EmailChipsField } from "@/components/fields/email-chips-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  useCreateReportSchedule,
  useDeleteReportSchedule,
  useReportSchedules,
  useUpdateReportSchedule,
} from "@/hooks/use-reports";
import { timezoneChoices } from "@/lib/choices";
import {
  buildCron,
  DEFAULT_CRON_PARTS,
  describeCron,
  formatTimeOfDay,
  ordinalDay,
  parseCron,
  type CronFrequency,
  type CronParts,
} from "@/lib/cron";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { ReportDefinition, ReportSchedule } from "@/lib/graphql/reports";
import { cn } from "@trenova/shared/lib/utils";
import { REPORT_FORMAT_CHOICES } from "@/types/report";
import {
  BellIcon,
  CalendarClockIcon,
  CheckIcon,
  MailIcon,
  PaperclipIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
  TriangleAlertIcon,
} from "lucide-react";
import { m } from "motion/react";
import { useMemo, useState } from "react";
import { useController, useForm, useWatch, type Control } from "react-hook-form";
import { toast } from "sonner";

type ScheduleFormValues = {
  cronExpression: string;
  timezone: string;
  formats: string[];
  emailRecipients: string[];
  emailAttach: boolean;
  notifyUserIds: string[];
  enabled: boolean;
};

const EMPTY_FORM: ScheduleFormValues = {
  cronExpression: "0 8 * * 1",
  timezone: "",
  formats: ["xlsx"],
  emailRecipients: [],
  emailAttach: false,
  notifyUserIds: [],
  enabled: true,
};

type CadenceTab = CronFrequency | "custom";

const FREQUENCY_OPTIONS: { value: CadenceTab; label: string }[] = [
  { value: "daily", label: "every day" },
  { value: "weekly", label: "every week" },
  { value: "monthly", label: "every month" },
  { value: "custom", label: "a custom schedule" },
];

const WEEKDAY_CHIPS: { value: number; label: string }[] = [
  { value: 1, label: "Mon" },
  { value: 2, label: "Tue" },
  { value: 3, label: "Wed" },
  { value: 4, label: "Thu" },
  { value: 5, label: "Fri" },
  { value: 6, label: "Sat" },
  { value: 0, label: "Sun" },
];

const TIME_STEP_MINUTES = 30;

function scheduleToForm(schedule: ReportSchedule): ScheduleFormValues {
  return {
    cronExpression: schedule.cronExpression,
    timezone: schedule.timezone,
    formats: [...schedule.formats],
    emailRecipients: [...schedule.emailRecipients],
    emailAttach: schedule.emailAttach,
    notifyUserIds: [...schedule.notifyUserIds],
    enabled: schedule.enabled,
  };
}

function scheduleToInput(schedule: ReportSchedule) {
  return {
    definitionId: schedule.definitionId,
    cronExpression: schedule.cronExpression,
    timezone: schedule.timezone || undefined,
    formats: [...schedule.formats],
    emailRecipients: [...schedule.emailRecipients],
    emailAttach: schedule.emailAttach,
    notifyUserIds: [...schedule.notifyUserIds],
  };
}

function timezoneLabel(value: string): string {
  if (!value) return "Organization default";
  return timezoneChoices.find((choice) => choice.value === value)?.label ?? value;
}

function useTimezoneOptions(current: string) {
  return useMemo(() => {
    if (!current || timezoneChoices.some((choice) => choice.value === current)) {
      return [...timezoneChoices];
    }
    return [...timezoneChoices, { value: current, label: current }];
  }, [current]);
}

function CadenceSentence({
  cronExpression,
  timezone,
}: {
  cronExpression: string;
  timezone: string;
}) {
  const described = describeCron(cronExpression.trim());

  return (
    <p className="text-2xs text-muted-foreground" aria-live="polite">
      {described ? (
        <>
          Runs <span className="font-medium text-foreground">{described.toLowerCase()}</span>
          {" · "}
          {timezoneLabel(timezone)}
        </>
      ) : cronExpression.trim() ? (
        "Custom cron cadence — the expression is validated when you save."
      ) : (
        "Enter a cron expression or pick a preset."
      )}
    </p>
  );
}

function ToggleChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "inline-flex h-6 items-center gap-1 rounded-md border px-2 text-xs font-medium",
        "transition-[border-color,background-color,color] duration-150",
        "focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
        active
          ? "border-blue-600 bg-blue-600/10 text-blue-600 dark:text-blue-400"
          : "border-input text-muted-foreground hover:border-border hover:bg-muted hover:text-foreground",
      )}
    >
      {active && <CheckIcon className="size-3" />}
      {children}
    </button>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-2xs font-semibold tracking-wide text-muted-foreground uppercase">
      {children}
    </span>
  );
}

const DAY_OF_MONTH_OPTIONS: { value: string; label: string }[] = Array.from(
  { length: 31 },
  (_, i) => ({ value: String(i + 1), label: ordinalDay(i + 1) }),
);

const TIME_OPTIONS: { value: string; label: string }[] = Array.from(
  { length: (24 * 60) / TIME_STEP_MINUTES },
  (_, i) => {
    const total = i * TIME_STEP_MINUTES;
    return {
      value: String(total),
      label: formatTimeOfDay(Math.floor(total / 60), total % 60),
    };
  },
);

const pillTrigger = "h-7 w-auto gap-1 border-transparent bg-transparent px-1.5 font-medium text-foreground hover:bg-muted data-popup-open:bg-muted";

function timeOptions(hour: number, minute: number): { value: string; label: string }[] {
  const current = hour * 60 + minute;
  if (current % TIME_STEP_MINUTES === 0) return TIME_OPTIONS;
  return [
    ...TIME_OPTIONS,
    { value: String(current), label: formatTimeOfDay(hour, minute) },
  ].sort((a, b) => Number(a.value) - Number(b.value));
}

function TimeSelect({
  hour,
  minute,
  onChange,
}: {
  hour: number;
  minute: number;
  onChange: (next: { hour: number; minute: number }) => void;
}) {
  const options = timeOptions(hour, minute);

  return (
    <Select
      items={options}
      value={String(hour * 60 + minute)}
      onValueChange={(value) => {
        const total = Number(value);
        onChange({ hour: Math.floor(total / 60), minute: total % 60 });
      }}
    >
      <SelectTrigger size="sm" className={pillTrigger} aria-label="Time of day">
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="max-h-64" alignItemWithTrigger={false}>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function CadenceBuilder({ control }: { control: Control<ScheduleFormValues> }) {
  const { field } = useController({
    control,
    name: "cronExpression",
    rules: { required: "A cron expression is required" },
  });

  const [parts, setParts] = useState<CronParts>(
    () => parseCron(field.value) ?? DEFAULT_CRON_PARTS,
  );
  const [tab, setTab] = useState<CadenceTab>(
    () => parseCron(field.value)?.frequency ?? "custom",
  );

  const applyParts = (next: CronParts) => {
    setParts(next);
    field.onChange(buildCron(next));
  };

  const handleFrequencyChange = (value: CadenceTab | null) => {
    if (value === null) return;
    setTab(value);
    if (value === "custom") return;
    const base = parseCron(field.value) ?? parts;
    applyParts({ ...base, frequency: value });
  };

  const toggleWeekday = (day: number) => {
    const weekdays = parts.weekdays.includes(day)
      ? parts.weekdays.filter((value) => value !== day)
      : [...parts.weekdays, day];
    if (weekdays.length === 0) return;
    applyParts({ ...parts, weekdays });
  };

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border bg-muted/30 p-3">
      <div className="flex flex-wrap items-center gap-x-1.5 gap-y-2 text-sm">
        <span className="text-muted-foreground">Run</span>
        <Select items={FREQUENCY_OPTIONS} value={tab} onValueChange={handleFrequencyChange}>
          <SelectTrigger size="sm" className={pillTrigger} aria-label="Frequency">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {FREQUENCY_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {tab === "monthly" && (
          <>
            <span className="text-muted-foreground">on the</span>
            <Select
              items={DAY_OF_MONTH_OPTIONS}
              value={String(parts.dayOfMonth)}
              onValueChange={(value) => applyParts({ ...parts, dayOfMonth: Number(value) })}
            >
              <SelectTrigger size="sm" className={pillTrigger} aria-label="Day of month">
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="max-h-64" alignItemWithTrigger={false}>
                {DAY_OF_MONTH_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        )}

        {tab !== "custom" && (
          <>
            <span className="text-muted-foreground">at</span>
            <TimeSelect
              hour={parts.hour}
              minute={parts.minute}
              onChange={({ hour, minute }) => applyParts({ ...parts, hour, minute })}
            />
          </>
        )}
      </div>

      {tab === "weekly" && (
        <m.div
          key="weekly"
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.15, ease: "easeOut" }}
          className="flex flex-wrap gap-1.5"
        >
          {WEEKDAY_CHIPS.map((chip) => (
            <ToggleChip
              key={chip.value}
              active={parts.weekdays.includes(chip.value)}
              onClick={() => toggleWeekday(chip.value)}
            >
              {chip.label}
            </ToggleChip>
          ))}
        </m.div>
      )}

      {tab === "custom" && (
        <m.div
          key="custom"
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.15, ease: "easeOut" }}
        >
          <InputField
            control={control}
            name="cronExpression"
            placeholder="0 8 * * 1"
            description="Five fields: minute, hour, day-of-month, month, day-of-week."
            rules={{ required: "A cron expression is required" }}
          />
        </m.div>
      )}
    </div>
  );
}

function FormatChipsField({ control }: { control: Control<ScheduleFormValues> }) {
  const { field, fieldState } = useController({
    control,
    name: "formats",
    rules: {
      validate: (value) => value.length > 0 || "Pick at least one format",
    },
  });

  return (
    <div className="flex flex-col gap-2.5">
      <SectionLabel>Formats</SectionLabel>
      <div className="flex flex-wrap gap-1.5">
        {REPORT_FORMAT_CHOICES.map((choice) => (
          <ToggleChip
            key={choice.value}
            active={field.value.includes(choice.value)}
            onClick={() =>
              field.onChange(
                field.value.includes(choice.value)
                  ? field.value.filter((format) => format !== choice.value)
                  : [...field.value, choice.value],
              )
            }
          >
            {choice.label}
          </ToggleChip>
        ))}
      </div>
      {fieldState.error && <p className="text-2xs text-destructive">{fieldState.error.message}</p>}
    </div>
  );
}

function ScheduleForm({
  initialValues,
  onSubmit,
  onCancel,
  submitting,
  submitLabel,
}: {
  initialValues: ScheduleFormValues;
  onSubmit: (values: ScheduleFormValues) => void;
  onCancel: () => void;
  submitting: boolean;
  submitLabel: string;
}) {
  const { control, handleSubmit } = useForm<ScheduleFormValues>({
    defaultValues: initialValues,
  });
  const cronExpression = useWatch({ control, name: "cronExpression" });
  const timezone = useWatch({ control, name: "timezone" });
  const emailRecipients = useWatch({ control, name: "emailRecipients" });

  const timezoneOptions = useTimezoneOptions(timezone);

  return (
    <m.form
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="flex flex-col gap-5 rounded-lg border border-border bg-card p-4"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="flex flex-col gap-2.5">
        <SectionLabel>Cadence</SectionLabel>
        <CadenceBuilder control={control} />
        <div className="max-w-xs">
          <SelectField
            control={control}
            name="timezone"
            label="Timezone"
            options={timezoneOptions}
            placeholder="Organization default"
            isClearable
          />
        </div>
        <CadenceSentence cronExpression={cronExpression} timezone={timezone} />
      </div>

      <FormatChipsField control={control} />

      <div className="flex flex-col gap-3">
        <SectionLabel>Delivery</SectionLabel>
        <EmailChipsField
          control={control}
          name="emailRecipients"
          label="Email Recipients"
          placeholder="Add an email and press Enter"
          description="Each recipient gets an email with a link to the report when it completes."
        />
        {emailRecipients.length > 0 && (
          <SwitchField
            control={control}
            name="emailAttach"
            label="Attach the report file"
            description="Attached when within the size limit; larger files are linked instead."
            outlined
          />
        )}
        <UserMultiSelectAutocompleteField<ScheduleFormValues>
          control={control}
          name="notifyUserIds"
          label="In-App Recipients"
          placeholder="Search teammates..."
          description="Teammates get an in-app notification with the download when it completes."
          maxCount={2}
          triggerClassName="h-7 text-xs"
        />
      </div>

      <div className="flex items-center justify-between border-t border-border pt-3">
        <SwitchField control={control} name="enabled" label="Enabled" />
        <div className="flex shrink-0 gap-2">
          <Button type="button" variant="outline" size="sm" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" size="sm" disabled={submitting}>
            {submitting ? "Saving..." : submitLabel}
          </Button>
        </div>
      </div>
    </m.form>
  );
}

function DeliveryFacts({ schedule }: { schedule: ReportSchedule }) {
  const emailCount = schedule.emailRecipients.length;
  const notifyCount = schedule.notifyUserIds.length;

  return (
    <>
      <span className="text-muted-foreground/50">·</span>
      {emailCount === 0 && notifyCount === 0 ? (
        <span>In-app only</span>
      ) : (
        <span className="flex items-center gap-2">
          {emailCount > 0 && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="flex cursor-default items-center gap-1 tabular-nums">
                    <MailIcon className="size-3" />
                    {emailCount}
                    {schedule.emailAttach && <PaperclipIcon className="size-3" />}
                  </span>
                }
              />
              <TooltipContent side="bottom">
                {schedule.emailRecipients.join(", ")}
                {schedule.emailAttach ? " — file attached" : ""}
              </TooltipContent>
            </Tooltip>
          )}
          {notifyCount > 0 && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="flex cursor-default items-center gap-1 tabular-nums">
                    <BellIcon className="size-3" />
                    {notifyCount}
                  </span>
                }
              />
              <TooltipContent side="bottom">
                {notifyCount} in-app recipient{notifyCount === 1 ? "" : "s"}
              </TooltipContent>
            </Tooltip>
          )}
        </span>
      )}
    </>
  );
}

function ScheduleRow({
  schedule,
  index,
  onEdit,
  onDelete,
  onToggleEnabled,
  deleting,
  toggling,
}: {
  schedule: ReportSchedule;
  index: number;
  onEdit: () => void;
  onDelete: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  deleting: boolean;
  toggling: boolean;
}) {
  const cadence = describeCron(schedule.cronExpression);

  return (
    <m.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, delay: Math.min(index, 8) * 0.03, ease: "easeOut" }}
      className={cn(
        "group flex items-center gap-3 rounded-lg border border-border bg-card p-3",
        "transition-[border-color,box-shadow,opacity] duration-200 hover:border-brand/60",
        !schedule.enabled && "opacity-60",
      )}
    >
      <div
        className={cn(
          "flex size-8 shrink-0 items-center justify-center rounded-md",
          schedule.enabled ? "bg-blue-500/10 text-blue-600 dark:text-blue-400" : "bg-muted text-muted-foreground",
        )}
      >
        <CalendarClockIcon className="size-4" strokeWidth={1.75} />
      </div>

      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-center gap-2">
          {cadence ? (
            <span className="truncate text-sm font-medium">{cadence}</span>
          ) : (
            <code className="truncate text-sm">{schedule.cronExpression}</code>
          )}
          {schedule.consecutiveFailures > 0 && (
            <Badge variant="warning" className="h-5 gap-1 text-2xs">
              <TriangleAlertIcon className="size-3" />
              {schedule.consecutiveFailures} failed
            </Badge>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-2xs text-muted-foreground">
          <span>{timezoneLabel(schedule.timezone)}</span>
          <span className="text-muted-foreground/50">·</span>
          <span className="tracking-wide uppercase">
            {schedule.formats.map((format) => format.toUpperCase()).join(" ")}
          </span>
          <DeliveryFacts schedule={schedule} />
          {schedule.enabled && schedule.nextRunAt ? (
            <>
              <span className="text-muted-foreground/50">·</span>
              <span className="flex items-center gap-1">
                Next run <HoverCardTimestamp timestamp={schedule.nextRunAt} />
              </span>
            </>
          ) : null}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1">
        <div className="flex gap-0.5 opacity-0 transition-opacity duration-150 group-focus-within:opacity-100 group-hover:opacity-100">
          <Button variant="ghost" size="icon" onClick={onEdit} aria-label="Edit schedule">
            <PencilIcon className="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            disabled={deleting}
            aria-label="Delete schedule"
          >
            <Trash2Icon className="size-3.5 text-destructive" />
          </Button>
        </div>
        <Switch
          size="sm"
          checked={schedule.enabled}
          disabled={toggling}
          onCheckedChange={onToggleEnabled}
          aria-label={schedule.enabled ? "Disable schedule" : "Enable schedule"}
        />
      </div>
    </m.div>
  );
}

export function ReportSchedulesDialog({
  open,
  onOpenChange,
  definition,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  definition: ReportDefinition | null;
}) {
  const definitionId = definition?.id;
  const { data: schedules, isLoading } = useReportSchedules(definitionId, open);
  const createSchedule = useCreateReportSchedule();
  const updateSchedule = useUpdateReportSchedule();
  const deleteSchedule = useDeleteReportSchedule();

  const [editing, setEditing] = useState<ReportSchedule | "new" | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const closeForm = () => setEditing(null);

  const handleSubmit = (values: ScheduleFormValues) => {
    if (!definitionId || !editing) return;

    const shared = {
      definitionId,
      cronExpression: values.cronExpression.trim(),
      timezone: values.timezone.trim() || undefined,
      formats: values.formats,
      emailRecipients: values.emailRecipients,
      emailAttach: values.emailAttach,
      notifyUserIds: values.notifyUserIds,
      enabled: values.enabled,
    };

    const callbacks = {
      onSuccess: () => {
        toast.success("Schedule saved");
        closeForm();
      },
      onError: (error: unknown) =>
        toast.error(graphQLErrorMessage(error, "Failed to save the schedule")),
    };

    if (editing === "new") {
      createSchedule.mutate(shared, callbacks);
    } else {
      updateSchedule.mutate({ ...shared, id: editing.id, version: editing.version }, callbacks);
    }
  };

  const handleToggleEnabled = (schedule: ReportSchedule, enabled: boolean) => {
    setTogglingId(schedule.id);
    updateSchedule.mutate(
      { ...scheduleToInput(schedule), enabled, id: schedule.id, version: schedule.version },
      {
        onSuccess: () => toast.success(enabled ? "Schedule enabled" : "Schedule disabled"),
        onError: (error) =>
          toast.error(graphQLErrorMessage(error, "Failed to update the schedule")),
        onSettled: () => setTogglingId(null),
      },
    );
  };

  const handleDelete = (schedule: ReportSchedule) => {
    setDeletingId(schedule.id);
    deleteSchedule.mutate(schedule.id, {
      onSuccess: () => toast.success("Schedule deleted"),
      onError: (error) => toast.error(graphQLErrorMessage(error, "Failed to delete the schedule")),
      onSettled: () => setDeletingId(null),
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) closeForm();
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Schedules{definition ? ` — ${definition.name}` : ""}</DialogTitle>
          <DialogDescription>
            Scheduled runs execute with your permissions. Deliver completed reports by email or
            straight to teammates in the app.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          {isLoading && (
            <>
              <Skeleton className="h-14 rounded-lg" />
              <Skeleton className="h-14 rounded-lg" />
            </>
          )}
          {!isLoading && (schedules ?? []).length === 0 && editing === null && (
            <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border py-8">
              <div className="flex size-9 items-center justify-center rounded-md bg-muted">
                <CalendarClockIcon className="size-4.5 text-muted-foreground" strokeWidth={1.75} />
              </div>
              <div className="text-center">
                <p className="text-sm font-medium">No schedules yet</p>
                <p className="text-2xs text-muted-foreground">
                  Run this report automatically and deliver it to your team.
                </p>
              </div>
            </div>
          )}
          {(schedules ?? []).map((schedule, index) =>
            editing !== "new" && editing?.id === schedule.id ? (
              <ScheduleForm
                key={schedule.id}
                initialValues={scheduleToForm(schedule)}
                onSubmit={handleSubmit}
                onCancel={closeForm}
                submitting={updateSchedule.isPending}
                submitLabel="Save Changes"
              />
            ) : (
              <ScheduleRow
                key={schedule.id}
                schedule={schedule}
                index={index}
                deleting={deletingId === schedule.id}
                toggling={togglingId === schedule.id}
                onEdit={() => setEditing(schedule)}
                onDelete={() => handleDelete(schedule)}
                onToggleEnabled={(enabled) => handleToggleEnabled(schedule, enabled)}
              />
            ),
          )}
          {editing === "new" ? (
            <ScheduleForm
              initialValues={EMPTY_FORM}
              onSubmit={handleSubmit}
              onCancel={closeForm}
              submitting={createSchedule.isPending}
              submitLabel="Create Schedule"
            />
          ) : (
            <Button variant="outline" onClick={() => setEditing("new")}>
              <PlusIcon className="size-4" />
              Add Schedule
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
