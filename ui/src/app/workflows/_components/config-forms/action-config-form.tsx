import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import {
  billingValidateRequirementsConfigSchema,
  dataAPICallConfigSchema,
  documentValidateCompletenessConfigSchema,
  notificationSendEmailConfigSchema,
  shipmentUpdateStatusConfigSchema,
} from "@/lib/schemas/node-config-schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertCircle, CheckCircle2, Info, Plus, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { type z } from "zod";
import VariableInput from "./variable-input";

// Reusable help text component
function FieldHelp({
  children,
  type = "info",
}: {
  children: React.ReactNode;
  type?: "info" | "warning" | "success";
}) {
  const Icon =
    type === "warning"
      ? AlertCircle
      : type === "success"
        ? CheckCircle2
        : Info;
  const colorClass =
    type === "warning"
      ? "text-yellow-600 dark:text-yellow-500"
      : type === "success"
        ? "text-green-600 dark:text-green-500"
        : "text-muted-foreground";

  return (
    <div className="flex items-start gap-2 text-xs text-muted-foreground">
      <Icon className={`mt-0.5 size-3.5 shrink-0 ${colorClass}`} />
      <p>{children}</p>
    </div>
  );
}

interface ActionConfigFormProps {
  actionType: string;
  initialConfig: Record<string, any>;
  onSave: (config: Record<string, any>) => void;
  onCancel: () => void;
}

function ShipmentUpdateStatusForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof shipmentUpdateStatusConfigSchema>>({
    resolver: zodResolver(shipmentUpdateStatusConfigSchema),
    defaultValues: initialConfig,
  });

  const shipmentId = useWatch({ control, name: "shipmentId" });
  const status = useWatch({ control, name: "status" });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-5">
      <div className="space-y-3">
        <div className="space-y-2">
          <Label htmlFor="shipmentId" className="text-sm font-medium">
            Shipment ID
          </Label>
          <VariableInput
            value={shipmentId || ""}
            onChange={(value) => setValue("shipmentId", value)}
            placeholder="{{trigger.shipmentId}}"
          />
          <FieldHelp>
            The ID of the shipment to update. Use{" "}
            <code className="rounded bg-muted px-1 font-mono">
              {"{"}
              {"{"}trigger.shipmentId{"}}"}
            </code>{" "}
            to reference the shipment from the workflow trigger.
          </FieldHelp>
          {errors.shipmentId && (
            <p className="text-sm text-destructive">
              {errors.shipmentId.message}
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="status" className="text-sm font-medium">
            New Status
          </Label>
          <Select
            onValueChange={(value) => setValue("status", value as any)}
            value={status}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select status..." />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="new">New</SelectItem>
              <SelectItem value="in_transit">In Transit</SelectItem>
              <SelectItem value="delivered">Delivered</SelectItem>
              <SelectItem value="cancelled">Cancelled</SelectItem>
              <SelectItem value="on_hold">On Hold</SelectItem>
            </SelectContent>
          </Select>
          <FieldHelp>
            The status to set on the shipment. This will update the shipment's
            status field in the database.
          </FieldHelp>
          {errors.status && (
            <p className="text-sm text-destructive">{errors.status.message}</p>
          )}
        </div>
      </div>

      <Separator />

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Configuration</Button>
      </div>
    </form>
  );
}

