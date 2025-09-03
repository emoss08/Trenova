/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { dateToUnixTimestamp } from "@/lib/date";
import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { FilterIcon } from "lucide-react";
import { lazy, useMemo, useState } from "react";

const PTOChart = lazy(() => import("./approved-pto-chart"));

export default function ApprovePTOOverview() {
  const defaultStart = dateToUnixTimestamp(new Date());

  const [chartType, setChartType] = useState<
    WorkerPTOSchema["type"] | undefined
  >(undefined);

  const defaultChartEnd = useMemo(() => {
    const now = new Date();
    return dateToUnixTimestamp(
      new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59),
    );
  }, []);

  const [chartStartDate, setChartStartDate] = useState<number>(defaultStart);
  const [chartEndDate, setChartEndDate] = useState<number>(defaultChartEnd);

  const toInput = (unix?: number) => {
    if (!unix) return "";
    const d = new Date(unix * 1000);
    const yyyy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, "0");
    const dd = String(d.getDate()).padStart(2, "0");
    return `${yyyy}-${mm}-${dd}`;
  };

  return (
    <div className="flex flex-col gap-1 col-span-8 size-full">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium font-table">
          Approved PTO Overview
        </h3>
        <div className="flex items-center gap-1">
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm">
                <FilterIcon className="size-4" />
                <span className="text-xs">Filter</span>
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="w-72 p-3">
              <div className="grid gap-3">
                <div className="grid gap-1.5">
                  <Label htmlFor="chart-pto-type">Type</Label>
                  <Select
                    value={(chartType as string) ?? ""}
                    onValueChange={(v) =>
                      setChartType(
                        (v || undefined) as WorkerPTOSchema["type"] | undefined,
                      )
                    }
                  >
                    <SelectTrigger id="chart-pto-type">
                      <SelectValue placeholder="All types" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All</SelectItem>
                      <SelectItem value="Vacation">Vacation</SelectItem>
                      <SelectItem value="Sick">Sick</SelectItem>
                      <SelectItem value="Holiday">Holiday</SelectItem>
                      <SelectItem value="Bereavement">Bereavement</SelectItem>
                      <SelectItem value="Maternity">Maternity</SelectItem>
                      <SelectItem value="Paternity">Paternity</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="grid gap-1.5">
                    <Label htmlFor="chart-start-date">Start</Label>
                    <Input
                      id="chart-start-date"
                      type="date"
                      value={toInput(chartStartDate)}
                      onChange={(e) =>
                        setChartStartDate(
                          e.target.value
                            ? dateToUnixTimestamp(
                                new Date(`${e.target.value}T00:00:00`),
                              )
                            : defaultStart,
                        )
                      }
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="chart-end-date">End</Label>
                    <Input
                      id="chart-end-date"
                      type="date"
                      value={toInput(chartEndDate)}
                      onChange={(e) =>
                        setChartEndDate(
                          e.target.value
                            ? dateToUnixTimestamp(
                                new Date(`${e.target.value}T23:59:59`),
                              )
                            : defaultChartEnd,
                        )
                      }
                    />
                  </div>
                </div>
                <div className="flex justify-end gap-2 pt-1 border-t border-border/60">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setChartType(undefined);
                      setChartStartDate(defaultStart);
                      setChartEndDate(defaultChartEnd);
                    }}
                  >
                    Reset
                  </Button>
                </div>
              </div>
            </PopoverContent>
          </Popover>
        </div>
      </div>
      <div className="border border-border rounded-md p-3">
        <LazyComponent>
          <PTOChart
            startDate={chartStartDate}
            endDate={chartEndDate}
            type={chartType}
          />
        </LazyComponent>
      </div>
    </div>
  );
}
