import {
  PayProfileAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assignPayProfileToWorker, fetchPayProfileDetail } from "@/lib/graphql/driver-settlement";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { assignPayProfileFormSchema } from "@trenova/shared/types/driver-pay";
import { useEffect, useRef, type ReactNode } from "react";
import { Controller, useForm, useWatch, type Resolver, type UseFormReturn } from "react-hook-form";
import { z } from "zod";

/** The assignment's fields and, per active component, the rate this driver is paid instead. */
const assignWithOverridesSchema = assignPayProfileFormSchema.extend({
  overrides: z.record(z.string(), z.string()),
});
export type AssignPayProfileValues = z.infer<typeof assignWithOverridesSchema>;

export type AssignPayProfileStart = {
  workerId?: string;
  payProfileId?: string;
  /** The rates this driver is paid now, offered again while the same profile stays chosen. */
  overrides?: Readonly<Record<string, string>>;
};

function startValues(start: AssignPayProfileStart): AssignPayProfileValues {
  return {
    workerId: start.workerId ?? "",
    payProfileId: start.payProfileId ?? "",
    effectiveFrom: getTodayDate(),
    effectiveTo: null,
    splitPercent: 100,
    notes: "",
    overrides: { ...start.overrides },
  };
}

export type AssignPayProfileController = {
  form: UseFormReturn<AssignPayProfileValues>;
  isPending: boolean;
  submit: (event?: React.BaseSyntheticEvent) => Promise<void>;
};

/**
 * The pay assignment form behind its hosts (the pay profile page's dialog, the worker page's work
 * pane): the assignment, the driver's per-component overrides, and the mutation that starts it.
 * A rate left empty pays the profile's own.
 */
