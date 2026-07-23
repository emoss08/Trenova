import { FieldWrapper } from "@/components/fields/field-components";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import {
  type FormulaTemplate,
  formulaTemplateSchema,
  type FormulaTemplateStatus,
  type FormulaTemplateType,
} from "@/types/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileCode2, Save, X } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";
import { ExpressionEditor } from "./expression-editor";
import { ExpressionTestPanel } from "./expression-test-panel";
import { FormulaReferencePanel } from "./formula-reference-panel";
import { VariableDefinitionEditor } from "./variable-definition-editor";

type FormulaTemplateEditorProps = {
  initialData?: Partial<FormulaTemplate>;
  onSubmit: (data: FormulaTemplate) => void;
  onCancel?: () => void;
  isLoading?: boolean;
  className?: string;
};

const TEMPLATE_TYPES: {
  value: FormulaTemplateType;
  label: string;
}[] = [
  { value: "FreightCharge", label: "Freight Charge" },
  { value: "AccessorialCharge", label: "Accessorial Charge" },
];

const TEMPLATE_STATUSES: {
  value: FormulaTemplateStatus;
  label: string;
}[] = [
  { value: "Draft", label: "Draft" },
  { value: "Active", label: "Active" },
  { value: "Inactive", label: "Inactive" },
];

export function FormulaTemplateEditor({
  initialData,
  onSubmit,
  onCancel,
  isLoading = false,
  className,
}: FormulaTemplateEditorProps) {
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(formulaTemplateSchema),
    defaultValues: {
      name: "",
      description: "",
      type: "FreightCharge",
      expression: "",
      status: "Draft",
      schemaId: "shipment",
      variableDefinitions: [],
      ...initialData,
    },
  });

  const expression = useWatch({ control, name: "expression" });
  const schemaId = useWatch({ control, name: "schemaId" });
  const variableDefinitions = useWatch({
    control,
    name: "variableDefinitions",
  });

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className={cn("space-y-6", className)}
    >
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Card>
            <CardHeader className="flex flex-row items-center gap-3 border-b py-3">
              <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
                <FileCode2 className="size-4 text-primary" />
              </div>
              <div>
                <CardTitle className="text-sm font-medium">
                  Template Details
                </CardTitle>
                <p className="text-xs text-muted-foreground">
                  Basic information about your formula template
                </p>
              </div>
            </CardHeader>
            <CardContent className="space-y-4 p-4">
              <div className="grid grid-cols-2 gap-4">
                <FieldWrapper
                  label="Name"
                  required
                  error={errors.name?.message}
                >
                  <Input
                    {...register("name")}
                    placeholder="e.g., Standard Mileage Rate"
                    aria-invalid={!!errors.name}
                    className="h-9"
                  />
                </FieldWrapper>

                <div className="grid grid-cols-2 gap-3">
                  <FieldWrapper
                    label="Type"
                    required
                    error={errors.type?.message}
                  >
                    <select
                      {...register("type")}
                      className={cn(
                        "h-9 w-full rounded-lg border border-input bg-input/30 px-2.5 text-sm",
                        "focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:outline-none",
                      )}
                    >
                      {TEMPLATE_TYPES.map((type) => (
                        <option key={type.value} value={type.value}>
                          {type.label}
                        </option>
                      ))}
                    </select>
                  </FieldWrapper>

                  <FieldWrapper
                    label="Status"
                    required
                    error={errors.status?.message}
                  >
                    <select
                      {...register("status")}
                      className={cn(
                        "h-9 w-full rounded-lg border border-input bg-input/30 px-2.5 text-sm",
                        "focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:outline-none",
                      )}
                    >
                      {TEMPLATE_STATUSES.map((status) => (
                        <option key={status.value} value={status.value}>
                          {status.label}
                        </option>
                      ))}
                    </select>
                  </FieldWrapper>
                </div>
              </div>

              <FieldWrapper
                label="Description"
                description="Optional description of what this formula calculates"
                error={errors.description?.message}
              >
                <Input
                  {...register("description")}
                  placeholder="e.g., Calculates freight charge based on mileage and weight"
                  className="h-9"
                />
              </FieldWrapper>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="border-b py-3">
              <div className="flex items-center gap-3">
                <div className="flex size-8 items-center justify-center rounded-lg bg-chart-2/10">
                  <FileCode2 className="size-4 text-chart-2" />
                </div>
                <div>
                  <CardTitle className="text-sm font-medium">
                    Expression
                  </CardTitle>
                  <p className="text-xs text-muted-foreground">
                    Write your formula using variables and functions. Press
                    Ctrl+Space for autocomplete.
                  </p>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-4">
              <ExpressionEditor
                name="expression"
                customVariables={variableDefinitions}
                rules={{ required: true }}
                placeholder="e.g., totalDistance * 2.50 + max(totalWeight * 0.05, 25)"
                height="160px"
                control={control}
              />
              <ExpressionTestPanel
                expression={expression}
                schemaId={schemaId}
                customVariables={variableDefinitions}
              />
            </CardContent>
          </Card>

          <VariableDefinitionEditor
            control={control as never}
            register={register as never}
          />
        </div>

        <div className="space-y-6">
          <FormulaReferencePanel />

          <Card>
            <CardFooter className="flex-col gap-2 p-4">
              <Button
                type="submit"
                className="w-full gap-2"
                isLoading={isLoading}
                loadingText="Saving..."
              >
                <Save className="size-4" />
                Save Template
              </Button>
              {onCancel && (
                <Button
                  type="button"
                  variant="outline"
                  className="w-full gap-2"
                  onClick={onCancel}
                  disabled={isLoading}
                >
                  <X className="size-4" />
                  Cancel
                </Button>
              )}
            </CardFooter>
          </Card>
        </div>
      </div>
    </form>
  );
}
