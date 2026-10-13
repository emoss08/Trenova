import { useT } from "@trenova/shared/i18n/use-t";
import { SwitchField } from "@/components/fields/switch-field";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { formatDayList, parseDayList } from "@/lib/day-list";
import type { BillingControl } from "@/types/billing-control";
import { useState } from "react";
import { useController, useFormContext, useWatch } from "react-hook-form";

/**
 * Payment reminders: whether customers are emailed about overdue invoices,
 * and on which days past due. Each invoice draws each scheduled reminder
 * once; a customer can be left out from its billing profile.
 */
export function PaymentRemindersCard() {
  const t = useT();
  const { control } = useFormContext<BillingControl>();
  const enabled = useWatch({ control, name: "paymentRemindersEnabled" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Payment reminders")}</CardTitle>
        <CardDescription>
          {t(
            "Email customers about overdue invoices on the days past due you choose. One reminder lists every overdue invoice a customer has; disputed invoices and customers waiting on a promise to pay are left out.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="paymentRemindersEnabled"
              label={t("Send payment reminders")}
              description={t(
                "Reminders go out each morning to the invoice recipients on each customer's email profile, from your Billing email profile.",
              )}
              position="left"
            />
          </FormControl>
          {enabled ? (
            <FormControl className="max-w-[420px]">
              <ReminderDaysField />
            </FormControl>
          ) : null}
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ReminderDaysField() {
  const t = useT();
  const { control } = useFormContext<BillingControl>();
  const { field, fieldState } = useController({
    control,
    name: "paymentReminderDays",
    rules: {
      validate: (value) =>
        (value?.length ?? 0) > 0 || t("Choose at least one day past due to send a reminder on"),
    },
  });
  const [text, setText] = useState(() => formatDayList(field.value));
  const [typo, setTypo] = useState(false);
  const error = typo
    ? t("Enter whole days separated by commas, such as 1, 15, 30")
    : fieldState.error?.message;

  return (
    <div className="flex flex-col gap-1">
      <Label htmlFor="payment-reminder-days">{t("Remind on days past due")}</Label>
      <Input
        id="payment-reminder-days"
        value={text}
        aria-invalid={error ? true : undefined}
        aria-describedby="payment-reminder-days-help"
        placeholder="1, 15, 30"
        onChange={(event) => {
          setText(event.target.value);
          const days = parseDayList(event.target.value);
          setTypo(days === null);
          if (days !== null) {
            field.onChange(days);
          }
        }}
        onBlur={() => {
          field.onBlur();
          const days = parseDayList(text);
          if (days !== null) setText(formatDayList(days));
        }}
      />
      <p
        id="payment-reminder-days-help"
        className={error ? "text-danger-foreground text-xs" : "text-muted-foreground text-xs"}
      >
        {error ?? t("Up to six days, from 1 to 365. The last one is sent as the final reminder.")}
      </p>
    </div>
  );
}
