import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  billingQueueTransferModeChoices,
  invoiceDraftCreationModeChoices,
  invoicePostingModeChoices,
  queueAutoApproveModeChoices,
  readyToBillAssignmentModeChoices,
  transferScheduleChoices,
} from "@/lib/choices";
import type { BillingControl } from "@/types/billing-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { useBillingControlPlan } from "../billing-control-tabs";

const CHANGED = { shouldDirty: true, shouldValidate: true } as const;

const REASONS = defineLabels({
  manualDrafts: "Invoices can post on their own only when drafts are created on their own.",
});

/** How shipments move from delivered to invoiced without someone pushing each one along. */
export function AutomationTab() {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<BillingControl>();
  const transferMode = useWatch({ control, name: "billingQueueTransferMode" });
  const draftMode = useWatch({ control, name: "invoiceDraftCreationMode" });
  const approveMode = useWatch({ control, name: "queueAutoApproveMode" });
  // Nova's review is an agent's work that Nova presents: without it on the plan the queue has
  // no review to turn on and nothing to approve on Nova's word, and the server reads a saved
  // "When Nova agrees" as Off, which is what automation then does.
  const { novaReviews } = useBillingControlPlan();

  const transfersOnTheirOwn = transferMode === "AutomaticWhenReady";
  const draftsOnTheirOwn = draftMode === "AutomaticWhenTransferred";

  const postingOptions = invoicePostingModeChoices.map((choice) =>
    choice.value === "AutomaticWhenNoBlockingExceptions" && !draftsOnTheirOwn
      ? { ...choice, disabled: true, description: REASONS.manualDrafts }
      : choice,
  );
  const approveOptions = novaReviews
    ? queueAutoApproveModeChoices
    : queueAutoApproveModeChoices.filter((choice) => choice.value !== "WhenAgentAgrees");

  // Posting on its own needs drafts made on their own, so making drafts by hand again puts
  // posting back to review.
  const keepPostingPossible = () => {
    if (
      getValues("invoiceDraftCreationMode") !== "AutomaticWhenTransferred" &&
      getValues("invoicePostingMode") === "AutomaticWhenNoBlockingExceptions"
    ) {
      setValue("invoicePostingMode", "ManualReviewRequired", CHANGED);
    }
  };

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Ready to bill")}
        description={t(
          "How delivered shipments are marked ready to bill and moved into the billing queue.",
        )}
      >
        <SelectField
          control={control}
          name="readyToBillAssignmentMode"
          label={t("Mark ready to bill")}
          description={t(
            "Whether a shipment that meets its billing requirements is marked on its own or by someone.",
          )}
          options={readyToBillAssignmentModeChoices}
          required
        />
        <SelectField
          control={control}
          name="billingQueueTransferMode"
          label={t("Move to the billing queue")}
          description={t(
            "Whether shipments ready to bill move into the queue on their own or by someone.",
          )}
          options={billingQueueTransferModeChoices}
          required
        />
        <SettingsNested show={transfersOnTheirOwn}>
          <SelectField
            control={control}
            name="billingQueueTransferSchedule"
            label={t("How often")}
            description={t("How often ready shipments are moved into the queue.")}
            options={transferScheduleChoices}
            required={transfersOnTheirOwn}
          />
          <NumberField
            control={control}
            name="billingQueueTransferBatchSize"
            label={t("Shipments per run")}
            description={t("The most shipments moved into the queue at a time.")}
            min={1}
            required={transfersOnTheirOwn}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Drafts and posting")}
        description={t(
          "Whether invoices are drafted and posted for you once items reach the queue.",
        )}
      >
        <SelectField
          control={control}
          name="invoiceDraftCreationMode"
          label={t("Create invoice drafts")}
          description={t(
            "Whether a draft is created as each item reaches the queue, or by someone.",
          )}
          options={invoiceDraftCreationModeChoices}
          onValueChange={keepPostingPossible}
          required
        />
        <SettingsNested show={draftsOnTheirOwn}>
          <NumberField
            control={control}
            name="autoInvoiceBatchSize"
            label={t("Drafts per run")}
            description={t("The most invoice drafts created at a time.")}
            min={1}
            required={draftsOnTheirOwn}
          />
          <SwitchField
            control={control}
            name="notifyOnAutoInvoiceCreation"
            label={t("Notify when drafts are created")}
            description={t("Sends a notification each time drafts are created for you.")}
          />
        </SettingsNested>
        <SelectField
          control={control}
          name="invoicePostingMode"
          label={t("Post invoices")}
          description={t(
            "Whether every invoice waits for review, or posts on its own when nothing blocks it.",
          )}
          options={postingOptions}
          required
        />
      </SettingsSection>

      <SettingsSection
        title={t("Queue approval")}
        description={t(
          "How the queue reviews items before a biller sees them, and whether it approves the clean ones on its own.",
        )}
      >
        {novaReviews ? (
          <SwitchField
            control={control}
            name="queueAgentReviewEnabled"
            label={t("Nova reviews each item")}
            description={t(
              "Nova compares each item's charges with its contract, paperwork and the customer's recent invoices and writes the queue's brief; it only advises, and its cost shows under Billing queue review in AI usage.",
            )}
          />
        ) : null}
        <SelectField
          control={control}
          name="queueAutoApproveMode"
          label={t("Approve on its own")}
          description={
            approveMode === "WhenAgentAgrees"
              ? t(
                  "An item is approved without a biller once all five checks clear and Nova agrees with the charges.",
                )
              : approveMode === "WhenChecksClear"
                ? t("An item is approved without a biller once all five checks clear.")
                : t("Every item waits for a biller to approve it.")
          }
          options={approveOptions}
          required
        />
        <SettingsNested show={approveMode !== "Off"}>
          <NumberField
            control={control}
            name="queueAutoApproveMaxAmount"
            label={t("Only up to this amount")}
            description={t("Larger items wait for a biller. Leave empty for no limit.")}
            decimalScale={2}
            fixedDecimalScale
            min={0}
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
