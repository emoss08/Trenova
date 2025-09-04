/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import { AutoCompleteDateField } from "@/components/fields/date-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { dateToUnixTimestamp } from "@/lib/date";
import { zodResolver } from "@hookform/resolvers/zod";
import { Calendar, ChartColumn, FilterIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import * as z from "zod";

const PTOChart = lazy(() => import("./approved-pto-chart"));
const PTOCalendar = lazy(() => import("./approved-pto-calendar"));

const viewTypeChoices = ["chart", "calendar"] as const;

const filterSchema = z
  .object({
    type: z.string().optional(),
    startDate: z.number().min(1, { error: "Start date is required" }),
    endDate: z.number().min(1, { error: "End date is required" }),
  })
  .refine((data) => data.startDate < data.endDate, {
    message: "Start date must be before end date",
    path: ["endDate"],
  });

type FilterFormData = z.infer<typeof filterSchema>;

const ptoTypeOptions = [
  { value: "All", label: "All Types" },
  { value: "Vacation", label: "Vacation" },
  { value: "Sick", label: "Sick" },
  { value: "Holiday", label: "Holiday" },
  { value: "Bereavement", label: "Bereavement" },
  { value: "Maternity", label: "Maternity" },
  { value: "Paternity", label: "Paternity" },
];

export default function ApprovePTOOverview() {
  const [viewType, setViewType] = useQueryState(
    "viewType",
    parseAsStringLiteral(viewTypeChoices)
      .withOptions({
        shallow: true,
      })
      .withDefault("chart"),
  );

  const now = useMemo(() => new Date(), []);
  const defaultValues = useMemo(
    () => ({
      type: "All",
      startDate: dateToUnixTimestamp(
        new Date(now.getFullYear(), now.getMonth(), 1),
      ),
      endDate: dateToUnixTimestamp(
        new Date(now.getFullYear(), now.getMonth() + 1, 0),
      ),
    }),
    [now],
  );

  const form = useForm<FilterFormData>({
    resolver: zodResolver(filterSchema),
    defaultValues,
  });

  const [chartFilters, setChartFilters] = useState({
    startDate: defaultValues.startDate,
    endDate: defaultValues.endDate,
    type: undefined as FilterFormData["type"],
  });

  const onSubmit = (data: FilterFormData) => {
    setChartFilters({
      startDate: data.startDate,
      endDate: data.endDate,
      type: data.type === "All" ? undefined : data.type,
    });
  };

  useEffect(() => {
    const subscription = form.watch(() => {
      if (form.formState.isValid) {
        form.handleSubmit(onSubmit)();
      }
    });
    return () => subscription.unsubscribe();
  }, [form]);

  return (
    <div className="flex flex-col gap-1 col-span-8 size-full">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium font-table">
          Approved PTO Overview
        </h3>
        <div className="flex items-center gap-2">
          <div className="flex flex-row gap-0.5 items-center p-0.5 bg-sidebar border border-border rounded-md">
            <Button
              variant={viewType === "chart" ? "default" : "ghost"}
              onClick={() => setViewType("chart")}
            >
              <ChartColumn className="size-3.5" />
              <span>Chart</span>
            </Button>
            <Button
              variant={viewType === "calendar" ? "default" : "ghost"}
              onClick={() => setViewType("calendar")}
            >
              <Calendar className="size-3.5" />
              <span>Calendar</span>
            </Button>
          </div>
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline" className="h-full">
                <FilterIcon className="size-4" />
                <span className="text-xs">Filter</span>
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="w-80 p-4">
              <FormGroup cols={1}>
                <FormControl>
                  <SelectField
                    control={form.control}
                    name="type"
                    label="PTO Type"
                    placeholder="Select type"
                    options={ptoTypeOptions}
                  />
                </FormControl>
                <div className="grid grid-cols-2 gap-2">
                  <FormControl>
                    <AutoCompleteDateField
                      control={form.control}
                      name="startDate"
                      label="Start Date"
                      placeholder="Start date"
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <AutoCompleteDateField
                      control={form.control}
                      name="endDate"
                      label="End Date"
                      placeholder="End date"
                      rules={{ required: true }}
                    />
                  </FormControl>
                </div>
                {form.formState.errors.endDate && (
                  <div className="text-xs text-destructive">
                    {form.formState.errors.endDate.message}
                  </div>
                )}
                <div className="flex justify-end gap-2 pt-2 border-t border-border/60">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      form.reset();
                      setChartFilters({
                        startDate: defaultValues.startDate,
                        endDate: defaultValues.endDate,
                        type: undefined,
                      });
                    }}
                  >
                    Reset
                  </Button>
                </div>
              </FormGroup>
            </PopoverContent>
          </Popover>
        </div>
      </div>
      <div className="border border-border rounded-md p-3">
        <LazyComponent>
          {viewType === "chart" ? (
            <PTOChart
              startDate={chartFilters.startDate}
              endDate={chartFilters.endDate}
              type={chartFilters.type}
            />
          ) : (
            <PTOCalendar
              startDate={chartFilters.startDate}
              endDate={chartFilters.endDate}
              type={chartFilters.type}
            />
          )}
        </LazyComponent>
      </div>
    </div>
  );
}
