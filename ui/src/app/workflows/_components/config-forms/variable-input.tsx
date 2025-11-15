import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { Braces, Info } from "lucide-react";
import { useState } from "react";

interface VariableInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: "text" | "email" | "url";
}

// Categorized workflow variables
const VARIABLE_CATEGORIES = [
  {
    label: "Trigger Data",
    description: "Data from the event that started this workflow",
    variables: [
      {
        value: "{{trigger.shipmentId}}",
        label: "Shipment ID",
        description: "ID of the shipment that triggered the workflow",
      },
      {
        value: "{{trigger.status}}",
        label: "Status",
        description: "Current status value from the trigger",
      },
      {
        value: "{{trigger.customerId}}",
        label: "Customer ID",
        description: "Associated customer ID",
      },
      {
        value: "{{trigger.proNumber}}",
        label: "PRO Number",
        description: "Shipment PRO number",
      },
    ],
  },
  {
    label: "Previous Node Output",
    description: "Results from the node that executed before this one",
    variables: [
      {
        value: "{{previousNode.result}}",
        label: "Result Data",
        description: "Full result object from previous action",
      },
      {
        value: "{{previousNode.success}}",
        label: "Success Status",
        description: "Whether the previous action succeeded (true/false)",
      },
      {
        value: "{{previousNode.message}}",
        label: "Message",
        description: "Message or error from previous action",
      },
    ],
  },
  {
    label: "Workflow Context",
    description: "Information about the current workflow execution",
    variables: [
      {
        value: "{{workflow.executionId}}",
        label: "Execution ID",
        description: "Unique ID for this workflow run",
      },
      {
        value: "{{workflow.startedAt}}",
        label: "Started At",
        description: "Timestamp when workflow started",
      },
      {
        value: "{{workflow.organizationId}}",
        label: "Organization ID",
        description: "ID of the organization running this workflow",
      },
    ],
  },
];

export default function VariableInput({
  value,
  onChange,
  placeholder,
  type = "text",
}: VariableInputProps) {
  const [inputRef, setInputRef] = useState<HTMLInputElement | null>(null);

  const insertVariable = (variable: string) => {
    if (!inputRef) return;

    const start = inputRef.selectionStart || 0;
    const end = inputRef.selectionEnd || 0;
    const newValue =
      value.substring(0, start) + variable + value.substring(end);

    onChange(newValue);

    setTimeout(() => {
      inputRef.focus();
      const newCursorPos = start + variable.length;
      inputRef.setSelectionRange(newCursorPos, newCursorPos);
    }, 0);
  };

  return (
    <div className="space-y-1.5">
      <div className="flex gap-2">
        <Input
          ref={setInputRef}
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="flex-1 font-mono text-sm"
        />
        <Popover>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="icon"
              title="Insert variable"
            >
              <Braces className="size-4" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-96" align="end">
            <div className="space-y-3">
              <div className="space-y-1">
                <p className="text-sm font-semibold">Workflow Variables</p>
                <p className="text-xs text-muted-foreground">
                  Click a variable to insert it at the cursor position. Variables
                  are resolved when the workflow executes.
                </p>
              </div>

              <Separator />

              <div className="max-h-80 space-y-4 overflow-y-auto pr-1">
                {VARIABLE_CATEGORIES.map((category) => (
                  <div key={category.label} className="space-y-2">
                    <div>
                      <p className="text-xs font-semibold text-foreground">
                        {category.label}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {category.description}
                      </p>
                    </div>
                    <div className="space-y-1">
                      {category.variables.map((variable) => (
                        <button
                          key={variable.value}
                          type="button"
                          onClick={() => insertVariable(variable.value)}
                          className="block w-full rounded-md px-3 py-2 text-left hover:bg-accent"
                        >
                          <div className="flex items-start justify-between gap-2">
                            <div className="min-w-0 flex-1">
                              <div className="text-sm font-medium">
                                {variable.label}
                              </div>
                              <div className="mt-0.5 font-mono text-xs text-muted-foreground">
                                {variable.value}
                              </div>
                              <div className="mt-1 text-xs text-muted-foreground">
                                {variable.description}
                              </div>
                            </div>
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                ))}
              </div>

              <Separator />

              <div className="flex items-start gap-2 rounded-md bg-muted p-2">
                <Info className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
                <div className="text-xs text-muted-foreground">
                  <p className="font-medium">Custom Paths</p>
                  <p className="mt-1">
                    You can access nested fields using dot notation:{" "}
                    <code className="rounded bg-background px-1 py-0.5 font-mono">
                      {"{"}
                      {"{"}trigger.customer.email{"}}"}
                    </code>
                  </p>
                </div>
              </div>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}
