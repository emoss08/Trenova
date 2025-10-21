"use no memo";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SQLEditorField } from "@/components/fields/sql-editor-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { PulsatingDots } from "@/components/ui/pulsating-dots";
import { Separator } from "@/components/ui/separator";
import { variableValueTypeChoices } from "@/lib/choices";
import {
  VariableFormatSchema,
  VariableSchema,
  VariableValueType,
} from "@/lib/schemas/variable-schema";
import { api } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  CheckCircleIcon,
  FlaskConicalIcon,
} from "lucide-react";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";

export function FormatForm() {
  const { control, watch } = useFormContext<VariableFormatSchema>();
  const [testValue, setTestValue] = useState("");
  const [testResult, setTestResult] = useState<string | null>(null);

  const formatSQL = watch("formatSql");
  const valueType = watch("valueType");

  const validateSQL = useMutation({
    mutationFn: async (sql: string) => {
      return await api.variables.validateFormatSQL({
        formatSQL: sql,
      });
    },
    onSuccess: (data) => {
      if (data.valid) {
        toast.success("SQL is valid", {
          icon: <CheckCircleIcon className="h-4 w-4" />,
        });
      }
    },
    onError: (error: any) => {
      const message = error.response?.data?.message || "Invalid SQL";
      toast.error(message, {
        icon: <AlertCircleIcon className="h-4 w-4" />,
      });
    },
  });

  const testFormat = useMutation({
    mutationFn: async () => {
      return await api.variables.testFormat({
        formatSQL,
        testValue,
      });
    },
    onSuccess: (data) => {
      setTestResult(data.result);
      toast.success("Test completed successfully");
    },
    onError: (error: any) => {
      const message = error.response?.data?.message || "Test failed";
      toast.error(message);
      setTestResult(null);
    },
  });

  return (
    <div className="space-y-6">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="isActive"
            label="Enabled"
            description="Make this format available for use across the application"
            position="left"
            outlined
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="name"
            label="Format Name"
            placeholder="e.g., Currency USD, Date Short, Phone Format"
            rules={{ required: true }}
            maxLength={100}
            description="A descriptive name for this format"
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="valueType"
            label="Value Type"
            placeholder="Select the type of value this format works with"
            description="The data type this format expects to receive"
            options={variableValueTypeChoices}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Explain what this format does and when to use it..."
            description="Help text for users to understand this format"
            rows={3}
          />
        </FormControl>
        <FormControl cols="full">
          <div className="relative">
            <SQLEditorField
              control={control}
              name="formatSql"
              rules={{ required: true }}
              label="Format SQL"
              placeholder="-- Example: UPPER(:value) or TO_CHAR(:value::numeric, 'FM$999,999.00')"
              description="SQL expression that transforms the value. Use :value as the placeholder."
              height="200px"
            />
            <div className="absolute right-3 bottom-[40px]">
              <Button
                type="button"
                variant="outline"
                size="xs"
                disabled={!formatSQL || validateSQL.isPending}
                onClick={() => validateSQL.mutate(formatSQL)}
              >
                {validateSQL.isPending ? (
                  <PulsatingDots color="foreground" size={0.5} />
                ) : (
                  <>
                    <CheckCircleIcon className="size-4" />
                    Validate SQL
                  </>
                )}
              </Button>
            </div>
          </div>
        </FormControl>
      </FormGroup>
      <Separator />
      <FormSection
        title="Test Format"
        description="Test your format with a sample value to see the result"
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-xs font-medium">Test Value</label>
              <Input
                type="text"
                value={testValue}
                onChange={(e) => setTestValue(e.target.value)}
                placeholder={mapValueTypeToPlaceholder(valueType)}
              />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium">Result</label>
              <div className="flex h-7 w-full rounded-md border border-input bg-primary/5 px-3 py-1 text-xs text-foreground">
                {testResult || (
                  <span className="text-muted-foreground">
                    Result will appear here...
                  </span>
                )}
              </div>
            </div>
          </div>
          <div className="flex justify-end">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={
                !formatSQL || !testValue || !valueType || testFormat.isPending
              }
              onClick={() => testFormat.mutate()}
            >
              {testFormat.isPending ? (
                "Testing..."
              ) : (
                <>
                  <FlaskConicalIcon className="mr-2 h-4 w-4" />
                  Test Format
                </>
              )}
            </Button>
          </div>
        </div>
      </FormSection>
    </div>
  );
}

function mapValueTypeToPlaceholder(valueType: VariableSchema["valueType"]) {
  switch (valueType) {
    case VariableValueType.enum.Number:
      return "e.g., 1234.56";
    case VariableValueType.enum.Date:
      return "e.g., 2024-12-15";
    case VariableValueType.enum.Boolean:
      return "true or false";
    case VariableValueType.enum.String:
      return "Enter a test value...";
    case VariableValueType.enum.Currency:
      return "e.g., 1234.56";
  }
}
