import { InputField } from "@/components/fields/input-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { deleteTimeEntry } from "@/lib/graphql/timesheet";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { formatHours } from "@trenova/shared/lib/timesheet";
import {
  removeTimeEntryFormSchema,
  type RemoveTimeEntryFormValues,
} from "@trenova/shared/types/timesheet";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type RemovableTimeEntry = {
  id: string;
  clockedInAt: number;
  clockedOutAt: number | null | undefined;
  paidMinutes: number;
};

export type RemoveEntryDialogProps = {
  entry: RemovableTimeEntry | null;
  onOpenChange: (open: boolean) => void;
  onRemoved: () => void;
};

/**
 * Taking a punch off the record. It is a dialog with a reason rather than a
 * confirm: the hours disappear from a wage record, and whoever asks later
 * deserves an answer.
 */
export function RemoveEntryDialog({ entry, onOpenChange, onRemoved }: RemoveEntryDialogProps) {
  const form = useForm<RemoveTimeEntryFormValues>({
    resolver: zodResolver(removeTimeEntryFormSchema) as Resolver<RemoveTimeEntryFormValues>,
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (entry) reset({ reason: "" });
  }, [entry, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    boolean,
    RemoveTimeEntryFormValues,
    unknown,
    RemoveTimeEntryFormValues
  >({
    form,
    resourceName: "Entry",
    mutationFn: (values) => deleteTimeEntry({ id: entry?.id ?? "", reason: values.reason }),
    onSuccess: () => {
      toast.success("Entry removed");
      onRemoved();
      onOpenChange(false);
    },
  });

  const when = entry
    ? formatUnixInUserTimezone(entry.clockedInAt, {
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "2-digit",
      })
    : "";

  return (
    <Dialog open={entry !== null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Remove this entry</DialogTitle>
          <DialogDescription>
            {entry
              ? `${formatHours(entry.paidMinutes)} from ${when} comes off the week. The reason is kept.`
              : ""}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={1}>
              <FormControl cols="full">
                <InputField<RemoveTimeEntryFormValues>
                  control={control}
                  name="reason"
                  label="Why"
                  placeholder="e.g. Duplicate punch"
                  rules={{ required: true }}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Keep it
              </Button>
              <Button type="submit" variant="destructive" isLoading={isPending}>
                Remove
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
