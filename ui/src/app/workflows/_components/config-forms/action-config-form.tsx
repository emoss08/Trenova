import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { shipmentStatusChoices } from "@/lib/choices";
import {
  BillingValidateRequirementsConfig,
  billingValidateRequirementsConfigSchema,
  notificationSendEmailConfigSchema,
  ShipmentUpdateStatusConfigSchema,
  shipmentUpdateStatusConfigSchema,
} from "@/lib/schemas/node-config-schema";
import { ActionConfigFormProps } from "@/types/workflow";
import { zodResolver } from "@hookform/resolvers/zod";
import { InfoIcon } from "lucide-react";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { type z } from "zod";
import { DataAPICallForm } from "./data-api-call-form";
import { DocumentValidateCompletenessForm } from "./document-validate-compless-form";
import { VariableInput } from "./inputs/variable-input";

function ShipmentUpdateStatusForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const { control, handleSubmit, setError } =
    useForm<ShipmentUpdateStatusConfigSchema>({
      resolver: zodResolver(shipmentUpdateStatusConfigSchema),
      defaultValues: initialConfig,
    });

  const handleSave = useCallback(
    (values: ShipmentUpdateStatusConfigSchema) => {
      const validated = shipmentUpdateStatusConfigSchema.safeParse(values);
      if (!validated.success) {
        validated.error.issues.forEach((issue) => {
          setError(issue.path[0] as keyof ShipmentUpdateStatusConfigSchema, {
            message: issue.message,
          });
        });
        return;
      }

      onSave(validated.data);
    },
    [onSave, setError],
  );

  return (
    <Form onSubmit={handleSubmit(handleSave)}>
      <FormGroup cols={1} className="px-4 pb-2">
        <FormControl>
          <VariableInput
            name="shipmentId"
            control={control}
            rules={{ required: true }}
            label="Shipment ID"
            description="The ID of the shipment to update. Use {{trigger.shipmentId}} to reference the shipment from the workflow trigger."
            placeholder="{{trigger.shipmentId}}"
            type="text"
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label="New Status"
            placeholder="New Status"
            description="The status to set on the shipment. This will update the shipment's status field in the database."
            options={shipmentStatusChoices}
          />
        </FormControl>
      </FormGroup>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Configuration</Button>
      </DialogFooter>
    </Form>
  );
}

function NotificationSendEmailForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const { control, handleSubmit } = useForm<
    z.infer<typeof notificationSendEmailConfigSchema>
  >({
    resolver: zodResolver(notificationSendEmailConfigSchema),
    defaultValues: initialConfig,
  });

  return (
    <>
      <div className="px-4 pb-2">
        <div className="rounded-md border border-blue-500 bg-blue-500/10 p-2">
          <div className="flex items-center gap-1">
            <InfoIcon className="size-4 text-blue-500" />
            <span className="text-sm font-medium text-blue-500">Notice</span>
          </div>
          <p className="text-sm text-blue-500">
            The email will be sent from your organization&apos;s configured
            email profile. Variables will be replaced with actual values when
            the workflow executes.
          </p>
        </div>
      </div>
      <Form onSubmit={handleSubmit(onSave)}>
        <FormGroup cols={1} className="px-4 pb-2">
          <FormControl>
            <VariableInput
              name="to"
              control={control}
              rules={{ required: true }}
              label="Recipient Email Address"
              description="The email address to send to. Can be a static email address or a variable like {{trigger.customer.email}}."
              placeholder="customer@example.com or {{trigger.customer.email}}"
              type="email"
            />
          </FormControl>
          <FormControl>
            <InputField
              name="subject"
              control={control}
              rules={{ required: true }}
              label="Email Subject"
              description="The subject line of the email. You can use variables to personalize it, such as {{trigger.proNumber}}."
              placeholder="Shipment {{trigger.proNumber}} Status Update"
              type="text"
            />
          </FormControl>
          <FormControl>
            <TextareaField
              name="body"
              control={control}
              rules={{ required: true }}
              label="Email Body"
              description="The main content of the email. You can use variables to include dynamic information from the workflow."
              placeholder={`Hello,\n\nYour shipment {{trigger.proNumber}} has been updated to status: {{trigger.status}}.\n\nThank you!`}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit">Save Configuration</Button>
        </DialogFooter>
      </Form>
    </>
  );
}

export function BillingValidateRequirementsForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const { control, handleSubmit } = useForm<BillingValidateRequirementsConfig>({
    resolver: zodResolver(billingValidateRequirementsConfigSchema),
    defaultValues: initialConfig,
  });

  return (
    <Form onSubmit={handleSubmit(onSave)}>
      <FormGroup cols={1} className="px-4 py-2">
        <FormControl>
          <VariableInput
            name="shipmentId"
            control={control}
            rules={{ required: true }}
            label="Shipment ID"
            description="The ID of the shipment to validate requirements for. Use {{trigger.shipmentId}} to reference the shipment from the workflow trigger."
            placeholder="{{trigger.shipmentId}}"
            type="text"
          />
        </FormControl>
      </FormGroup>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Configuration</Button>
      </DialogFooter>
    </Form>
  );
}

export function ActionConfigForm({
  actionType,
  initialConfig,
  onSave,
  onCancel,
}: ActionConfigFormProps) {
  switch (actionType) {
    case "shipment_update_status":
      return (
        <ShipmentUpdateStatusForm
          initialConfig={initialConfig}
          onSave={onSave}
          onCancel={onCancel}
        />
      );
    case "notification_send_email":
      return (
        <NotificationSendEmailForm
          initialConfig={initialConfig}
          onSave={onSave}
          onCancel={onCancel}
        />
      );
    case "billing_validate_requirements":
      return (
        <BillingValidateRequirementsForm
          initialConfig={initialConfig}
          onSave={onSave}
          onCancel={onCancel}
        />
      );
    case "data_api_call":
      return (
        <DataAPICallForm
          initialConfig={initialConfig}
          onSave={onSave}
          onCancel={onCancel}
        />
      );
    case "document_validate_completeness":
      return (
        <DocumentValidateCompletenessForm
          initialConfig={initialConfig}
          onSave={onSave}
          onCancel={onCancel}
        />
      );
    default:
      return (
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Configuration form for action type &quot;{actionType}&quot; is not
            yet implemented.
          </p>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </div>
      );
  }
}
