import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { BillingControl } from "@/types/billing-control";
import { useFormContext, useWatch } from "react-hook-form";

/** The latest day statements can go out on, so the day falls in every month. */
const MAX_STATEMENT_DAY = 28;

/**
 * Monthly statements: whether every customer with a balance or activity is
 * emailed a statement of the month just closed, and on which day of the
 * month. A customer can be left out from its billing profile.
 */
export function MonthlyStatementsCard() {
  const t = useT();
  const { control } = useFormContext<BillingControl>();
  const enabled = useWatch({ control, name: "monthlyStatementsEnabled" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Monthly statements")}</CardTitle>
        <CardDescription>
          {t(
            "Email each customer a PDF statement of the month just closed: the balance it opened with, every invoice, payment and credit, what it owes, and its open invoices, aged. Customers whose account was settled and quiet all month are left out.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="monthlyStatementsEnabled"
              label={t("Send monthly statements")}
              description={t(
                "Statements go to the invoice recipients on each customer's email profile, from your Billing email profile. Each month is sent once.",
              )}
              position="left"
            />
          </FormControl>
          {enabled ? (
            <FormControl className="max-w-[420px]">
              <NumberField
                control={control}
                name="monthlyStatementDay"
                label={t("Day of the month to send")}
                description={t(
                  "From 1 to 28, so the day falls in every month. Statements are dated the last day of the month before.",
                )}
                min={1}
                max={MAX_STATEMENT_DAY}
                rules={{
                  required: t("Choose the day of the month to send statements on"),
                  min: { value: 1, message: t("Choose a day from 1 to 28") },
                  max: { value: MAX_STATEMENT_DAY, message: t("Choose a day from 1 to 28") },
                }}
              />
            </FormControl>
          ) : null}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
