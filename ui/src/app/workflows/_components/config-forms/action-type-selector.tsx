import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export type ActionCategory = {
  label: string;
  actions: {
    value: string;
    label: string;
    description: string;
  }[];
};

const actionCategories: ActionCategory[] = [
  {
    label: "Shipment",
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

export function ActionTypeSelector({
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
            <SelectLabel className="px-2 py-1.5 text-xs text-muted-foreground">
              {category.label}
            </SelectLabel>
            {category.actions.map((action) => (
              <SelectItem
                key={action.value}
                value={action.value}
                className="text-sm font-normal"
              >
                {action.label}
              </SelectItem>
            ))}
          </SelectGroup>
        ))}
      </SelectContent>
    </Select>
  );
}