export function useAssignPayProfileForm({
  open,
  start,
  onAssigned,
}: {
  open: boolean;
  start: AssignPayProfileStart;
  onAssigned: () => void;
}): AssignPayProfileController {
  const queryClient = useQueryClient();
  const form = useForm<AssignPayProfileValues>({
    resolver: zodResolver(assignWithOverridesSchema) as Resolver<AssignPayProfileValues>,
    defaultValues: startValues(start),
  });
  const { handleSubmit, reset } = form;

  const startKey = `${start.workerId ?? ""}:${start.payProfileId ?? ""}:${JSON.stringify(start.overrides ?? {})}`;
  const loaded = useRef<string | null>(null);
  useEffect(() => {
    if (!open) {
      loaded.current = null;
      return;
    }
    if (loaded.current === startKey) return;
    loaded.current = startKey;
    reset(startValues(start));
    // `start` is read through its key, so a new object with the same contents does not reset.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, reset, startKey]);

  const mutation = useApiMutation({
    form,
    resourceName: "Pay Assignment",
    mutationFn: (values: AssignPayProfileValues) =>
      assignPayProfileToWorker({
        workerId: values.workerId,
        payProfileId: values.payProfileId,
        effectiveFrom: values.effectiveFrom,
        effectiveTo: values.effectiveTo ?? undefined,
        splitPercent: String(values.splitPercent),
        rateOverrides: Object.entries(values.overrides)
          .filter(([, rate]) => rate.trim() !== "" && !Number.isNaN(Number(rate)))
          .map(([componentId, rate]) => ({ componentId, rate: rate.trim() })),
        notes: values.notes || undefined,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["worker-pay"] });
      void queryClient.invalidateQueries({ queryKey: ["pay-profile-list"] });
      void queryClient.invalidateQueries({ queryKey: ["pay-profile-assignments"] });
      onAssigned();
    },
  });

  return {
    form,
    isPending: mutation.isPending,
    submit: handleSubmit((values) => mutation.mutateAsync(values).then(() => undefined)),
  };
}

/** The assignment's own fields; the driver and profile pickers show only when not given. */
export function AssignPayProfileFields({
  form,
  showWorker,
  showProfile,
  columns = 2,
}: {
  form: UseFormReturn<AssignPayProfileValues>;
  showWorker: boolean;
  showProfile: boolean;
  columns?: 1 | 2;
}) {
  const t = useT();
  const { control } = form;
  return (
    <FormGroup cols={columns}>
      {showWorker ? (
        <FormControl cols="full">
          <WorkerAutocompleteField
            control={control}
            name="workerId"
            label={t("Driver")}
            placeholder={t("Select driver")}
            required
            description={t(
              "The driver who will be paid under this profile from the effective date forward.",
            )}
          />
        </FormControl>
      ) : null}
      {showProfile ? (
        <FormControl cols="full">
          <PayProfileAutocompleteField
            control={control}
            name="payProfileId"
            label={t("Pay profile")}
            placeholder={t("Select pay profile")}
            required
            description={t(
              "Profiles are shared templates. Set this driver's own rates below instead of copying a profile.",
            )}
          />
        </FormControl>
      ) : null}
      <FormControl>
        <AutoCompleteDateField
          control={control}
          name="effectiveFrom"
          label={t("Effective from")}
          required
          description={t("Pay for shipments delivered on or after this date uses this assignment.")}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="splitPercent"
          label={t("Split percent")}
          sideText="%"
          description={t("100 for solo drivers; 50 each for an even team split.")}
          required
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="notes"
          label={t("Notes")}
          placeholder={t("e.g. Negotiated rate bump at 1-year anniversary")}
          description={t("Why this assignment or rate was set, kept in the assignment history.")}
        />
      </FormControl>
    </FormGroup>
  );
}

/**
 * One row per active component of the chosen profile: its name, the profile's rate, and the
 * driver's own rate. A filled rate is an override; it replaces the component's rate and any
 * mileage bands for this driver only.
 */
export function PayOverrideFields({
  form,
  open,
  initialProfileId,
  initialOverrides,
  heading,
  className,
}: {
  form: UseFormReturn<AssignPayProfileValues>;
  open: boolean;
  /** The profile the starting overrides belong to; another profile starts with none. */
  initialProfileId?: string;
  initialOverrides?: Readonly<Record<string, string>>;
  /** Words above the rows, shown only when the profile has components to override. */
  heading?: ReactNode;
  className?: string;
}) {
  const t = useT();
  const { control, setValue } = form;
  const profileId = useWatch({ control, name: "payProfileId" });

  const previous = useRef(profileId);
  useEffect(() => {
    if (previous.current === profileId) return;
    previous.current = profileId;
    setValue("overrides", profileId === initialProfileId ? { ...initialOverrides } : {}, {
      shouldDirty: true,
    });
  }, [profileId, initialProfileId, initialOverrides, setValue]);

  const { data: profile } = useQuery({
    queryKey: ["pay-profile-detail", profileId],
    queryFn: ({ signal }) => fetchPayProfileDetail(profileId, { signal }),
    enabled: open && profileId !== "",
  });
  const components = (profile?.components ?? []).filter((component) => component.isActive);
  if (components.length === 0) return null;

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      {heading}
      <div className="flex flex-col divide-y divide-border-subtle rounded-md border border-border-subtle">
        {components.map((component) => {
          const percent = component.method === "PercentOfRevenue";
          return (
            <div
              key={component.id}
              className="grid grid-cols-[minmax(0,1fr)_7.5rem] items-center gap-3 px-2.5 py-2 text-xs"
            >
              <span className="min-w-0">
                <span className="block font-medium">{component.description || component.kind}</span>
                <span className="block text-muted-foreground tabular-nums">
                  {percent
                    ? t("Profile {0}%", Number(component.rate))
                    : t("Profile {0}", Number(component.rate))}
                </span>
              </span>
              <Controller
                control={control}
                name={`overrides.${component.id}`}
                render={({ field }) => (
                  <Input
                    value={field.value ?? ""}
                    onChange={(event) => field.onChange(event.target.value)}
                    onBlur={field.onBlur}
                    placeholder={String(Number(component.rate))}
                    inputMode="decimal"
                    aria-label={t("Rate for {0}", component.description || component.kind)}
                    className={cn(
                      "h-7 text-xs tabular-nums",
                      (field.value ?? "").trim() !== "" && "bg-brand-subtle",
                    )}
                  />
                )}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}
