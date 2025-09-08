/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { AutoCompleteDateField } from "@/components/fields/date-field";
import { SelectField } from "@/components/fields/select-field";
import {
  FleetCodeAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { getCommonDatePresets } from "@/lib/date";
import { ptoFilterSchema, PTOFilterSchema } from "@/lib/schemas/worker-schema";
import { useUser } from "@/stores/user-store";
import { PTOType } from "@/types/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { FilterIcon } from "lucide-react";
import { useMemo } from "react";
import { useForm } from "react-hook-form";
import { ptoTypeOptions } from "./use-pto-filters";

interface PTOFilterPopoverProps {
  defaultValues: {
    startDate: number;
    endDate: number;
    type?: PTOType;
    workerId?: string;
    fleetCodeId?: string;
  };
  onSubmit: (data: PTOFilterSchema) => void;
  onReset: () => void;
}

export function PTOFilterPopover({
  defaultValues,
  onSubmit,
  onReset,
}: PTOFilterPopoverProps) {
  const user = useUser();
  const datePresets = useMemo(
    () => getCommonDatePresets(user?.timezone),
    [user?.timezone],
  );

  const form = useForm<PTOFilterSchema>({
    resolver: zodResolver(ptoFilterSchema),
    defaultValues: {
      type: defaultValues.type || "All",
      startDate: defaultValues.startDate,
      endDate: defaultValues.endDate,
      workerId: defaultValues.workerId,
      fleetCodeId: defaultValues.fleetCodeId,
    },
  });

  const handleSubmit = (data: PTOFilterSchema) => {
    onSubmit({
      ...data,
      type:
        data.type === "All" ? undefined : (data.type as PTOType | undefined),
    });
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
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" className="h-full">
          <FilterIcon className="size-4" />
          <span className="text-xs">Filter</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[500px] p-0">
        <div className="flex">
          <div>
            <FormGroup dense cols={2} className="p-2">
              <FormControl className="min-h-[2em]" cols="full">
                <SelectField
                  control={form.control}
                  name="type"
                  label="PTO Type"
                  placeholder="Select type"
                  options={ptoTypeOptions}
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <WorkerAutocompleteField
                  control={form.control}
                  name="workerId"
                  label="Worker"
                  placeholder="Select worker"
                  clearable
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <FleetCodeAutocompleteField
                  control={form.control}
                  name="fleetCodeId"
                  label="Fleet Code"
                  placeholder="Select fleet code"
                  clearable
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <AutoCompleteDateField
                  control={form.control}
                  name="startDate"
                  label="Start Date"
                  placeholder="Start date"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl className="min-h-[2em]">
                <AutoCompleteDateField
                  control={form.control}
                  name="endDate"
                  label="End Date"
                  placeholder="End date"
                  rules={{ required: true }}
                />
              </FormControl>
            </FormGroup>
            <div className="flex justify-end gap-2 border-t border-border p-2">
              <Button size="sm" variant="outline" onClick={handleReset}>
                Reset
              </Button>
              <Button size="sm" onClick={form.handleSubmit(handleSubmit)}>
                Apply
              </Button>
            </div>
          </div>
          <div className="flex flex-col border-l p-2">
            <label className="text-sm font-medium mb-1">Presets</label>
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
                {preset.label}
              </Button>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
