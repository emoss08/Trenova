import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { VARIABLE_CATEGORIES } from "@/lib/workflow";
import { Braces } from "lucide-react";
import { useState } from "react";
import { CustomPathNotice } from "./config-help-components";
import { VariableCategoryItem } from "./variable-category-item";

interface VariableInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: "text" | "email" | "url";
}

export default function VariableInput({
  value,
  onChange,
  placeholder,
  type = "text",
}: VariableInputProps) {
  const [inputRef, setInputRef] = useState<HTMLInputElement | null>(null);

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
              className="flex size-7 items-center justify-center [&_svg]:size-3.5"
              size="icon"
              title="Insert variable"
            >
              <Braces className="size-3.5" />
            </Button>
          </PopoverTrigger>
          <PopoverContent
            className="w-96"
            side="right"
            sideOffset={15}
            align="center"
          >
            <div className="space-y-3">
              <div className="space-y-1">
                <p className="text-sm font-semibold">Workflow Variables</p>
                <p className="text-xs text-muted-foreground">
                  Click a variable to insert it at the cursor position.
                  Variables are resolved when the workflow executes.
                </p>
              </div>
              <Separator />
              <div className="max-h-80 space-y-4 overflow-y-auto pr-1">
                {VARIABLE_CATEGORIES.map((category) => (
                  <VariableCategoryItem
                    key={category.label}
                    category={category}
                    value={value}
                    onChange={onChange}
                    inputRef={inputRef}
                  />
                ))}
              </div>
              <Separator />
              <CustomPathNotice />
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}
