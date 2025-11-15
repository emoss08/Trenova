/*
 * Formula Tester Component
 *
 * Interactive playground for testing formula expressions with sample data
 */

import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { Play, CheckCircle2, XCircle, Info } from "lucide-react";

import { FormulaEditorField } from "@/components/fields/formula-editor-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { formulaTemplateService, type TestFormulaResponse } from "@/services/formula-template-service";

interface TestFormData {
  expression: string;
}

interface VariableInput {
  name: string;
  value: string;
  type: "number" | "string" | "boolean";
}

const defaultVariables: VariableInput[] = [
  { name: "weight", value: "5000", type: "number" },
  { name: "distance", value: "250", type: "number" },
  { name: "hasHazmat", value: "false", type: "boolean" },
];

const exampleFormulas = [
  {
    name: "Weight-based with Hazmat",
    expression: "if(hasHazmat, weight * 0.15 + 200, weight * 0.10)",
    description: "Different rates for hazmat shipments",
  },
  {
    name: "Distance-based Tiered",
    expression: "if(distance < 100, 150, if(distance < 500, distance * 1.5, distance * 1.2))",
    description: "Tiered pricing based on distance",
  },
  {
    name: "Combined Calculation",
    expression: "weight * 0.10 + distance * 1.25 + if(hasHazmat, 200, 0)",
    description: "Combines weight, distance, and hazmat surcharge",
  },
];

