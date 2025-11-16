import { Button } from "@/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { HelpCircle } from "lucide-react";

export type ActionCategory = {
  label: string;
  description: string;
  actions: {
    value: string;
    label: string;
    description: string;
    example?: string;
  }[];
};

const actionCategories: ActionCategory[] = [
  {
    label: "Shipment Operations",
    description: "Manage shipment lifecycle and assignments",
    actions: [
      {
        value: "shipment_update_status",
        label: "Update Shipment Status",
        description: "Change the current status of a shipment",
        example: "Mark shipment as 'In Transit' or 'Delivered'",
      },
    ],
  },
  {
    label: "Billing & Finance",
    description: "Handle billing, invoicing, and financial operations",
    actions: [
      {
        value: "billing_validate_requirements",
        label: "Validate Billing Readiness",
        description:
          "Check if shipment has all required information for billing",
        example: "Verify customer, charges, and delivery date are set",
      },
    ],
  },
  {
    label: "Document Management",
    description: "Manage shipment documents and compliance",
    actions: [
      {
        value: "document_validate_completeness",
        label: "Check Required Documents",
        description: "Verify all required documents are uploaded",
        example: "Check for BOL, POD, and Invoice before billing",
      },
    ],
  },
  {
    label: "Notifications & Alerts",
    description: "Send notifications to users, customers, or external systems",
    actions: [
      {
        value: "notification_send_email",
        label: "Send Email Notification",
        description: "Send a customizable email message",
        example: "Notify customer when shipment is out for delivery",
      },
    ],
  },
  {
    label: "Data & Integration",
    description: "Integrate with external systems and transform data",
    actions: [
      {
        value: "data_api_call",
        label: "Call External API",
        description: "Make HTTP request to external service",
        example: "Update tracking in customer's TMS system",
      },
    ],
  },
  {
    label: "Compliance & Safety",
    description: "Ensure regulatory compliance and safety requirements",
    actions: [
      {
        value: "compliance_check_hazmat",
        label: "Check Hazmat Compliance",
        description: "Verify hazardous materials documentation",
        example: "Ensure hazmat placards and paperwork are complete",
      },
    ],
  },
];

interface ActionTypeSelectorProps {
  value?: string;
  onChange: (value: string) => void;
}

export function ActionTypeSelector({
  value,
  onChange,
}: ActionTypeSelectorProps) {
  const selectedAction = actionCategories
    .flatMap((cat) => cat.actions)
    .find((action) => action.value === value);

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Select value={value} onValueChange={onChange}>
          <SelectTrigger className="flex-1 **:data-desc:hidden">
            <SelectValue placeholder="Select an action type..." />
          </SelectTrigger>
          <SelectContent className="[&_*[role=option]]:ps-2 [&_*[role=option]]:pe-8 [&_*[role=option]>span]:start-auto [&_*[role=option]>span]:end-2">
            {actionCategories.map((category) => {
              return (
                <SelectGroup key={category.label}>
                  <SelectLabel className="flex items-center gap-2 px-2 py-1.5 text-2xs font-semibold text-muted-foreground uppercase">
                    {category.label}
                  </SelectLabel>
                  {category.actions.map((action) => (
                    <SelectItem
                      key={action.value}
                      value={action.value}
                      className="pl-8 text-sm font-normal"
                    >
                      {action.label}
                      <span
                        className="mt-1 block text-xs text-muted-foreground"
                        data-desc
                      >
                        {action.description}
                      </span>
                    </SelectItem>
                  ))}
                </SelectGroup>
              );
            })}
          </SelectContent>
        </Select>

        <HoverCard>
          <HoverCardTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="flex size-7 cursor-help items-center justify-center [&_svg]:size-3.5"
            >
              <HelpCircle className="size-3.5" />
            </Button>
          </HoverCardTrigger>
          <HoverCardContent className="w-80 p-0" side="left">
            <div className="space-y-2">
              <div className="px-4 py-2">
                <h4 className="text-sm font-semibold">Action Types</h4>
                <p className="text-xs text-muted-foreground">
                  Actions are the operations performed by workflow nodes. Each
                  action can access workflow data using variables like{" "}
                  <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">
                    {"{"}
                    {"{"}trigger.shipmentId{"}}"}
                  </code>
                  .
                </p>
              </div>
              <div className="space-y-1">
                <p className="border-b border-border pb-1 text-center text-xs font-medium">
                  Available Categories
                </p>
                <div className="flex flex-col gap-1 px-4 py-1">
                  {actionCategories.map((cat) => {
                    return (
                      <div key={cat.label} className="flex items-start gap-2">
                        <div>
                          <p className="text-xs font-medium">{cat.label}</p>
                          <p className="text-xs text-muted-foreground">
                            {cat.description}
                          </p>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          </HoverCardContent>
        </HoverCard>
      </div>

      {selectedAction && selectedAction.example && (
        <div className="rounded-md border border-border bg-muted/50 p-3">
          <p className="text-xs font-medium text-muted-foreground">
            Example Use Case:
          </p>
          <p className="mt-1 text-sm">{selectedAction.example}</p>
        </div>
      )}
    </div>
  );
}
