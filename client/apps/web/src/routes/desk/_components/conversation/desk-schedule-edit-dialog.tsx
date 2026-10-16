import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { ConversationSchedule } from "@/types/assistant";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useForm } from "react-hook-form";
import { storeSchedule } from "./desk-schedules";
import { ScheduleCadenceField } from "./schedule-cadence-field";
import {
  MAX_SCHEDULE_PROMPT,
  scheduleEditSchema,
  scheduleEditValues,
  type ScheduleEditValues,
} from "./schedule-edit-form";

/**
 * Changes what a schedule asks, how often and when, from its card in a
 * conversation or its row on the Scheduled page. The time is on the
 * schedule's own clock, the one it was made on. Open while `schedule` is set.
 */
export function DeskScheduleEditDialog({
  schedule,
  onClose,
}: {
  schedule: ConversationSchedule | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={schedule !== null} onOpenChange={(next) => !next && onClose()}>
      {schedule && <EditForm key={schedule.id} schedule={schedule} onClose={onClose} />}
    </Dialog>
  );
}

function EditForm({
  schedule,
  onClose,
}: {
  schedule: ConversationSchedule;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<ScheduleEditValues>({
    resolver: zodResolver(scheduleEditSchema(t)),
    defaultValues: scheduleEditValues(schedule),
  });
  const { control, handleSubmit } = form;

  const mutation = useApiMutation({
    form,
    resourceName: "Schedule",
    mutationFn: (values: ScheduleEditValues) =>
      apiService.assistantService.editSchedule(schedule.id, values),
    onSuccess: (saved) => {
      storeSchedule(queryClient, schedule, saved);
      onClose();
    },
  });

  return (
    <DialogContent size="sm">
      <DialogHeader>
        <DialogTitle>{t("Edit schedule")}</DialogTitle>
        <DialogDescription>{t("Times are on {0} time.", schedule.timezone)}</DialogDescription>
      </DialogHeader>
      <Form
        onSubmit={(event) => {
          event.stopPropagation();
          void handleSubmit((values) => mutation.mutateAsync(values))(event);
        }}
      >
        <FormGroup cols={1} className="pb-4">
          <FormControl>
            <TextareaField
              control={control}
              name="prompt"
              label={t("What to ask")}
              rules={{ required: true }}
              maxLength={MAX_SCHEDULE_PROMPT}
            />
          </FormControl>
          <FormControl>
            <ScheduleCadenceField control={control} />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button type="submit" isLoading={mutation.isPending} loadingText={t("Saving...")}>
            {t("Save")}
          </Button>
        </DialogFooter>
      </Form>
    </DialogContent>
  );
}
