import { CheckboxField } from "@/components/fields/checkbox-field";
import { FormulaEditorField } from "@/components/fields/formula-editor-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { TextField } from "@/components/fields/text-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { NumberField } from "@/components/ui/number-input";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { FormulaTemplateSchema } from "@/lib/schemas/formula-template-schema";
import { useFormContext } from "react-hook-form";

const categoryChoices = [
  { value: "BaseRate", label: "Base Rate" },
  { value: "DistanceBased", label: "Distance Based" },
  { value: "WeightBased", label: "Weight Based" },
  { value: "DimensionalWeight", label: "Dimensional Weight" },
  { value: "FuelSurcharge", label: "Fuel Surcharge" },
  { value: "Accessorial", label: "Accessorial" },
  { value: "TimeBasedRate", label: "Time Based Rate" },
  { value: "ZoneBased", label: "Zone Based" },
  { value: "Custom", label: "Custom" },
];

export function FormulaTemplateForm() {
  const { control } = useFormContext<FormulaTemplateSchema>();

  return (
    <Tabs defaultValue="general" className="w-full">
      <TabsList className="grid w-full grid-cols-3">
        <TabsTrigger value="general">General</TabsTrigger>
        <TabsTrigger value="formula">Formula</TabsTrigger>
        <TabsTrigger value="settings">Settings</TabsTrigger>
      </TabsList>

      <TabsContent value="general" className="space-y-4 mt-4">
        <FormGroup cols={2}>
          <FormControl>
            <TextField
              name="name"
              control={control}
              label="Name"
              placeholder="e.g., Weight-Based Rate Calculation"
              rules={{ required: "Name is required" }}
              description="A descriptive name for this formula template"
            />
          </FormControl>

          <FormControl>
            <SelectField
              name="category"
              control={control}
              label="Category"
              placeholder="Select category"
              options={categoryChoices}
              rules={{ required: "Category is required" }}
              description="Classification of the formula template"
            />
          </FormControl>
        </FormGroup>

        <FormControl>
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Describe what this formula calculates and when it should be used..."
            rows={3}
            description="Optional description to help users understand the purpose"
          />
        </FormControl>

        <Separator />

        <div className="space-y-2">
          <Label>Template Options</Label>
          <FormGroup cols={2}>
            <FormControl>
              <CheckboxField
                name="isActive"
                control={control}
                label="Active"
                description="Enable this template for use"
              />
            </FormControl>

            <FormControl>
              <CheckboxField
                name="isDefault"
                control={control}
                label="Default Template"
                description="Use as default for this category"
              />
            </FormControl>
          </FormGroup>
        </div>
      </TabsContent>

      <TabsContent value="formula" className="space-y-4 mt-4">
        <FormControl>
          <FormulaEditorField
            name="expression"
            control={control}
            label="Formula Expression"
            placeholder="e.g., if(hasHazmat, weight * 0.15 + 200, weight * 0.10)"
            height="300px"
            enableAutocomplete
            enableValidation
            rules={{ required: "Expression is required" }}
            description="Write your formula using variables, functions, and operators. Press Ctrl+Space for autocomplete."
          />
        </FormControl>

        <div className="rounded-lg border p-4 bg-muted/30">
          <h4 className="text-sm font-medium mb-2">Quick Reference</h4>
          <div className="grid grid-cols-2 gap-4 text-xs">
            <div>
              <p className="font-medium mb-1">Common Variables:</p>
              <ul className="space-y-0.5 text-muted-foreground">
                <li>• <code className="bg-background px-1">weight</code> - Shipment weight</li>
                <li>• <code className="bg-background px-1">distance</code> - Travel distance</li>
                <li>• <code className="bg-background px-1">hasHazmat</code> - Hazmat indicator</li>
              </ul>
            </div>
            <div>
              <p className="font-medium mb-1">Common Functions:</p>
              <ul className="space-y-0.5 text-muted-foreground">
                <li>• <code className="bg-background px-1">if(cond, true, false)</code></li>
                <li>• <code className="bg-background px-1">min(a, b, ...)</code></li>
                <li>• <code className="bg-background px-1">max(a, b, ...)</code></li>
                <li>• <code className="bg-background px-1">round(num, decimals)</code></li>
              </ul>
            </div>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="settings" className="space-y-4 mt-4">
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              name="minRate"
              control={control}
              label="Minimum Rate"
              placeholder="0.00"
              formattedOptions={{
                style: "currency",
                currency: "USD",
              }}
              description="Optional minimum rate constraint"
            />
          </FormControl>

          <FormControl>
            <NumberField
              name="maxRate"
              control={control}
              label="Maximum Rate"
              placeholder="0.00"
              formattedOptions={{
                style: "currency",
                currency: "USD",
              }}
              description="Optional maximum rate constraint"
            />
          </FormControl>
        </FormGroup>

        <FormControl>
          <TextField
            name="outputUnit"
            control={control}
            label="Output Unit"
            placeholder="USD"
            description="Currency or unit for the calculated rate"
          />
        </FormControl>

        <div className="rounded-lg border p-4 bg-muted/30">
          <p className="text-sm text-muted-foreground">
            <strong>Note:</strong> Rate constraints are optional. If specified, calculated
            rates will be clamped to stay within the min/max range. Leave blank for no
            constraints.
          </p>
        </div>
      </TabsContent>
    </Tabs>
  );
}