// Notification Send Email Form
function NotificationSendEmailForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof notificationSendEmailConfigSchema>>({
    resolver: zodResolver(notificationSendEmailConfigSchema),
    defaultValues: initialConfig,
  });

  const to = useWatch({ control, name: "to" });
  const subject = useWatch({ control, name: "subject" });
  const body = useWatch({ control, name: "body" });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-5">
      <div className="space-y-3">
        <div className="space-y-2">
          <Label htmlFor="to" className="text-sm font-medium">
            Recipient Email Address
          </Label>
          <VariableInput
            value={to || ""}
            onChange={(value) => setValue("to", value)}
            placeholder="customer@example.com or {{trigger.customer.email}}"
            type="email"
          />
          <FieldHelp>
            The email address to send to. Can be a static email address or a
            variable like{" "}
            <code className="rounded bg-muted px-1 font-mono">
              {"{"}
              {"{"}trigger.customer.email{"}}"}
            </code>
            .
          </FieldHelp>
          {errors.to && (
            <p className="text-sm text-destructive">{errors.to.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="subject" className="text-sm font-medium">
            Email Subject
          </Label>
          <Input
            value={subject || ""}
            onChange={(e) => setValue("subject", e.target.value)}
            placeholder="Shipment {{trigger.proNumber}} Status Update"
            className="font-mono text-sm"
          />
          <FieldHelp>
            The subject line of the email. You can use variables to personalize
            it, such as{" "}
            <code className="rounded bg-muted px-1 font-mono">
              Shipment {"{"}
              {"{"}trigger.proNumber{"}}"}
            </code>
            .
          </FieldHelp>
          {errors.subject && (
            <p className="text-sm text-destructive">{errors.subject.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="body" className="text-sm font-medium">
            Email Body
          </Label>
          <Textarea
            value={body || ""}
            onChange={(e) => setValue("body", e.target.value)}
            placeholder={`Hello,\n\nYour shipment {{trigger.proNumber}} has been updated to status: {{trigger.status}}.\n\nThank you!`}
            rows={6}
            className="font-mono text-sm"
          />
          <FieldHelp>
            The main content of the email. Use variables to include dynamic
            information from the workflow. HTML is supported.
          </FieldHelp>
          {errors.body && (
            <p className="text-sm text-destructive">{errors.body.message}</p>
          )}
        </div>
      </div>

      <div className="rounded-md border border-blue-200 bg-blue-50 p-3 dark:border-blue-900 dark:bg-blue-950">
        <FieldHelp type="info">
          <span className="font-medium">Preview:</span> The email will be sent
          from your organization&apos;s configured email profile. Variables will
          be replaced with actual values when the workflow executes.
        </FieldHelp>
      </div>

      <Separator />

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Configuration</Button>
      </div>
    </form>
  );
}

export function BillingValidateRequirementsForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof billingValidateRequirementsConfigSchema>>({
    resolver: zodResolver(billingValidateRequirementsConfigSchema),
    defaultValues: initialConfig,
  });

  const shipmentId = useWatch({ control, name: "shipmentId" });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="shipmentId">Shipment ID</Label>
        <VariableInput
          value={shipmentId || ""}
          onChange={(value) => setValue("shipmentId", value)}
          placeholder="{{trigger.shipmentId}}"
        />
        {errors.shipmentId && (
          <p className="text-sm text-destructive">
            {errors.shipmentId.message}
          </p>
        )}
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Configuration</Button>
      </div>
    </form>
  );
}

function DataAPICallForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof dataAPICallConfigSchema>>({
    resolver: zodResolver(dataAPICallConfigSchema),
    defaultValues: initialConfig,
  });

  const url = useWatch({ control, name: "url" });
  const method = useWatch({ control, name: "method" });
  const body = useWatch({ control, name: "body" });
  const [headers, setHeaders] = useState<Record<string, string>>(
    initialConfig.headers || {},
  );
  const [newHeaderKey, setNewHeaderKey] = useState("");
  const [newHeaderValue, setNewHeaderValue] = useState("");

  const addHeader = () => {
    if (newHeaderKey && newHeaderValue) {
      const updated = { ...headers, [newHeaderKey]: newHeaderValue };
      setHeaders(updated);
      setValue("headers", updated);
      setNewHeaderKey("");
      setNewHeaderValue("");
    }
  };

  const removeHeader = (key: string) => {
    const updated = { ...headers };
    // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
    delete updated[key];
    setHeaders(updated);
    setValue("headers", updated);
  };

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="url">URL</Label>
        <VariableInput
          value={url || ""}
          onChange={(value) => setValue("url", value)}
          placeholder="https://api.example.com/endpoint"
          type="url"
        />
        {errors.url && (
          <p className="text-sm text-destructive">{errors.url.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="method">Method</Label>
        <Select
          onValueChange={(value) => setValue("method", value as any)}
          value={method || "GET"}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="GET">GET</SelectItem>
            <SelectItem value="POST">POST</SelectItem>
            <SelectItem value="PUT">PUT</SelectItem>
            <SelectItem value="DELETE">DELETE</SelectItem>
            <SelectItem value="PATCH">PATCH</SelectItem>
          </SelectContent>
        </Select>
        {errors.method && (
          <p className="text-sm text-destructive">{errors.method.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label>Headers</Label>
        <div className="space-y-2">
          {Object.entries(headers).map(([key, value]) => (
            <div key={key} className="flex items-center gap-2">
              <Badge variant="secondary" className="flex items-center gap-1">
                <span className="font-mono text-xs">
                  {key}: {value}
                </span>
                <button
                  type="button"
                  onClick={() => removeHeader(key)}
                  className="ml-1"
                >
                  <X className="size-3" />
                </button>
              </Badge>
            </div>
          ))}
          <div className="flex gap-2">
            <Input
              placeholder="Header name"
              value={newHeaderKey}
              onChange={(e) => setNewHeaderKey(e.target.value)}
              className="flex-1"
            />
            <Input
              placeholder="Header value"
              value={newHeaderValue}
              onChange={(e) => setNewHeaderValue(e.target.value)}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={addHeader}
            >
              <Plus className="size-4" />
            </Button>
          </div>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="body">Request Body (JSON)</Label>
        <Textarea
          value={body || ""}
          onChange={(e) => setValue("body", e.target.value)}
          placeholder='{"key": "value"}'
          rows={4}
          className="font-mono text-sm"
        />
        {errors.body && (
          <p className="text-sm text-destructive">{errors.body.message}</p>
        )}
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Configuration</Button>
      </div>
    </form>
  );
}

// Document Validate Completeness Form
function DocumentValidateCompletenessForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<z.infer<typeof documentValidateCompletenessConfigSchema>>({
    resolver: zodResolver(documentValidateCompletenessConfigSchema),
    defaultValues: initialConfig,
  });

  const shipmentId = useWatch({ control, name: "shipmentId" });
  const [requiredDocuments, setRequiredDocuments] = useState<string[]>(
    initialConfig.requiredDocuments || [],
  );
  const [newDocument, setNewDocument] = useState("");

  const addDocument = () => {
    if (newDocument) {
      const updated = [...requiredDocuments, newDocument];
      setRequiredDocuments(updated);
      setValue("requiredDocuments", updated);
      setNewDocument("");
    }
  };

  const removeDocument = (index: number) => {
    const updated = requiredDocuments.filter((_, i) => i !== index);
    setRequiredDocuments(updated);
    setValue("requiredDocuments", updated);
  };

  useEffect(() => {
    setValue("requiredDocuments", requiredDocuments);
  }, [requiredDocuments, setValue]);

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="shipmentId">Shipment ID</Label>
        <VariableInput
          value={shipmentId || ""}
          onChange={(value) => setValue("shipmentId", value)}
          placeholder="{{trigger.shipmentId}}"
        />
        {errors.shipmentId && (
          <p className="text-sm text-destructive">
            {errors.shipmentId.message}
          </p>
        )}
      </div>

      <div className="space-y-2">
        <Label>Required Documents</Label>
        <div className="space-y-2">
          {requiredDocuments.map((doc, index) => (
            <div key={index} className="flex items-center gap-2">
              <Badge variant="secondary" className="flex items-center gap-1">
                <span className="text-xs">{doc}</span>
                <button
                  type="button"
                  onClick={() => removeDocument(index)}
                  className="ml-1"
                >
                  <X className="size-3" />
                </button>
              </Badge>
            </div>
          ))}
          <div className="flex gap-2">
            <Input
              placeholder="Document type (e.g., BOL, POD, Invoice)"
              value={newDocument}
              onChange={(e) => setNewDocument(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addDocument();
                }
              }}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={addDocument}
            >
              <Plus className="size-4" />
            </Button>
          </div>
        </div>
        {errors.requiredDocuments && (
          <p className="text-sm text-destructive">
            {errors.requiredDocuments.message}
          </p>
        )}
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Save Configuration</Button>
      </div>
    </form>
  );
}

// Main Action Config Form component that routes to the right form
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
