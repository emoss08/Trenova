import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  FormulaTemplateFormValues,
  FormulaTestCase,
  FormulaTestCaseInput,
  RunTestCasesResponse,
  TestCaseCandidate,
  TestCaseResult,
} from "@trenova/shared/types/formula-template";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckCircle2Icon,
  FlaskConicalIcon,
  PencilIcon,
  PlayIcon,
  PlusIcon,
  Trash2Icon,
  XCircleIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";
import type { LivePreviewState } from "./use-live-preview";
import { ScenarioDialog } from "./scenario-dialog";

type StudioScenariosPaneProps = {
  templateId: string | null;
  preview: LivePreviewState;
};

function ScenarioRow({
  scenario,
  result,
  onEdit,
  onDelete,
}: {
  scenario: FormulaTestCase;
  result?: TestCaseResult;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div
      className={cn(
        "group flex items-center justify-between gap-2 rounded-md border px-3 py-2",
        result && (result.passed ? "border-emerald-500/40 bg-emerald-500/5" : "border-destructive/40 bg-destructive/5"),
      )}
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          {result &&
            (result.passed ? (
              <CheckCircle2Icon className="size-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
            ) : (
              <XCircleIcon className="text-destructive size-3.5 shrink-0" />
            ))}
          <span className="truncate text-sm font-medium">{scenario.name}</span>
        </div>
        <div className="text-muted-foreground text-xs">
          Expects {formatCurrency(scenario.expectedAmount)}
          {result && !result.passed && !result.error && (
            <span className="text-destructive"> — got {formatCurrency(result.actualAmount)}</span>
          )}
          {result?.error && <span className="text-destructive"> — {result.error}</span>}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button type="button" variant="ghost" size="icon-xs" onClick={onEdit}>
                <PencilIcon className="size-3" />
              </Button>
            }
          />
          <TooltipContent>Edit scenario</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                onClick={onDelete}
                className="hover:text-destructive"
              >
                <Trash2Icon className="size-3" />
              </Button>
            }
          />
          <TooltipContent>Delete scenario</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}

