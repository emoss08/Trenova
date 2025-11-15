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
import { conditionConfigSchema } from "@/lib/schemas/node-config-schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useWatch } from "react-hook-form";
import { type z } from "zod";

interface ConditionConfigFormProps {
  initialConfig: Record<string, any>;
  onSave: (config: Record<string, any>) => void;
  onCancel: () => void;
}

export function ConditionConfigForm({
  initialConfig,
  onSave,
  onCancel,
}: ConditionConfigFormProps) {
  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { errors },
  } = useForm<z.infer<typeof conditionConfigSchema>>({
    resolver: zodResolver(conditionConfigSchema),
    defaultValues: initialConfig,
  });

  const operator = useWatch({ control, name: "operator" });
  const value = useWatch({ control, name: "value" });
  const field = useWatch({ control, name: "field" });

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="field">Field</Label>
        <Input
          {...register("field")}
          placeholder="trigger.status"
          className="font-mono"
        />
        <p className="text-xs text-muted-foreground">
          Path to the field in workflow state (e.g., trigger.shipmentStatus,
          previousNode.result.weight)
        </p>
        {errors.field && (
          <p className="text-sm text-destructive">{errors.field.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="operator">Operator</Label>
        <Select
          onValueChange={(value) => setValue("operator", value as any)}
          defaultValue={operator}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select operator..." />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="equals">Equals (&==)</SelectItem>
            <SelectItem value="notEquals">Not Equals (!=)</SelectItem>
            <SelectItem value="contains">Contains</SelectItem>
            <SelectItem value="greaterThan">Greater Than (&gt;)</SelectItem>
            <SelectItem value="lessThan">Less Than (&lt;)</SelectItem>
          </SelectContent>
        </Select>
        {errors.operator && (
          <p className="text-sm text-destructive">{errors.operator.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="value">Value</Label>
        <Input
          {...register("value")}
          placeholder="Enter value to compare against"
        />
        <p className="text-xs text-muted-foreground">
          The value will be automatically converted to the correct type (string,
          number, or boolean)
        </p>
        {errors.value && (
          <p className="text-sm text-destructive">{errors.value.message}</p>
        )}
      </div>

      <div className="rounded-md border border-border bg-muted/50 p-3">
        <p className="text-sm font-medium">Preview</p>
        <p className="font-mono text-sm text-muted-foreground">
          {field ?? "field"} {operator === "equals" && "=="}
          {operator === "notEquals" && "!="}
          {operator === "contains" && "contains"}
          {operator === "greaterThan" && ">"}
          {operator === "lessThan" && "<"} {value ?? "value"}
        </p>
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
