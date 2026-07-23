import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { fetchDashControl, updateDashControl } from "@trenova/shared/lib/graphql/driver-portal";
import { dashControlFormSchema, type DashControlFormValues } from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export default function DashControlForm() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({
    queryKey: ["dash-control"],
    queryFn: fetchDashControl,
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
      enableDetentionAlerts: data.enableDetentionAlerts,
      detentionAlertThresholdMinutes: data.detentionAlertThresholdMinutes,
    },
  });
  const { handleSubmit, setError, reset } = form;

  const mutation = useApiMutation({
    mutationFn: (values: DashControlFormValues) =>
      updateDashControl({
        version: data.version,
        ...values,
      }),
    onSuccess: (_, values) => {
      toast.success("Dash control updated — drivers see the change immediately");
      reset(values);
      void queryClient.invalidateQueries({ queryKey: ["dash-control"] });
    },
    setFormError: setError,
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
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function LoadWorkflowCard() {
  const { control } = useFormContext<DashControlFormValues>();
  const requireAck = useWatch({ control, name: "requireLoadAcknowledgment" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>Load Workflow</CardTitle>
        <CardDescription>
          What drivers can do on their assigned loads. Everything here is enforced server-side —
          turning a toggle off removes the feature from Dash immediately.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="requireLoadAcknowledgment"
              label="Load Acceptance"
              description="Drivers see an accept/decline card on new assignments so dispatch knows the load was received."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowLoadRefusals"
              label="Allow Declines"
              disabled={!requireAck}
              description="Drivers may decline a load with a reason. Turn off for forced dispatch — drivers can only acknowledge."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowStopActions"
              label="Self-Service Arrive / Depart"
              description="Drivers record their own arrivals and departures at stops, driving move status and detention math."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowLoadDocumentUpload"
              label="POD / BOL Upload"
              description="Drivers photograph and upload signed paperwork straight from the cab."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowLoadComments"
              label="Load Messaging"
              description="Drivers can send messages on load chat. Reading dispatch notes is always allowed."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function PayVisibilityCard() {
  const { control } = useFormContext<DashControlFormValues>();
  const showLoadPay = useWatch({ control, name: "showLoadPay" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>Pay Visibility</CardTitle>
        <CardDescription>
          Settlement statements are always visible to drivers — these toggles only control per-load
          pay detail shown before settlement.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="showLoadPay"
              label="Per-Load Pay"
              description="Show what each load pays and recent pay events as they accrue."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="showPayEstimates"
              label="Pay Estimates"
              disabled={!showLoadPay}
              description="Show an estimated payout on active loads before pay accrues, based on the driver's pay plan."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function MoneyCard() {
  const { control } = useFormContext<DashControlFormValues>();
  const allowExpenses = useWatch({ control, name: "allowExpenseSubmission" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>Expenses &amp; Disputes</CardTitle>
        <CardDescription>
          Driver-initiated money workflows — reimbursements and settlement challenges.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="allowExpenseSubmission"
              label="Expense Submission"
              description="Drivers submit out-of-pocket expenses (lumpers, tolls, scales) for reimbursement review."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="requireExpenseReceipt"
              label="Require Receipts"
              disabled={!allowExpenses}
              description="Expenses cannot be approved until a receipt photo is attached."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowSettlementDisputes"
              label="Settlement Disputes"
              description="Drivers can flag a statement or line item for review from Dash."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ProfileCard() {
  const { control } = useFormContext<DashControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Profile Self-Service</CardTitle>
        <CardDescription>
          What drivers can maintain on their own record. Compliance dates (CDL, medical) are always
          carrier-controlled regardless of these settings.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="allowProfileDocumentUpload"
              label="Qualification Document Upload"
              description="Drivers upload renewed CDLs, medical cards, and other DQ-file documents from their phone."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowContactInfoEdit"
              label="Contact Info Edits"
              description="Drivers keep their own phone, address, and emergency contact current."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowPtoRequests"
              label="Time-Off Requests"
              description="Drivers request PTO from Dash; requests land in the existing approval workflow."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AlertsCard() {
  const { control } = useFormContext<DashControlFormValues>();
  const detentionAlerts = useWatch({ control, name: "enableDetentionAlerts" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>Reminders &amp; Alerts</CardTitle>
        <CardDescription>
          Automated notifications driven by driver activity and credential dates.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="sendCredentialReminders"
              label="Credential Expiry Reminders"
              description="Push drivers reminders at 30/14/3 days before a credential expires. Compliance always gets expired-credential alerts."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enableDetentionAlerts"
              label="Detention Alerts"
              description="Alert dispatch when a driver dwells at a stop beyond the threshold — a billing candidate for detention accessorials."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="detentionAlertThresholdMinutes"
              label="Detention Threshold (minutes)"
              disabled={!detentionAlerts}
              rules={{ required: detentionAlerts }}
              description="Dwell time beyond which a stop is flagged. 120 minutes is the common free-time convention."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
