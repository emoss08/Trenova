import { AutoCompleteDateField } from "@/components/fields/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import {
  fiscalPeriodStatusChoices,
  fiscalPeriodTypeChoices,
} from "@/lib/choices";
import { FiscalPeriodSchema } from "@/lib/schemas/fiscal-period-schema";
import { FiscalYearStatusSchema } from "@/lib/schemas/fiscal-year-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext, useWatch } from "react-hook-form";
import { useAuthStore } from "@/stores/auth-store";
import { fetchData } from "@/services/api-service";

export function FiscalPeriodForm({ isCreate = false }: { isCreate?: boolean }) {
  const { control } = useFormContext<FiscalPeriodSchema>();
  const status = useWatch({ control, name: "status" });

  const isClosed = status === FiscalYearStatusSchema.enum.Closed;
  const isLocked = status === FiscalYearStatusSchema.enum.Locked;

  return (
    <div className="flex flex-col">
      <FormGroup cols={2} className="pb-2 border-b">
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Select status"
            description="Current workflow status"
            options={fiscalPeriodStatusChoices}
            isReadOnly
          />
        </FormControl>

        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="periodType"
            label="Type"
            placeholder="Select type"
            description="Current workflow status"
            options={fiscalPeriodTypeChoices}
          />
        </FormControl>

        <FormControl cols="full">
          <NumberField
            control={control}
            rules={{ required: true }}
            name="periodNumber"
            label="Period"
            placeholder="Enter period number"
            description="Fiscal period number"
            min={1}
            max={12}
          />
        </FormControl>

        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Period 1"
            description="Display name for reports and references"
            maxLength={100}
            readOnly={!isCreate && (isClosed || isLocked)} // Read-only if closed/locked
          />
        </FormControl>
      </FormGroup>

      <FormSection
        title="Date Configuration"
        description="Define the fiscal period and calendar year settings"
        className="py-2 border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              rules={{ required: true }}
              control={control}
              name="startDate"
              label="Start Date"
              placeholder="Select start date"
              description="First day of fiscal period"
            />
          </FormControl>

          <FormControl>
            <AutoCompleteDateField
              rules={{ required: true }}
              control={control}
              name="endDate"
              label="End Date"
              placeholder="Select end date"
              description="Last day of fiscal period"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
