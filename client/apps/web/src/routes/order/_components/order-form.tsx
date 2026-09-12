import { useT } from "@trenova/shared/i18n/use-t";
import { CustomerAutocompleteField, UserAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { currencyChoices, orderStatusChoices } from "@/lib/choices";
import { fetchOrderDetail } from "@/lib/graphql/order";
import type { Order } from "@trenova/shared/types/order";
import { useQuery } from "@tanstack/react-query";
import { useFormContext, useWatch } from "react-hook-form";
import { OrderChargesSection } from "./order-charges-section";
import { OrderLegsSection } from "./order-legs-section";
import { OrderSummarySection } from "./order-summary-section";

type OrderFormProps = {
  mode: "create" | "edit";
};

export function OrderForm({ mode }: OrderFormProps) {
  const t = useT();

  const { control } = useFormContext<Order>();
  const currencyCode = useWatch({ control, name: "currencyCode" }) || "USD";
  const orderId = useWatch({ control, name: "id" });

  const { data: order } = useQuery({
    queryKey: ["order-detail", orderId],
    queryFn: ({ signal }) => fetchOrderDetail(orderId!, { signal }),
    enabled: mode === "edit" && !!orderId,
  });

  const hasLegs = (order?.legs.length ?? 0) > 0;

  return (
    <div className="flex flex-col gap-4">
      <FormSection
        title={t("General Information")}
        description={t("The customer, ownership, and reference numbers that identify this order")}
      >
        <FormGroup cols={2}>
          {mode === "edit" && (
            <FormControl>
              <InputField
                control={control}
                name="orderNumber"
                label={t("Order Number")}
                placeholder={t("Order Number")}
                description={t("System-generated identifier for this order. Read-only.")}
                readOnly
                disabled
              />
            </FormControl>
          )}
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t("Lifecycle stage, derived automatically from the status of the order's shipment legs.")}
              options={orderStatusChoices}
              isReadOnly
            />
          </FormControl>
          <FormControl>
            <CustomerAutocompleteField<Order>
              control={control}
              rules={{ required: true }}
              name="customerId"
              label={t("Customer")}
              placeholder={t("Select a customer")}
              description={
                hasLegs
                  ? "The customer cannot be changed while the order has legs; detach them first."
                  : "The customer this order is billed to. Every shipment leg must share this customer."
              }
              disabled={hasLegs}
            />
          </FormControl>
          <FormControl>
            <UserAutocompleteField<Order>
              control={control}
              name="ownerId"
              label={t("Owner")}
              placeholder={t("Select an owner")}
              description={t("Team member accountable for coordinating and billing this order.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="poNumber"
              label={t("PO Number")}
              placeholder={t("e.g. PO-10432")}
              description={t("The customer's purchase order number, shown on their invoice.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="bol"
              label={t("BOL")}
              placeholder={t("e.g. BOL-88213")}
              description={t("Bill of lading number associated with the overall order.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Commercial")}
        description={t("The quoted price and currency the order is billed in")}
        className="border-border border-t pt-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="currencyCode"
              label={t("Currency")}
              placeholder={t("Select currency")}
              description={t("Currency used for every monetary amount on this order and its invoices.")}
              options={currencyChoices}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="quotedAmount"
              label={t("Quoted Amount")}
              placeholder="0.00"
              description={t("Price quoted to the customer for the whole order, including expected extra charges.")}
              decimalScale={2}
              thousandSeparator
              sideText={currencyCode}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="baseAmount"
              label={t("Base Amount")}
              placeholder="0.00"
              description={t("Base freight amount before accessorial or other extra charges are applied.")}
              decimalScale={2}
              thousandSeparator
              sideText={currencyCode}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      {mode === "edit" && <OrderSummarySection />}
      {mode === "edit" && <OrderLegsSection />}
      {mode === "edit" && <OrderChargesSection />}
    </div>
  );
}
