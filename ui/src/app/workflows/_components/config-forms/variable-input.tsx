import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Braces } from "lucide-react";
import { useState } from "react";

interface VariableInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: "text" | "email" | "url";
}

// Common workflow variables available for insertion
const AVAILABLE_VARIABLES = [
  { value: "{{trigger.shipmentId}}", label: "Trigger: Shipment ID" },
  { value: "{{trigger.status}}", label: "Trigger: Status" },
  { value: "{{trigger.customerId}}", label: "Trigger: Customer ID" },
  { value: "{{trigger.carrierId}}", label: "Trigger: Carrier ID" },
  { value: "{{trigger.documentId}}", label: "Trigger: Document ID" },
  { value: "{{previousNode.result}}", label: "Previous Node: Result" },
  { value: "{{previousNode.data}}", label: "Previous Node: Data" },
  { value: "{{workflow.executionId}}", label: "Workflow: Execution ID" },
  { value: "{{workflow.startedAt}}", label: "Workflow: Started At" },
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
    <div className="flex gap-2">
      <Input
        ref={setInputRef}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="flex-1"
      />
      <Popover>
        <PopoverTrigger asChild>
          <Button type="button" variant="outline" size="icon">
            <Braces className="size-4" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-80" align="end">
          <div className="space-y-2">
            <p className="text-sm font-medium">Insert Variable</p>
            <p className="text-xs text-muted-foreground">
              Click a variable to insert it at the cursor position
            </p>
            <div className="max-h-64 space-y-1 overflow-y-auto">
              {AVAILABLE_VARIABLES.map((variable) => (
                <button
                  key={variable.value}
                  type="button"
                  onClick={() => insertVariable(variable.value)}
                  className="block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-accent"
                >
                  <div className="font-medium">{variable.label}</div>
                  <div className="font-mono text-xs text-muted-foreground">
                    {variable.value}
                  </div>
                </button>
              ))}
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
