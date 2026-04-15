import {
  FiscalPeriodAutocompleteField,
  FiscalYearAutocompleteField,
} from "@/components/autocomplete-fields";
import { cn } from "@/lib/utils";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";

type FiscalPeriodSelectorProps = {
  value: string | null;
  onChange: (periodId: string) => void;
  className?: string;
};

type SelectorFormValues = {
  fiscalYearId: string;
  fiscalPeriodId: string;
};

export function FiscalPeriodSelector({ value, onChange, className }: FiscalPeriodSelectorProps) {
  const [selectedYearId, setSelectedYearId] = useState<string | null>(null);

  const form = useForm<SelectorFormValues>({
    values: {
      fiscalYearId: selectedYearId ?? "",
      fiscalPeriodId: value ?? "",
    },
  });

  const periodSearchParams = useMemo(() => {
    if (!selectedYearId) return undefined;
    return {
      fieldFilters: JSON.stringify([
        { field: "fiscalYearId", operator: "eq", value: selectedYearId },
      ]),
    };
  }, [selectedYearId]);

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div className="w-[240px]">
        <FiscalYearAutocompleteField
          control={form.control}
          name="fiscalYearId"
          label="Fiscal Year"
          placeholder="Select fiscal year"
          onOptionChange={(option) => {
            const yearId = option?.id ?? null;
            setSelectedYearId(yearId);
            form.setValue("fiscalPeriodId", "");
          }}
        />
      </div>
      <div className="w-[240px]">
        <FiscalPeriodAutocompleteField
          control={form.control}
          label="Fiscal Period"
          name="fiscalPeriodId"
          placeholder="Select period"
          extraSearchParams={periodSearchParams}
          onOptionChange={(option) => {
            if (option?.id) {
              onChange(option.id);
            }
          }}
        />
      </div>
    </div>
  );
}