export function FormulaTester() {
  const [variables, setVariables] = useState<VariableInput[]>(defaultVariables);
  const [testResult, setTestResult] = useState<TestFormulaResponse | null>(null);
  const [isTesting, setIsTesting] = useState(false);

  const { control, handleSubmit, setValue, watch } = useForm<TestFormData>({
    defaultValues: {
      expression: exampleFormulas[0].expression,
    },
  });

  const expression = watch("expression");

  const addVariable = () => {
    setVariables([...variables, { name: "", value: "", type: "number" }]);
  };

  const removeVariable = (index: number) => {
    setVariables(variables.filter((_, i) => i !== index));
  };

  const updateVariable = (index: number, field: keyof VariableInput, value: string) => {
    const updated = [...variables];
    updated[index] = { ...updated[index], [field]: value };
    setVariables(updated);
  };

  const parseVariableValue = (value: string, type: VariableInput["type"]): any => {
    if (type === "number") {
      const num = parseFloat(value);
      return isNaN(num) ? 0 : num;
    }
    if (type === "boolean") {
      return value.toLowerCase() === "true" || value === "1";
    }
    return value;
  };

  const onTest = async (data: TestFormData) => {
    setIsTesting(true);
    setTestResult(null);

    try {
      // Convert variables to the format expected by the API
      const variableValues: Record<string, any> = {};
      for (const variable of variables) {
        if (variable.name.trim()) {
          variableValues[variable.name] = parseVariableValue(variable.value, variable.type);
        }
      }

      const result = await formulaTemplateService.testFormula(
        data.expression,
        variableValues
      );

      setTestResult(result);

      if (result.success) {
        toast.success("Formula evaluated successfully!");
      } else {
        toast.error("Formula evaluation failed", {
          description: result.error,
        });
      }
    } catch (error) {
      toast.error("Failed to test formula", {
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setIsTesting(false);
    }
  };

  const loadExample = (example: typeof exampleFormulas[0]) => {
    setValue("expression", example.expression);
    toast.info("Example loaded", {
      description: example.description,
    });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Formula Tester</h2>
        <p className="text-muted-foreground">
          Test your formula expressions with sample data and see the results in real-time.
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Left Column - Editor and Variables */}
        <div className="space-y-6">
          {/* Formula Editor */}
          <Card>
            <CardHeader>
              <CardTitle>Formula Expression</CardTitle>
              <CardDescription>
                Write your formula using variables, functions, and operators
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit(onTest)} className="space-y-4">
                <FormulaEditorField
                  name="expression"
                  control={control}
                  label="Expression"
                  placeholder="Enter your formula here..."
                  height="250px"
                  enableAutocomplete
                  enableValidation
                  rules={{ required: "Expression is required" }}
                />

                <Button
                  type="submit"
                  disabled={isTesting || !expression?.trim()}
                  className="w-full"
                >
                  <Play className="mr-2 size-4" />
                  {isTesting ? "Testing..." : "Test Formula"}
                </Button>
              </form>
            </CardContent>
          </Card>

          {/* Variables */}
          <Card>
            <CardHeader>
              <CardTitle>Test Variables</CardTitle>
              <CardDescription>
                Define sample values for variables used in your formula
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {variables.map((variable, index) => (
                <div key={index} className="grid grid-cols-12 gap-2">
                  <div className="col-span-4">
                    <Input
                      placeholder="Variable name"
                      value={variable.name}
                      onChange={(e) => updateVariable(index, "name", e.target.value)}
                    />
                  </div>
                  <div className="col-span-4">
                    <Input
                      placeholder="Value"
                      value={variable.value}
                      onChange={(e) => updateVariable(index, "value", e.target.value)}
                    />
                  </div>
                  <div className="col-span-3">
                    <select
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background"
                      value={variable.type}
                      onChange={(e) =>
                        updateVariable(index, "type", e.target.value as VariableInput["type"])
                      }
                    >
                      <option value="number">Number</option>
                      <option value="string">String</option>
                      <option value="boolean">Boolean</option>
                    </select>
                  </div>
                  <div className="col-span-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeVariable(index)}
                    >
                      ×
                    </Button>
                  </div>
                </div>
              ))}

              <Button type="button" variant="outline" onClick={addVariable} className="w-full">
                + Add Variable
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* Right Column - Examples and Results */}
        <div className="space-y-6">
          {/* Example Formulas */}
          <Card>
            <CardHeader>
              <CardTitle>Example Formulas</CardTitle>
              <CardDescription>Click an example to load it into the editor</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {exampleFormulas.map((example, index) => (
                <button
                  key={index}
                  type="button"
                  onClick={() => loadExample(example)}
                  className="w-full text-left p-3 rounded-lg border hover:bg-accent transition-colors"
                >
                  <div className="font-medium">{example.name}</div>
                  <div className="text-sm text-muted-foreground mt-1">
                    {example.description}
                  </div>
                  <code className="text-xs bg-muted px-2 py-1 rounded mt-2 block">
                    {example.expression}
                  </code>
                </button>
              ))}
            </CardContent>
          </Card>

          {/* Test Result */}
          {testResult && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {testResult.success ? (
                    <CheckCircle2 className="size-5 text-green-600" />
                  ) : (
                    <XCircle className="size-5 text-red-600" />
                  )}
                  Test Result
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {testResult.success ? (
                  <>
                    <div>
                      <Label className="text-sm text-muted-foreground">Result</Label>
                      <div className="mt-1 p-3 bg-muted rounded-lg font-mono text-lg">
                        {JSON.stringify(testResult.result, null, 2)}
                      </div>
                    </div>

                    {testResult.resultType && (
                      <div>
                        <Label className="text-sm text-muted-foreground">Type</Label>
                        <div className="mt-1">
                          <Badge variant="secondary">{testResult.resultType}</Badge>
                        </div>
                      </div>
                    )}

                    {testResult.usedVariables && testResult.usedVariables.length > 0 && (
                      <div>
                        <Label className="text-sm text-muted-foreground">Variables Used</Label>
                        <div className="mt-1 flex flex-wrap gap-2">
                          {testResult.usedVariables.map((varName) => (
                            <Badge key={varName} variant="outline">
                              {varName}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                  </>
                ) : (
                  <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-lg border border-red-200 dark:border-red-900">
                    <div className="flex gap-2">
                      <Info className="size-5 text-red-600 flex-shrink-0 mt-0.5" />
                      <div>
                        <div className="font-medium text-red-900 dark:text-red-100">
                          Error
                        </div>
                        <div className="text-sm text-red-700 dark:text-red-300 mt-1">
                          {testResult.error}
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Info Card */}
          <Card>
            <CardHeader>
              <CardTitle>Available Features</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex items-start gap-2">
                <CheckCircle2 className="size-4 text-green-600 mt-0.5 flex-shrink-0" />
                <span>Real-time syntax validation as you type</span>
              </div>
              <div className="flex items-start gap-2">
                <CheckCircle2 className="size-4 text-green-600 mt-0.5 flex-shrink-0" />
                <span>Autocomplete for variables and functions (Ctrl+Space)</span>
              </div>
              <div className="flex items-start gap-2">
                <CheckCircle2 className="size-4 text-green-600 mt-0.5 flex-shrink-0" />
                <span>28+ built-in functions (math, arrays, conditionals)</span>
              </div>
              <div className="flex items-start gap-2">
                <CheckCircle2 className="size-4 text-green-600 mt-0.5 flex-shrink-0" />
                <span>Support for complex expressions with nested functions</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
