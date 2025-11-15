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
import { Textarea } from "@/components/ui/textarea";
import {
  billingValidateRequirementsConfigSchema,
  dataAPICallConfigSchema,
  documentValidateCompletenessConfigSchema,
  notificationSendEmailConfigSchema,
  shipmentUpdateStatusConfigSchema,
} from "@/lib/schemas/node-config-schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { type z } from "zod";
import VariableInput from "./variable-input";

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
        <Label htmlFor="status">New Status</Label>
        <Select
          onValueChange={(value) => setValue("status", value as any)}
          defaultValue={initialConfig.status}
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
        {errors.status && (
          <p className="text-sm text-destructive">{errors.status.message}</p>
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
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="to">To (Email)</Label>
        <VariableInput
          value={to || ""}
          onChange={(value) => setValue("to", value)}
          placeholder="{{trigger.customerEmail}}"
          type="email"
        />
        {errors.to && (
          <p className="text-sm text-destructive">{errors.to.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="subject">Subject</Label>
        <Input
          value={subject || ""}
          onChange={(e) => setValue("subject", e.target.value)}
          placeholder="Shipment Update Notification"
        />
        {errors.subject && (
          <p className="text-sm text-destructive">{errors.subject.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="body">Body</Label>
        <Textarea
          value={body || ""}
          onChange={(e) => setValue("body", e.target.value)}
          placeholder="Your shipment has been updated..."
          rows={5}
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
          defaultValue={method || "GET"}
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
