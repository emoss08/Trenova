import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  useCreateReportSchedule,
  useDeleteReportSchedule,
  useReportSchedules,
  useUpdateReportSchedule,
} from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@/lib/graphql";
import type { ReportDefinition, ReportSchedule } from "@/lib/graphql/reports";
import { REPORT_FORMAT_CHOICES } from "@/types/report";
import { PencilIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

type ScheduleFormValues = {
  cronExpression: string;
  timezone: string;
  formats: string[];
  emailRecipients: string;
  enabled: boolean;
};

const EMPTY_FORM: ScheduleFormValues = {
  cronExpression: "0 8 * * 1",
  timezone: "",
  formats: ["xlsx"],
  emailRecipients: "",
  enabled: true,
};

const CRON_PRESETS: { label: string; expression: string }[] = [
  { label: "Daily at 8 AM", expression: "0 8 * * *" },
  { label: "Weekdays at 8 AM", expression: "0 8 * * 1-5" },
  { label: "Mondays at 8 AM", expression: "0 8 * * 1" },
  { label: "First of month at 8 AM", expression: "0 8 1 * *" },
];

function scheduleToForm(schedule: ReportSchedule): ScheduleFormValues {
  return {
    cronExpression: schedule.cronExpression,
    timezone: schedule.timezone,
    formats: [...schedule.formats],
    emailRecipients: schedule.emailRecipients.join(", "),
    enabled: schedule.enabled,
  };
}

function parseRecipients(raw: string): string[] {
  return raw
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);
}

