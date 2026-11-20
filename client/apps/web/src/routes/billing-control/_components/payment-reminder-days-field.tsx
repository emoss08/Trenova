import { FieldWrapper } from "@/components/fields/field-components";
import { formatDayList, parseDayList } from "@/lib/day-list";
import type { BillingControl } from "@/types/billing-control";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { useController, useFormContext } from "react-hook-form";

const INPUT_ID = "payment-reminder-days";

/**
 * The days past due a reminder goes out on, typed as a list ("1, 15, 30") and held as
 * numbers, earliest first. A typo is said under the field and keeps the last list that read;
 * what the server allows (at most six days, each from 1 to 365) is checked by the schema.
 */
export function PaymentReminderDaysField() {
  const t = useT();
  const { control } = useFormContext<BillingControl>();
  const {
    field: { value, onChange, onBlur, ref },
    fieldState,
  } = useController({ control, name: "paymentReminderDays" });
  const [text, setText] = useState(() => formatDayList(value));
  const [typo, setTypo] = useState(false);
  const error = typo
    ? t("Enter whole days separated by commas, such as 1, 15, 30")
    : fieldState.error?.message;

  return (
    <FieldWrapper
      name="paymentReminderDays"
      label={t("Remind on days past due")}
      description={t("Up to six days, from 1 to 365; the last is sent as the final reminder.")}
      descriptionId={`${INPUT_ID}-description`}
      errorId={`${INPUT_ID}-error`}
      error={error}
      htmlFor={INPUT_ID}
      required
    >
      <Input
        id={INPUT_ID}
        ref={ref}
        value={text}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${INPUT_ID}-error` : `${INPUT_ID}-description`}
        placeholder="1, 15, 30"
        onChange={(event) => {
          setText(event.target.value);
          const days = parseDayList(event.target.value);
          setTypo(days === null);
          if (days !== null) {
            onChange(days);
          }
        }}
        onBlur={() => {
          onBlur();
          const days = parseDayList(text);
          if (days !== null) setText(formatDayList(days));
        }}
      />
    </FieldWrapper>
  );
}