export function StudioScenariosPane({ templateId, preview }: StudioScenariosPaneProps) {
  const queryClient = useQueryClient();
  const { getValues } = useFormContext<FormulaTemplateFormValues>();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<FormulaTestCase | null>(null);
  const [pendingDelete, setPendingDelete] = useState<FormulaTestCase | null>(null);
  const [runResults, setRunResults] = useState<RunTestCasesResponse | null>(null);

  const { data: scenarios, isLoading } = useQuery({
    ...queries.formulaTemplate.testCases(templateId ?? ""),
    enabled: !!templateId,
  });

  const invalidate = useCallback(async () => {
    if (templateId) {
      await queryClient.invalidateQueries({
        queryKey: queries.formulaTemplate.testCases(templateId).queryKey,
      });
    }
  }, [queryClient, templateId]);

  const buildCandidate = useCallback((): TestCaseCandidate => {
    const values = getValues();
    return {
      expression: values.expression,
      variableDefinitions: values.variableDefinitions,
      breakdownDefinitions: values.breakdownDefinitions,
      minCharge: values.minCharge ?? null,
      maxCharge: values.maxCharge ?? null,
    };
  }, [getValues]);

  const runMutation = useMutation({
    mutationFn: () => {
      if (!templateId) throw new Error("Template not saved yet");
      return apiService.formulaTemplateService.runTestCases(templateId, buildCandidate());
    },
    onSuccess: (result) => {
      setRunResults(result);
      if (result.failed === 0) {
        toast.success(`All ${result.total} scenarios pass`);
      } else {
        toast.error(`${result.failed} of ${result.total} scenarios fail`);
      }
    },
    onError: () => {
      toast.error("Failed to run scenarios");
    },
  });

  const saveMutation = useMutation({
    mutationFn: async (input: FormulaTestCaseInput) => {
      if (!templateId) throw new Error("Template not saved yet");
      if (editing) {
        return apiService.formulaTemplateService.updateTestCase(templateId, editing.id, {
          ...input,
          version: editing.version,
        });
      }
      return apiService.formulaTemplateService.createTestCase(templateId, input);
    },
    onSuccess: async () => {
      toast.success(editing ? "Scenario updated" : "Scenario added");
      setDialogOpen(false);
      setEditing(null);
      setRunResults(null);
      await invalidate();
    },
    onError: () => {
      toast.error("Failed to save scenario");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (scenario: FormulaTestCase) => {
      if (!templateId) throw new Error("Template not saved yet");
      await apiService.formulaTemplateService.deleteTestCase(templateId, scenario.id);
    },
    onSuccess: async () => {
      toast.success("Scenario deleted");
      setPendingDelete(null);
      setRunResults(null);
      await invalidate();
    },
    onError: () => {
      toast.error("Failed to delete scenario");
    },
  });

  const resultsById = useMemo(() => {
    const map = new Map<string, TestCaseResult>();
    for (const result of runResults?.results ?? []) {
      map.set(result.testCaseId, result);
    }
    return map;
  }, [runResults]);

  const currentSample = useMemo(
    () => ({
      variables: preview.testValues,
      result: typeof preview.result?.result === "number" ? preview.result.result : null,
    }),
    [preview.testValues, preview.result],
  );

  if (!templateId) {
    return (
      <div className="text-muted-foreground flex h-full flex-col items-center justify-center gap-2 p-4 text-center text-sm">
        <FlaskConicalIcon className="size-8 opacity-40" />
        <span>Save the template first, then pin its behaviour with test scenarios.</span>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <FlaskConicalIcon className="text-muted-foreground size-4" />
          <span className="text-sm font-semibold">Scenarios</span>
          {runResults && (
            <Badge
              variant={runResults.failed === 0 ? "active" : "inactive"}
              className="text-2xs"
            >
              {runResults.passed}/{runResults.total} passing
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-1.5">
          <Button
            type="button"
            variant="outline"
            size="xs"
            onClick={() => {
              setEditing(null);
              setDialogOpen(true);
            }}
            className="gap-1"
          >
            <PlusIcon className="size-3" />
            Add
          </Button>
          <Button
            type="button"
            size="xs"
            onClick={() => runMutation.mutate()}
            isLoading={runMutation.isPending}
            loadingText="Running..."
            disabled={!scenarios || scenarios.length === 0}
            className="gap-1"
          >
            <PlayIcon className="size-3" />
            Run All
          </Button>
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="space-y-2 p-3">
          {isLoading && (
            <>
              <Skeleton className="h-12" />
              <Skeleton className="h-12" />
            </>
          )}

          {!isLoading && (scenarios?.length ?? 0) === 0 && (
            <div className="text-muted-foreground flex flex-col items-center gap-2 py-8 text-center text-sm">
              <FlaskConicalIcon className="size-8 opacity-40" />
              <span>
                No scenarios yet. Add one to pin what this formula must produce — approval
                requires every scenario to pass.
              </span>
            </div>
          )}

          {scenarios?.map((scenario) => (
            <ScenarioRow
              key={scenario.id}
              scenario={scenario}
              result={resultsById.get(scenario.id)}
              onEdit={() => {
                setEditing(scenario);
                setDialogOpen(true);
              }}
              onDelete={() => setPendingDelete(scenario)}
            />
          ))}
        </div>
      </ScrollArea>

      <ScenarioDialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open);
          if (!open) setEditing(null);
        }}
        editing={editing}
        currentSample={currentSample}
        isSaving={saveMutation.isPending}
        onSave={(input) => saveMutation.mutate(input)}
      />

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              Delete scenario &quot;{pendingDelete?.name}&quot;?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This scenario will no longer gate approval of the template.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              size="default"
              onClick={() => pendingDelete && deleteMutation.mutate(pendingDelete)}
              disabled={deleteMutation.isPending}
              isLoading={deleteMutation.isPending}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
