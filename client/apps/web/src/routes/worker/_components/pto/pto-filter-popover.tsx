import { useT } from "@trenova/shared/i18n/use-t";
import {
  FleetCodeAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { getCommonDatePresets } from "@trenova/shared/lib/date";
import { ptoFilterSchema, type PTOFilter, type PTOType } from "@trenova/shared/types/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { FilterIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { ptoTypeOptions } from "./use-pto-filters";

export function PTOFilterPopover({
  defaultValues,
  onSubmit,
  onReset,
}: {
  defaultValues: {
    startDate: number;
    endDate: number;
    type?: PTOType;
    workerId?: string;
    fleetCodeId?: string;
  };
  onSubmit: (data: PTOFilter) => void;
  onReset: () => void;
}) {
  const t = useT();

  const [popoverOpen, setPopoverOpen] = useState(false);
  const datePresets = useMemo(() => getCommonDatePresets(), []);

  const form = useForm<PTOFilter>({
    resolver: zodResolver(ptoFilterSchema),
    defaultValues: {
      type: defaultValues.type || "All",
      startDate: defaultValues.startDate,
      endDate: defaultValues.endDate,
      workerId: defaultValues.workerId,
      fleetCodeId: defaultValues.fleetCodeId,
    },
  });

  const handleSubmit = (data: PTOFilter) => {
    onSubmit({
      ...data,
      type: data.type === "All" ? undefined : (data.type as PTOType | undefined),
    });
    setPopoverOpen(false);
  };

  const handleReset = () => {
    form.reset({
      type: "All",
      startDate: defaultValues.startDate,
      endDate: defaultValues.endDate,
      workerId: undefined,
      fleetCodeId: undefined,
    });

    onReset();
  };

  return (
    <Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" className="h-full">
            <FilterIcon className="size-4" />
            <span className="text-xs">{t("Filter")}</span>
          </Button>
        }
      />
      <PopoverContent align="end" className="dark w-[500px] p-0">
        <div className="flex">
          <div>
            <FormGroup dense cols={2} className="p-2">
              <FormControl className="min-h-[2em]" cols="full">
                <SelectField
                  control={form.control}
                  name="type"
                  label={t("PTO Type")}
                  placeholder={t("Select type")}
                  options={ptoTypeOptions}
                  description={t("Show only this kind of time off.")}
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <WorkerAutocompleteField
                  control={form.control}
                  name="workerId"
                  label={t("Worker")}
                  placeholder={t("Select worker")}
                  description={t("Show only this worker's time off.")}
                  clearable
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <FleetCodeAutocompleteField
                  control={form.control}
                  name="fleetCodeId"
                  label={t("Fleet Code")}
                  placeholder={t("Select fleet code")}
                  description={t("Show only workers in this fleet.")}
                  clearable
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <AutoCompleteDateField
                  control={form.control}
                  name="startDate"
                  label={t("Start Date")}
                  placeholder={t("Start date")}
                  rules={{ required: true }}
                  description={t("The earliest day of time off to include.")}
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <AutoCompleteDateField
                  control={form.control}
                  name="endDate"
                  label={t("End Date")}
                  placeholder={t("End date")}
                  rules={{ required: true }}
                  description={t("The latest day of time off to include.")}
                />
              </FormControl>
            </FormGroup>
            <div className="border-border flex justify-end gap-2 border-t p-2">
              <Button size="sm" variant="outline" onClick={handleReset}>
                {t("Reset")}
              </Button>
              <Button size="sm" onClick={form.handleSubmit(handleSubmit)}>
                {t("Apply")}
              </Button>
            </div>
          </div>
          <div className="flex flex-col border-l p-2">
            <label className="mb-1 text-sm font-medium">{t("Presets")}</label>
            {datePresets.map((preset) => (
              <Button
                key={preset.label}
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  const { startDate, endDate } = preset.getValue();
                  form.setValue("startDate", startDate);
                  form.setValue("endDate", endDate);
                  form.setValue("workerId", undefined);
                  form.setValue("fleetCodeId", undefined);
                  handleSubmit({
                    startDate,
                    endDate,
                    type: form.getValues("type"),
                    workerId: form.getValues("workerId"),
                    fleetCodeId: form.getValues("fleetCodeId"),
                  });
                }}
              >
                {t(preset.label)}
              </Button>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
