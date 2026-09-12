import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { createOrgHoliday, updateOrgHoliday, type OrgHolidayRow } from "@/lib/graphql/org-holiday";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import { formatUtcDate, fromUtcDateOnly, toUtcDateOnly } from "@trenova/shared/lib/holiday";
import {
  ORG_HOLIDAY_KIND_HINTS,
  ORG_HOLIDAY_KIND_LABELS,
  orgHolidayFormSchema,
  orgHolidayKindSchema,
  type OrgHolidayFormValues,
} from "@trenova/shared/types/org-holiday";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const KIND_OPTIONS = orgHolidayKindSchema.options.map((value) => ({
  value,
  label: ORG_HOLIDAY_KIND_LABELS[value],
  color: value === "Blackout" ? "#e11d48" : "#059669",
}));

export type HolidayDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Existing entry to edit; omit to create. */
  entry?: OrgHolidayRow | null;
  /** UTC-midnight date to start a new entry on (from clicking a day). */
  presetDate?: number;
  onSaved?: (entry: OrgHolidayRow) => void;
};

function defaultsFor(entry: OrgHolidayRow | null | undefined, presetDate?: number) {
  return {
    name: entry?.name ?? "",
    holidayDate: entry
      ? fromUtcDateOnly(entry.holidayDate)
      : presetDate
        ? fromUtcDateOnly(presetDate)
        : getTodayDate(),
    kind: entry?.kind ?? ("Holiday" as const),
    recursAnnually: entry?.recursAnnually ?? true,
    description: entry?.description ?? null,
  } satisfies OrgHolidayFormValues;
}

export function HolidayDialog({
  open,
  onOpenChange,
  entry,
  presetDate,
  onSaved,
}: HolidayDialogProps) {
  const t = useT();

  const isEdit = Boolean(entry);
  const form = useForm<OrgHolidayFormValues>({
    resolver: zodResolver(orgHolidayFormSchema) as Resolver<OrgHolidayFormValues>,
    defaultValues: defaultsFor(entry, presetDate),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset(defaultsFor(entry, presetDate));
  }, [open, entry, presetDate, reset]);

  const [kind, recursAnnually, holidayDate] = useWatch({
    control,
    name: ["kind", "recursAnnually", "holidayDate"],
  });
  const storedDate = useMemo(
    () => (holidayDate ? toUtcDateOnly(holidayDate) : null),
    [holidayDate],
  );

  const { mutateAsync, isPending } = useApiMutation<
    OrgHolidayRow,
    OrgHolidayFormValues,
    unknown,
    OrgHolidayFormValues
  >({
    form,
    resourceName: "Holiday",
    mutationFn: (values) => {
      const input = {
        name: values.name,
        holidayDate: toUtcDateOnly(values.holidayDate),
        kind: values.kind,
        recursAnnually: values.recursAnnually,
        description: values.description ?? undefined,
        version: entry?.version,
      };
      return entry ? updateOrgHoliday(entry.id, input) : createOrgHoliday(input);
    },
    onSuccess: (saved) => {
      toast.success(isEdit ? `${saved.name} updated` : `${saved.name} added`, {
        description: saved.recursAnnually
          ? `Every ${formatUtcDate(saved.holidayDate, { year: undefined })}.`
          : formatUtcDate(saved.holidayDate, { weekday: "short" }),
      });
      onSaved?.(saved);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("Edit date") : t("Add a date")}</DialogTitle>
          <DialogDescription>
            {t(
              "Holidays are skipped when a policy counts weekdays only. Blackouts stop time off from being requested on that day.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <InputField<OrgHolidayFormValues>
                  control={control}
                  name="name"
                  label={t("Name")}
                  placeholder={kind === "Blackout" ? "e.g. Peak season freeze" : "e.g. Labor Day"}
                  description={t("How the date is listed on the calendar.")}
                  rules={{ required: true }}
                  maxLength={100}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<OrgHolidayFormValues>
                  control={control}
                  name="holidayDate"
                  label={t("Date")}
                  placeholder={t("e.g. Jul 4")}
                  rules={{ required: true }}
                  description={
                    storedDate
                      ? recursAnnually
                        ? `Every ${formatUtcDate(storedDate, { year: undefined })}`
                        : formatUtcDate(storedDate, { weekday: "short" })
                      : "The day the holiday or blackout falls on."
                  }
                />
              </FormControl>
              <FormControl>
                <SelectField<OrgHolidayFormValues>
                  control={control}
                  name="kind"
                  label={t("Kind")}
                  options={KIND_OPTIONS}
                  placeholder={t("Choose a kind")}
                  rules={{ required: true }}
                  description={ORG_HOLIDAY_KIND_HINTS[kind ?? "Holiday"]}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<OrgHolidayFormValues>
                  control={control}
                  name="recursAnnually"
                  label={t("Repeats every year")}
                  description={t(
                    "On, the date is observed on the same day every year; off for one-off dates such as a single-year freeze.",
                  )}
                  position="left"
                  outlined
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<OrgHolidayFormValues>
                  control={control}
                  name="description"
                  label={t("Note")}
                  placeholder={t("e.g. Office closed; dispatch runs a skeleton crew")}
                  description={t("Optional context shown beside the date on the calendar.")}
                  maxLength={500}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Saving...")}>
                {isEdit ? t("Save changes") : t("Add date")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
