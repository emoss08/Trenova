import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Bell, Building2, Database, FileText, TruckIcon } from "lucide-react";

export type ActionCategory = {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  actions: {
    value: string;
    label: string;
    description: string;
  }[];
};

const actionCategories: ActionCategory[] = [
  {
    label: "Shipment",
    icon: TruckIcon,
    actions: [
      {
        value: "shipment_update_status",
        label: "Update Status",
        description: "Update shipment status",
      },
    ],
  },
  {
    label: "Billing",
    icon: Building2,
    actions: [
      {
        value: "billing_validate_requirements",
        label: "Validate Requirements",
        description: "Validate billing requirements",
      },
    ],
  },
  {
    label: "Document",
    icon: FileText,
    actions: [
      {
        value: "document_validate_completeness",
        label: "Validate Completeness",
        description: "Check if all required documents are present",
      },
    ],
  },
  {
    label: "Notification",
    icon: Bell,
    actions: [
      {
        value: "notification_send_email",
        label: "Send Email",
        description: "Send an email notification",
      },
    ],
  },
  {
    label: "Data",
    icon: Database,
    actions: [
      {
        value: "data_api_call",
        label: "API Call",
        description: "Make an HTTP API request",
      },
    ],
  },
];

interface ActionTypeSelectorProps {
  value?: string;
  onChange: (value: string) => void;
}

export default function ActionTypeSelector({
  value,
  onChange,
}: ActionTypeSelectorProps) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger>
        <SelectValue placeholder="Select an action type..." />
      </SelectTrigger>
      <SelectContent>
        {actionCategories.map((category) => (
          <SelectGroup key={category.label}>
            <SelectLabel className="flex items-center gap-2">
              <category.icon className="size-4" />
              {category.label}
            </SelectLabel>
            {category.actions.map((action) => (
              <SelectItem key={action.value} value={action.value}>
                <div className="flex flex-col">
                  <span className="font-medium">{action.label}</span>
                  <span className="text-xs text-muted-foreground">
                    {action.description}
                  </span>
                </div>
              </SelectItem>
            ))}
          </SelectGroup>
        ))}
      </SelectContent>
    </Select>
  );
}