function ScheduleForm({
  values,
  onChange,
  onSubmit,
  onCancel,
  submitting,
  submitLabel,
}: {
  values: ScheduleFormValues;
  onChange: (values: ScheduleFormValues) => void;
  onSubmit: () => void;
  onCancel: () => void;
  submitting: boolean;
  submitLabel: string;
}) {
  const canSubmit = values.cronExpression.trim() !== "" && values.formats.length > 0;

  return (
    <div className="flex flex-col gap-4 rounded-md border border-border p-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="schedule-cron">Cron Expression</Label>
        <Input
          id="schedule-cron"
          value={values.cronExpression}
          onChange={(event) => onChange({ ...values, cronExpression: event.target.value })}
          placeholder="0 8 * * 1"
        />
        <div className="flex flex-wrap gap-1">
          {CRON_PRESETS.map((preset) => (
            <Button
              key={preset.expression}
              variant="outline"
              size="sm"
              className="h-6 text-xs"
              onClick={() => onChange({ ...values, cronExpression: preset.expression })}
            >
              {preset.label}
            </Button>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="schedule-timezone">Timezone</Label>
        <Input
          id="schedule-timezone"
          value={values.timezone}
          onChange={(event) => onChange({ ...values, timezone: event.target.value })}
          placeholder="Organization default"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label>Formats</Label>
        <div className="flex flex-wrap gap-3">
          {REPORT_FORMAT_CHOICES.map((choice) => (
            <label key={choice.value} className="flex items-center gap-1.5 text-sm">
              <Checkbox
                checked={values.formats.includes(choice.value)}
                onCheckedChange={(checked) =>
                  onChange({
                    ...values,
                    formats: checked
                      ? [...values.formats, choice.value]
                      : values.formats.filter((format) => format !== choice.value),
                  })
                }
              />
              {choice.label}
            </label>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="schedule-recipients">Email Recipients</Label>
        <Input
          id="schedule-recipients"
          value={values.emailRecipients}
          onChange={(event) => onChange({ ...values, emailRecipients: event.target.value })}
          placeholder="ops@example.com, billing@example.com"
        />
        <p className="text-xs text-muted-foreground">
          Recipients get a link to the report — never the file itself.
        </p>
      </div>
      <div className="flex items-center justify-between">
        <Label htmlFor="schedule-enabled">Enabled</Label>
        <Switch
          id="schedule-enabled"
          checked={values.enabled}
          onCheckedChange={(enabled) => onChange({ ...values, enabled })}
        />
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button size="sm" onClick={onSubmit} disabled={submitting || !canSubmit}>
          {submitting ? "Saving..." : submitLabel}
        </Button>
      </div>
    </div>
  );
}

function ScheduleRow({
  schedule,
  onEdit,
  onDelete,
  deleting,
}: {
  schedule: ReportSchedule;
  onEdit: () => void;
  onDelete: () => void;
  deleting: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-2 rounded-md border border-border p-3">
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <code className="text-sm">{schedule.cronExpression}</code>
          <Badge variant={schedule.enabled ? "active" : "inactive"}>
            {schedule.enabled ? "Enabled" : "Disabled"}
          </Badge>
          {schedule.consecutiveFailures > 0 && (
            <Badge variant="warning">
              {schedule.consecutiveFailures} consecutive failure
              {schedule.consecutiveFailures === 1 ? "" : "s"}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{schedule.timezone}</span>
          <span>•</span>
          <span>{schedule.formats.map((format) => format.toUpperCase()).join(", ")}</span>
          {schedule.nextRunAt ? (
            <>
              <span>•</span>
              <span className="flex items-center gap-1">
                Next run <HoverCardTimestamp timestamp={schedule.nextRunAt} />
              </span>
            </>
          ) : null}
        </div>
      </div>
      <div className="flex gap-1">
        <Button variant="ghost" size="icon" onClick={onEdit} aria-label="Edit schedule">
          <PencilIcon className="size-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          onClick={onDelete}
          disabled={deleting}
          aria-label="Delete schedule"
        >
          <Trash2Icon className="size-4 text-destructive" />
        </Button>
      </div>
    </div>
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
  const [formValues, setFormValues] = useState<ScheduleFormValues>(EMPTY_FORM);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const closeForm = () => setEditing(null);

  const handleSubmit = () => {
    if (!definitionId) return;

    const shared = {
      definitionId,
      cronExpression: formValues.cronExpression.trim(),
      timezone: formValues.timezone.trim() || undefined,
      formats: formValues.formats,
      emailRecipients: parseRecipients(formValues.emailRecipients),
      enabled: formValues.enabled,
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
    } else if (editing) {
      updateSchedule.mutate({ ...shared, id: editing.id, version: editing.version }, callbacks);
    }
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
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Schedules{definition ? ` — ${definition.name}` : ""}</DialogTitle>
          <DialogDescription>
            Scheduled runs execute with your permissions and deliver an in-app notification with a
            download link.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          {isLoading && <Skeleton className="h-16" />}
          {!isLoading && (schedules ?? []).length === 0 && editing === null && (
            <p className="py-2 text-center text-sm text-muted-foreground">
              No schedules yet for this report.
            </p>
          )}
          {(schedules ?? []).map((schedule) =>
            editing !== "new" && editing?.id === schedule.id ? (
              <ScheduleForm
                key={schedule.id}
                values={formValues}
                onChange={setFormValues}
                onSubmit={handleSubmit}
                onCancel={closeForm}
                submitting={updateSchedule.isPending}
                submitLabel="Save Changes"
              />
            ) : (
              <ScheduleRow
                key={schedule.id}
                schedule={schedule}
                deleting={deletingId === schedule.id}
                onEdit={() => {
                  setFormValues(scheduleToForm(schedule));
                  setEditing(schedule);
                }}
                onDelete={() => handleDelete(schedule)}
              />
            ),
          )}
          {editing === "new" ? (
            <ScheduleForm
              values={formValues}
              onChange={setFormValues}
              onSubmit={handleSubmit}
              onCancel={closeForm}
              submitting={createSchedule.isPending}
              submitLabel="Create Schedule"
            />
          ) : (
            <Button
              variant="outline"
              onClick={() => {
                setFormValues(EMPTY_FORM);
                setEditing("new");
              }}
            >
              <PlusIcon className="size-4" />
              Add Schedule
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
