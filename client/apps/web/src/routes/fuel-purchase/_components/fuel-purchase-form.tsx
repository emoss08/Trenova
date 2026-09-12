import { useT } from "@trenova/shared/i18n/use-t";
import {
  FuelCardAutocompleteField,
  TractorAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { IftaJurisdictionSelectField } from "@/components/fields/ifta-jurisdiction-select-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { InfoPopover } from "@/components/info-popover";
import { currencyChoices, fuelQuantityUnitChoices, iftaFuelTypeChoices } from "@/lib/choices";
import { computeFuelTotal } from "@/lib/fuel-purchase";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { FuelPurchaseFormValues } from "@trenova/shared/types/fuel-purchase";
import { FileSpreadsheetIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

type FuelPurchaseFormProps = {
  isEdit: boolean;
  imported?: boolean;
  importBatchId?: string | null;
};

function metaString(option: GraphQLSelectOption | null, key: string): string {
  const value = option?.meta?.[key];
  return typeof value === "string" ? value : "";
}

export function FuelPurchaseForm({
  isEdit,
  imported = false,
  importBatchId,
}: FuelPurchaseFormProps) {
  const t = useT();

  const { control, getValues, setValue } = useFormContext<FuelPurchaseFormValues>();
  const quantity = useWatch({ control, name: "quantity" });
  const unitPrice = useWatch({ control, name: "unitPrice" });
  const quantityUnit = useWatch({ control, name: "quantityUnit" });
  const totalAmount = useWatch({ control, name: "totalAmount" });
  const currencyCode = useWatch({ control, name: "currencyCode" });

  const computed = computeFuelTotal(quantity, unitPrice);
  const [overridden, setOverridden] = useState(() => {
    const initial = getValues();
    const initialComputed = computeFuelTotal(initial.quantity, initial.unitPrice);
    return (
      initialComputed !== null &&
      initial.totalAmount.trim() !== "" &&
      initial.totalAmount !== initialComputed
    );
  });
  const previous = useRef({ total: totalAmount, computed });

  useEffect(() => {
    const before = previous.current;
    previous.current = { total: totalAmount, computed };
    if (computed === null) return;
    if (computed !== before.computed) {
      if (!overridden && totalAmount !== computed) {
        setValue("totalAmount", computed, { shouldDirty: true });
      }
      return;
    }
    if (totalAmount !== before.total && totalAmount !== computed) {
      setOverridden(true);
    }
  }, [computed, totalAmount, overridden, setValue]);

  const useComputed = useCallback(() => {
    if (computed === null) return;
    setValue("totalAmount", computed, { shouldDirty: true, shouldValidate: true });
    setOverridden(false);
  }, [computed, setValue]);

  const prefillWorker = useCallback(
    (option: GraphQLSelectOption | null) => {
      const primaryWorkerId = metaString(option, "primaryWorkerId");
      if (!primaryWorkerId) return;
      const current = getValues("workerId");
      if (current && current.trim() !== "") return;
      setValue("workerId", primaryWorkerId, { shouldDirty: true });
    },
    [getValues, setValue],
  );

  const showComputedHint = overridden && computed !== null && computed !== totalAmount;
  const unitLabel = quantityUnit === "Litre" ? "litre" : "gallon";

  return (
    <div className="flex flex-col gap-2">
      {isEdit && imported ? (
        <Alert>
          <FileSpreadsheetIcon className="size-4" />
          <AlertTitle>{t("Imported from a fuel card statement")}</AlertTitle>
          <AlertDescription>
            {t("The card and transaction reference came from the statement and are read-only here, so the same statement cannot be counted twice if it is imported again. {0}", importBatchId ? ` ${t("Batch {0}.", importBatchId)}` : "")}
          </AlertDescription>
        </Alert>
      ) : null}

      <FormSection
        title={t("Unit, driver & place")}
        description={t("Which tractor was fuelled, who was driving, and where the fuel was bought.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <TractorAutocompleteField<FuelPurchaseFormValues>
              control={control}
              name="tractorId"
              label={t("Tractor")}
              rules={{ required: true }}
              placeholder={t("Select a tractor")}
              description={t("The unit the fuel went into. Its miles and this fuel meet on the return.")}
              onOptionChange={prefillWorker}
            />
          </FormControl>
          <FormControl>
            <WorkerAutocompleteField<FuelPurchaseFormValues>
              control={control}
              name="workerId"
              label={t("Driver")}
              placeholder={t("Select a driver")}
              clearable
              description={t("Filled from the tractor's primary driver when left empty.")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateTimeField
              control={control}
              name="purchasedAt"
              label={t("Purchased at")}
              rules={{ required: true }}
              placeholder={t("Date and time of the purchase")}
              description={t("Decides which quarter the purchase falls in. Cannot be in the future.")}
            />
          </FormControl>
          <FormControl>
            <div className="relative">
              <IftaJurisdictionSelectField<FuelPurchaseFormValues>
                control={control}
                name="jurisdictionId"
                label={t("Jurisdiction")}
                rules={{ required: true }}
                placeholder={t("Select a jurisdiction")}
                description={t("Where the pump was. This decides which line of the return gets the tax-paid credit.")}
              />
              <InfoPopover title={t("Jurisdiction vs vendor state")} className="absolute top-0 right-0">
                {t("The jurisdiction is the state or province the pump stands in, which is what the return credits. A chain's billing address or the state on the receipt header can be somewhere else entirely, so read the location line, not the vendor line.")}
              </InfoPopover>
            </div>
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="vendor"
              label={t("Vendor")}
              placeholder={t("e.g. Love's #412")}
              maxLength={200}
              description={t("The truck stop or station, as it appears on the receipt.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="vendorCity"
              label={t("City")}
              placeholder={t("e.g. Amarillo")}
              maxLength={100}
              description={t("Helps a reviewer place the stop when the vendor name is ambiguous.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Fuel")}
        description={t("What was bought and what it cost. The total follows quantity × price until you change it.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="fuelType"
              label={t("Fuel type")}
              options={iftaFuelTypeChoices}
              rules={{ required: true }}
              placeholder={t("Select a fuel type")}
              description={t("DEF, reefer and other non-IFTA products are tracked as spend only.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="quantityUnit"
              label={t("Unit")}
              options={fuelQuantityUnitChoices}
              rules={{ required: true }}
              placeholder={t("Select a unit")}
              description={t("Litres are converted to US gallons for the return.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="quantity"
              valueType="string"
              label={t("Quantity")}
              placeholder="120.500"
              decimalScale={3}
              thousandSeparator
              rules={{ required: true }}
              description={`As printed on the receipt, up to three decimals, in ${unitLabel}s.`}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              decimalScale={4}
              thousandSeparator
              name="unitPrice"
              valueType="string"
              label={`Price per ${unitLabel}`}
              placeholder="3.8990"
              description={t("Up to four decimals. Leave empty when only the total is known.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              decimalScale={4}
              thousandSeparator
              control={control}
              name="totalAmount"
              valueType="string"
              label={t("Total paid")}
              placeholder="469.83"
              rules={{ required: true }}
              description={
                showComputedHint
                  ? `Differs from quantity × price (${computed}).`
                  : "Computed from quantity and price; type over it when the receipt says otherwise."
              }
            />
            {showComputedHint ? (
              <Button
                type="button"
                variant="link"
                size="sm"
                className="h-auto justify-start px-0 py-0 text-xs"
                onClick={useComputed}
              >
                {t("Use computed ({0})", formatCurrency(Number(computed), currencyCode || "USD"))}
              </Button>
            ) : null}
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="currencyCode"
              label={t("Currency")}
              placeholder={t("USD")}
              description={t("Three-letter code. Canadian purchases are usually CAD.")}
              options={currencyChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="odometer"
              label={t("Odometer")}
              placeholder="412113"
              sideText="mi"
              min={0}
              description={t("Reading at the pump, if recorded. Lets fuel be checked against miles run.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Card & tax")}
        description={t("How it was paid for and whether fuel tax was included at the pump.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <FuelCardAutocompleteField<FuelPurchaseFormValues>
              control={control}
              name="fuelCardId"
              label={t("Fuel card")}
              placeholder={t("Select a card")}
              clearable
              disabled={imported}
              description={t("The card the purchase was charged to, when one was used.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="cardLastFour"
              valueType="string"
              label={
                <span className="inline-flex items-center gap-1">
                  {t("Card last four")}
                  <InfoPopover title={t("Why only the last four")}>
                    {t("Statements identify a card by its last four digits, and that is all a purchase needs to be matched back to it. The full card number is never stored.")}
                  </InfoPopover>
                </span>
              }
              placeholder="4821"
              readOnly={imported}
              description={t("From the receipt, when the card is not registered here.")}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="transactionReference"
              label={
                <span className="inline-flex items-center gap-1">
                  {t("Transaction reference")}
                  <InfoPopover title={t("Why record it")}>
                    {t("The reference is unique within the organization. When a statement is imported later, rows whose reference is already on file are marked as already imported rather than recorded twice, so keying it here protects the quarter from double-counting.")}
                  </InfoPopover>
                </span>
              }
              placeholder={t("e.g. EFS-77812")}
              maxLength={100}
              readOnly={imported}
              description={t("The provider's transaction number from the receipt or statement.")}
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="taxPaid"
              label={t("Fuel tax paid at the pump")}
              description={t("Leave on for retail purchases. Turn off for bulk or tax-exempt fuel.")}
              tooltip={t("Bulk or untaxed fuel still counts in the fleet's MPG gallons but earns no tax-paid credit on the return.")}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="notes"
              label={t("Notes")}
              placeholder={t("e.g. Receipt shows two products; reefer fuel entered separately")}
              maxLength={2000}
              description={t("Anything a reviewer of the quarter should know about this purchase.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
