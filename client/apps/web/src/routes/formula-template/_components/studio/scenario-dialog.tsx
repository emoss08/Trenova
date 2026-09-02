import { TestDataEditor } from "@/components/formula-editor/test-data-editor";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatCurrency } from "@trenova/shared/lib/utils";
import {
  formulaTestCaseInputSchema,
  type FormulaTestCase,
  type FormulaTestCaseInput,
  type VariableDefinitionInput,
} from "@trenova/shared/types/formula-template";
import { FlaskConicalIcon, SparklesIcon } from "lucide-react";
import { useEffect, useState } from "react";

export type ScenarioDraft = {
  name: string;
  description: string;
  variables: Record<string, unknown>;
  expectedAmount: string;
  tolerance: string;
};

/** Inputs and result carried over from a preview the author wants to pin. */
export type ScenarioPrefill = {
  variables: Record<string, unknown>;
  result: number | null;
};

function draftFromCase(
  testCase: FormulaTestCase | null,
  prefill: ScenarioPrefill | null = null,
): ScenarioDraft {
  if (!testCase) {
    return {
      name: "",
      description: "",
      variables: prefill ? { ...prefill.variables } : {},
      expectedAmount: prefill?.result != null ? String(prefill.result) : "",
      tolerance: "0.01",
    };
  }

  return {
    name: testCase.name,
    description: testCase.description,
    variables: { ...testCase.variables },
    expectedAmount: String(testCase.expectedAmount),
    tolerance: String(testCase.tolerance),
  };
}

type ScenarioDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editing: FormulaTestCase | null;
  /** When creating, start from these inputs and this expected result. */
  prefill?: ScenarioPrefill | null;
  currentSample?: ScenarioPrefill;
  schemaId?: string;
  customVariables?: VariableDefinitionInput[];
  isSaving: boolean;
  onSave: (input: FormulaTestCaseInput) => void;
};

export function ScenarioDialog({
  open,
  onOpenChange,
  editing,
  prefill = null,
  currentSample,
  schemaId = "shipment",
  customVariables = [],
  isSaving,
  onSave,
}: ScenarioDialogProps) {
  const [draft, setDraft] = useState<ScenarioDraft>(() => draftFromCase(editing, prefill));
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setDraft(draftFromCase(editing, prefill));
      setError(null);
    }
  }, [open, editing, prefill]);

  const handleUseCurrentSample = () => {
    if (!currentSample) return;
    setDraft((prev) => ({
      ...prev,
      variables: { ...currentSample.variables },
      expectedAmount:
        currentSample.result != null ? String(currentSample.result) : prev.expectedAmount,
    }));
  };

  const handleSave = () => {
    // The same schema the service parses; a scenario the dialog accepts is one
    // the server accepts, name length and all.
    const parsed = formulaTestCaseInputSchema.safeParse({
      name: draft.name.trim(),
      description: draft.description,
      variables: draft.variables,
      expectedAmount: draft.expectedAmount === "" ? Number.NaN : draft.expectedAmount,
      tolerance: draft.tolerance === "" ? 0.01 : draft.tolerance,
    });

    if (!parsed.success) {
      const first = parsed.error.issues[0];
      const field = first?.path[0];
      const label =
        field === "expectedAmount"
          ? "Expected charge"
          : field === "tolerance"
            ? "Tolerance"
            : field === "name"
              ? "Name"
              : "Scenario";
      const message = first?.message?.includes("NaN")
        ? "must be a number"
        : (first?.message ?? "is invalid");
      setError(`${label}: ${message}`);
      return;
    }

    onSave(parsed.data);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FlaskConicalIcon className="size-4" />
            {editing ? "Edit Scenario" : "New Scenario"}
          </DialogTitle>
          <DialogDescription>
            A scenario pins the charge this formula must produce for a known set of inputs. It
            re-runs on demand and must pass before the template can be approved.
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[60vh]">
          <div className="space-y-3 p-0.5">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label htmlFor="scenario-name" className="text-xs">
                  Name
                </Label>
                <Input
                  id="scenario-name"
                  value={draft.name}
                  onChange={(event) => setDraft((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="500 mile hazmat load"
                  className="h-8"
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="scenario-description" className="text-xs">
                  Description
                </Label>
                <Input
                  id="scenario-description"
                  value={draft.description}
                  onChange={(event) =>
                    setDraft((prev) => ({ ...prev, description: event.target.value }))
                  }
                  placeholder="Optional"
                  className="h-8"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label htmlFor="scenario-expected" className="text-xs">
                  Expected charge ($)
                </Label>
                <Input
                  id="scenario-expected"
                  inputMode="decimal"
                  value={draft.expectedAmount}
                  onChange={(event) =>
                    setDraft((prev) => ({ ...prev, expectedAmount: event.target.value }))
                  }
                  placeholder="1250.00"
                  className="h-8 font-mono"
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="scenario-tolerance" className="text-xs">
                  Tolerance ($)
                </Label>
                <Input
                  id="scenario-tolerance"
                  inputMode="decimal"
                  value={draft.tolerance}
                  onChange={(event) =>
                    setDraft((prev) => ({ ...prev, tolerance: event.target.value }))
                  }
                  placeholder="0.01"
                  className="h-8 font-mono"
                />
              </div>
            </div>

            {currentSample && (
              <Button
                type="button"
                variant="outline"
                size="xs"
                onClick={handleUseCurrentSample}
                className="gap-1.5"
              >
                <SparklesIcon className="size-3" />
                Use current sample data
                {currentSample.result != null && ` (${formatCurrency(currentSample.result)})`}
              </Button>
            )}

            <div className="space-y-1">
              <Label className="text-xs">Input values</Label>
              <TestDataEditor
                values={draft.variables}
                onChange={(variables) => setDraft((prev) => ({ ...prev, variables }))}
                schemaId={schemaId}
                customVariables={customVariables}
              />
            </div>

            {error && <p className="text-destructive text-xs">{error}</p>}
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={handleSave}
            isLoading={isSaving}
            loadingText="Saving..."
          >
            {editing ? "Save Scenario" : "Add Scenario"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
