import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { fetchDashControl, updateDashControl } from "@trenova/shared/lib/graphql/driver-portal";
import { digestCadenceLabel, WEEKDAY_LABELS } from "@trenova/shared/lib/csa";
import {
  dashControlFormSchema,
  type DashControlFormValues,
} from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const CADENCE_OPTIONS = (["Immediate", "Daily", "Weekly"] as const).map((value) => ({
  value,
  label: digestCadenceLabel(value),
}));

const WEEKDAY_OPTIONS = WEEKDAY_LABELS.map((label, value) => ({ value: String(value), label }));

export default function DashControlForm() {
  const t = useT();

  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({
    queryKey: ["dash-control"],
    queryFn: ({ signal }) => fetchDashControl({ signal }),
  });

  const form = useForm<DashControlFormValues>({
    resolver: zodResolver(dashControlFormSchema) as Resolver<DashControlFormValues>,
    defaultValues: {
      requireLoadAcknowledgment: data.requireLoadAcknowledgment,
      allowLoadRefusals: data.allowLoadRefusals,
      allowStopActions: data.allowStopActions,
      allowLoadDocumentUpload: data.allowLoadDocumentUpload,
      allowLoadComments: data.allowLoadComments,
      showLoadPay: data.showLoadPay,
      showPayEstimates: data.showPayEstimates,
      allowExpenseSubmission: data.allowExpenseSubmission,
      requireExpenseReceipt: data.requireExpenseReceipt,
      allowSettlementDisputes: data.allowSettlementDisputes,
      allowProfileDocumentUpload: data.allowProfileDocumentUpload,
      allowContactInfoEdit: data.allowContactInfoEdit,
      allowPtoRequests: data.allowPtoRequests,
      sendCredentialReminders: data.sendCredentialReminders,
      requireContactChangeApproval: data.requireContactChangeApproval,
      driverDigestCadence: data.driverDigestCadence,
      driverDigestWeekday: String(
        data.driverDigestWeekday,
      ) as DashControlFormValues["driverDigestWeekday"],
      enableDetentionAlerts: data.enableDetentionAlerts,
      detentionAlertThresholdMinutes: data.detentionAlertThresholdMinutes,
    },
  });
  const { handleSubmit, reset } = form;

  const mutation = useApiMutation({
    mutationFn: (values: DashControlFormValues) =>
      updateDashControl({
        version: data.version,
        ...values,
        driverDigestWeekday: Number(values.driverDigestWeekday),
      }),
    onSuccess: (_, values) => {
      toast.success(t("Dash control updated — drivers see the change immediately"));
      reset(values);
      void queryClient.invalidateQueries({ queryKey: ["dash-control"] });
    },
    form,
    resourceName: "Dash Control",
  });

  const onSubmit = useCallback(
    (values: DashControlFormValues) => mutation.mutate(values),
    [mutation],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <LoadWorkflowCard />
          <PayVisibilityCard />
          <MoneyCard />
          <ProfileCard />
          <AlertsCard />
          <FormSaveDock saveButtonContent={t("Save Changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}

function LoadWorkflowCard() {
  const t = useT();

  const { control } = useFormContext<DashControlFormValues>();
  const requireAck = useWatch({ control, name: "requireLoadAcknowledgment" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Load Workflow")}</CardTitle>
        <CardDescription>
          {t(
            "What drivers can do on their assigned loads. Everything here is enforced server-side — turning a toggle off removes the feature from Dash immediately.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="requireLoadAcknowledgment"
              label={t("Load Acceptance")}
              description={t(
                "Drivers see an accept/decline card on new assignments so dispatch knows the load was received.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowLoadRefusals"
              label={t("Allow Declines")}
              disabled={!requireAck}
              description={t(
                "Drivers may decline a load with a reason. Turn off for forced dispatch — drivers can only acknowledge.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowStopActions"
              label={t("Self-Service Arrive / Depart")}
              description={t(
                "Drivers record their own arrivals and departures at stops, driving move status and detention math.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowLoadDocumentUpload"
              label={t("POD / BOL Upload")}
              description={t(
                "Drivers photograph and upload signed paperwork straight from the cab.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowLoadComments"
              label={t("Load Messaging")}
              description={t(
                "Drivers can send messages on load chat. Reading dispatch notes is always allowed.",
              )}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function PayVisibilityCard() {
  const t = useT();

  const { control } = useFormContext<DashControlFormValues>();
  const showLoadPay = useWatch({ control, name: "showLoadPay" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Pay Visibility")}</CardTitle>
        <CardDescription>
          {t(
            "Settlement statements are always visible to drivers — these toggles only control per-load pay detail shown before settlement.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="showLoadPay"
              label={t("Per-Load Pay")}
              description={t("Show what each load pays and recent pay events as they accrue.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="showPayEstimates"
              label={t("Pay Estimates")}
              disabled={!showLoadPay}
              description={t(
                "Show an estimated payout on active loads before pay accrues, based on the driver's pay plan.",
              )}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function MoneyCard() {
  const t = useT();

  const { control } = useFormContext<DashControlFormValues>();
  const allowExpenses = useWatch({ control, name: "allowExpenseSubmission" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Expenses & Disputes")}</CardTitle>
        <CardDescription>
          {t("Driver-initiated money workflows — reimbursements and settlement challenges.")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="allowExpenseSubmission"
              label={t("Expense Submission")}
              description={t(
                "Drivers submit out-of-pocket expenses (lumpers, tolls, scales) for reimbursement review.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requireExpenseReceipt"
              label={t("Require Receipts")}
              disabled={!allowExpenses}
              description={t("Expenses cannot be approved until a receipt photo is attached.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowSettlementDisputes"
              label={t("Settlement Disputes")}
              description={t("Drivers can flag a statement or line item for review from Dash.")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ProfileCard() {
  const t = useT();

  const { control } = useFormContext<DashControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Profile Self-Service")}</CardTitle>
        <CardDescription>
          {t(
            "What drivers can maintain on their own record. Compliance dates (CDL, medical) are always carrier-controlled regardless of these settings.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="allowProfileDocumentUpload"
              label={t("Qualification Document Upload")}
              description={t(
                "Drivers upload renewed CDLs, medical cards, and other DQ-file documents from their phone.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowContactInfoEdit"
              label={t("Contact Info Edits")}
              description={t(
                "Drivers keep their own phone, address, and emergency contact current.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requireContactChangeApproval"
              label={t("Approve Contact Edits")}
              description={t(
                "A driver's edit waits on the office as a change request instead of landing straight on the record.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowPtoRequests"
              label={t("Time-Off Requests")}
              description={t(
                "Drivers request PTO from Dash; requests land in the existing approval workflow.",
              )}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AlertsCard() {
  const t = useT();

  const { control } = useFormContext<DashControlFormValues>();
  const detentionAlerts = useWatch({ control, name: "enableDetentionAlerts" });
  const reminders = useWatch({ control, name: "sendCredentialReminders" });
  const cadence = useWatch({ control, name: "driverDigestCadence" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Reminders & Alerts")}</CardTitle>
        <CardDescription>
          {t("Automated notifications driven by driver activity and credential dates.")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="sendCredentialReminders"
              label={t("Credential Expiry Reminders")}
              description={t(
                "Push drivers reminders at 30/14/3 days before a credential expires. Compliance always gets expired-credential alerts.",
              )}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="driverDigestCadence"
              label={t("How Drivers Are Told")}
              options={CADENCE_OPTIONS}
              isReadOnly={!reminders}
              description={t(
                "Bundle everything a driver owes into one notice instead of one each. The per-obligation reminders are suppressed while a round-up is in use, so nobody is told twice.",
              )}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="driverDigestWeekday"
              label={t("Weekly Round-Up Day")}
              options={WEEKDAY_OPTIONS}
              isReadOnly={cadence !== "Weekly"}
              description={t(
                "The day the weekly notice goes out. A weekly round-up looks a fortnight ahead so nothing falls due in the gap between two of them.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enableDetentionAlerts"
              label={t("Detention Alerts")}
              description={t(
                "Alert dispatch when a driver dwells at a stop beyond the threshold — a billing candidate for detention accessorials.",
              )}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="detentionAlertThresholdMinutes"
              label={t("Detention Threshold (minutes)")}
              disabled={!detentionAlerts}
              rules={{ required: detentionAlerts }}
              description={t(
                "Dwell time beyond which a stop is flagged. 120 minutes is the common free-time convention.",
              )}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
