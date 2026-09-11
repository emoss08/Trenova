import { useT } from "@trenova/shared/i18n/use-t";
import type { IftaPeriod } from "@/lib/graphql/ifta-return";
import { iftaYearOptions, periodInclusiveEnd, type IftaPeriodKey } from "@/lib/ifta-return";
import { IftaReturnStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { formatUnixDate } from "@trenova/shared/lib/date";
import type { IftaReturnStatus } from "@trenova/shared/types/fuel-ifta-enums";
import { CalendarClockIcon } from "lucide-react";
import { useMemo } from "react";

type QuarterValue = "1" | "2" | "3" | "4";

const QUARTER_ITEMS: SegmentedControlItem<QuarterValue>[] = [
  { value: "1", label: "Q1" },
  { value: "2", label: "Q2" },
  { value: "3", label: "Q3" },
  { value: "4", label: "Q4" },
];

function isQuarterValue(value: string): value is QuarterValue {
  return value === "1" || value === "2" || value === "3" || value === "4";
}

type QuarterPickerProps = {
  period: IftaPeriodKey;
  onPeriodChange: (period: IftaPeriodKey) => void;
  status: IftaReturnStatus | null;
  amendmentNumber: number;
  /** The quarter's bounds and due date, from the return or from the period query. */
  detail: IftaPeriod | null;
};

export function QuarterPicker({
  period,
  onPeriodChange,
  status,
  amendmentNumber,
  detail,
}: QuarterPickerProps) {
  const t = useT();

  const years = useMemo(
    () => iftaYearOptions(new Date().getFullYear(), period.year),
    [period.year],
  );
  const yearItems = useMemo(
    () => years.map((year) => ({ value: String(year), label: String(year) })),
    [years],
  );

  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Select
          value={String(period.year)}
          items={yearItems}
          onValueChange={(value) => onPeriodChange({ ...period, year: Number(value) })}
        >
          <SelectTrigger aria-label={t("Year")} className="h-7 w-24 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {yearItems.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <SegmentedControl<QuarterValue>
          items={QUARTER_ITEMS}
          value={String(period.quarter) as QuarterValue}
          onValueChange={(value) => {
            if (isQuarterValue(value)) onPeriodChange({ ...period, quarter: Number(value) });
          }}
          aria-label={t("Quarter")}
        />
        {status ? (
          <IftaReturnStatusBadge status={status} />
        ) : (
          <Badge variant="outline">{t("Not generated")}</Badge>
        )}
        {amendmentNumber > 0 ? <Badge variant="purple">{t("Amendment {0}", amendmentNumber)}</Badge> : null}
      </div>
      {detail ? (
        <p className="text-muted-foreground flex items-center gap-1.5 text-xs">
          <CalendarClockIcon className="size-3.5" />
          <span>
            {formatUnixDate(detail.start)} – {formatUnixDate(periodInclusiveEnd(detail.end))}
          </span>
          <span aria-hidden>·</span>
          <span>{t("Due {0}", formatUnixDate(detail.dueDate))}</span>
        </p>
      ) : null}
    </div>
  );
}
