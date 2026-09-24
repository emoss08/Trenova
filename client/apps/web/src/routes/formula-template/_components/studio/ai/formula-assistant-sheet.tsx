import type { PageBinding, PageRequest } from "@/components/assistant/message-thread";
import { PageAssistant } from "@/components/assistant/page-assistant";
import { describeApiError } from "@/lib/api-error-message";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantArtifact } from "@/types/assistant";
import {
  pageDraftEditSchema,
  type FormulaProposal,
  type PageDraft,
  type PageDraftEdit,
  type PricedScenario,
} from "@/types/page-draft";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { VariableDefinition } from "@trenova/shared/types/formula-template";
import { AlertTriangleIcon, CheckCircle2Icon, CheckIcon, PlusIcon, XIcon } from "lucide-react";
import { nanoid } from "nanoid";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { proposalForEditor } from "./formula-draft";
import { acceptableScenarios, scenarioAmount, scenarioToTestCaseInput } from "./proposed-scenarios";

type FormulaAssistantSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  templateId: string | null;
  /** What the editor holds, read when a question is sent. */
  readDraft: () => PageDraft;
  onInsert: (result: { expression: string; variableDefinitions: VariableDefinition[] }) => void;
  /** Something the studio asks for the person, such as explaining the expression. */
  request: PageRequest | null;
  onRequestSent: () => void;
};

/**
 * The formula assistant beside the editor. It is the shared conversation
 * about this template: it reads the expression and variables on screen, prices
 * sample loads with the formula engine, and proposes a formula the person
 * inserts into the editor. Nothing is saved until they save the template.
 */
export function FormulaAssistantSheet({
  open,
  onOpenChange,
  templateId,
  readDraft,
  onInsert,
  request,
  onRequestSent,
}: FormulaAssistantSheetProps) {
  const t = useT();

  // A template not yet saved has a conversation of its own for as long as the
  // studio is open.
  const [unsavedKey] = useState(() => nanoid());
  const [proposal, setProposal] = useState<{ key: string; value: FormulaProposal } | null>(null);

  const showProposal = useCallback((key: string, edit: PageDraftEdit) => {
    if (edit.action === "propose_formula" && edit.formula) {
      setProposal({ key, value: edit.formula });
    }
  }, []);

  const page = useMemo<PageBinding>(
    () => ({
      surface: "formula",
      readDraft,
      onDraftEdit: (edit) => showProposal(nanoid(), edit),
    }),
    [readDraft, showProposal],
  );

  const onOpenArtifact = useCallback(
    (artifact: AssistantArtifact) => {
      const edit = pageDraftEditSchema.safeParse(artifact.payload);
      if (artifact.kind === "draft_edit" && edit.success) {
        showProposal(artifact.id, edit.data);
      }
    },
    [showProposal],
  );

  const openThread = useCallback(
    () => apiService.formulaTemplateService.openAssistantThread(templateId),
    [templateId],
  );

  const insert = (value: FormulaProposal) => {
    onInsert(proposalForEditor(value));
    toast.success(t("Formula inserted into the editor"), {
      description: t("Review and test it before saving."),
    });
    setProposal(null);
    onOpenChange(false);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex w-full flex-col gap-0 sm:w-[480px] sm:max-w-[480px]"
      >
        <SheetHeader className="border-b pb-3">
          <SheetTitle className="flex items-center gap-2">
            <AssistMark className="size-4" />
            {t("Formula assistant")}
          </SheetTitle>
          <SheetDescription>
            {t(
              "Describe how this template should price a shipment, or ask about the formula in the editor. A formula it proposes lands in the editor only when you insert it, and nothing is saved until you save the template.",
            )}
          </SheetDescription>
        </SheetHeader>

        <PageAssistant
          className="min-h-0 flex-1"
          conversationKey={["formula", templateId ?? unsavedKey]}
          open={openThread}
          page={page}
          pageRequest={request}
          onPageRequestSent={onRequestSent}
          onOpenArtifact={onOpenArtifact}
          header={
            proposal ? (
              <FormulaProposalCard
                key={proposal.key}
                proposal={proposal.value}
                templateId={templateId}
                onInsert={() => insert(proposal.value)}
                onDismiss={() => setProposal(null)}
              />
            ) : null
          }
        />
      </SheetContent>
    </Sheet>
  );
}

/**
 * A formula the assistant proposed: the expression, the variables it needs,
 * how it works, and what the engine charged for the sample loads it tried.
 * Every amount here is the engine's.
 */
function FormulaProposalCard({
  proposal,
  templateId,
  onInsert,
  onDismiss,
}: {
  proposal: FormulaProposal;
  templateId: string | null;
  onInsert: () => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [added, setAdded] = useState<ReadonlySet<string>>(() => new Set());

  const addScenario = useMutation({
    mutationFn: (scenario: PricedScenario) => {
      if (!templateId) {
        throw new Error(t("Save the template before adding scenarios"));
      }
      return apiService.formulaTemplateService.createTestCase(
        templateId,
        scenarioToTestCaseInput(scenario),
      );
    },
    onSuccess: async (_created, scenario) => {
      setAdded((previous) => new Set(previous).add(scenario.name));
      if (templateId) {
        await queryClient.invalidateQueries({
          queryKey: queries.formulaTemplate.testCases(templateId).queryKey,
        });
      }
    },
    onError: (error, scenario) => {
      toast.error(t("Could not add “{0}”", scenario.name), {
        description: describeApiError(error),
      });
    },
  });

  const pending = acceptableScenarios(proposal.scenarios).filter(
    (scenario) => !added.has(scenario.name),
  );

  const addAll = async () => {
    let count = 0;
    for (const scenario of pending) {
      try {
        await addScenario.mutateAsync(scenario);
        count += 1;
      } catch {
        break;
      }
    }
    if (count > 0) {
      toast.success(t("{0} scenarios added", count));
    }
  };

  const checkedAmount = Number(proposal.check.result);

  return (
    <div className="flex max-h-[45vh] flex-col gap-3 overflow-y-auto rounded-md border p-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-medium">{t("Proposed formula")}</p>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={t("Dismiss proposal")}
          onClick={onDismiss}
        >
          <XIcon className="size-3.5" />
        </Button>
      </div>

      <pre className="bg-muted overflow-x-auto rounded-md border p-2.5 font-mono text-xs whitespace-pre-wrap">
        {proposal.expression}
      </pre>

      {proposal.variables.length > 0 && (
        <div className="overflow-hidden rounded-md border">
          {proposal.variables.map((variable) => (
            <div
              key={variable.name}
              className="flex items-center justify-between gap-2 border-b px-3 py-1.5 text-xs last:border-b-0"
            >
              <span className="font-mono">{variable.name}</span>
              <span className="flex items-center gap-1.5">
                {variable.defaultValue !== undefined && variable.defaultValue !== null && (
                  <span className="text-muted-foreground">= {String(variable.defaultValue)}</span>
                )}
                <Badge variant="neutral" appearance="outline">
                  {variable.type}
                </Badge>
              </span>
            </div>
          ))}
        </div>
      )}

      {proposal.explanation !== "" && (
        <p className="text-sm leading-relaxed whitespace-pre-wrap">{proposal.explanation}</p>
      )}

      <Alert variant={proposal.check.valid ? "success" : "warning"} size="sm">
        {proposal.check.valid ? <CheckCircle2Icon /> : <AlertTriangleIcon />}
        <AlertDescription>
          {proposal.check.valid
            ? Number.isFinite(checkedAmount) && proposal.check.result !== ""
              ? t("Validated against sample data — result {0}", formatCurrency(checkedAmount))
              : t("Validated against sample data")
            : t("Validation warning: {0}", proposal.check.error)}
        </AlertDescription>
      </Alert>

      {proposal.scenarios.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center justify-between gap-2">
            <p className="text-muted-foreground text-xs font-medium">{t("Proposed scenarios")}</p>
            {templateId && pending.length > 1 && (
              <Button
                type="button"
                variant="ghost"
                size="xs"
                onClick={() => void addAll()}
                disabled={addScenario.isPending}
              >
                <PlusIcon className="size-3" />
                {t("Add all")}
              </Button>
            )}
          </div>
          <div className="overflow-hidden rounded-md border">
            {proposal.scenarios.map((scenario) => {
              const amount = scenarioAmount(scenario);
              const isAdded = added.has(scenario.name);
              return (
                <div
                  key={scenario.name}
                  className="flex items-start justify-between gap-3 border-b px-3 py-2 text-xs last:border-b-0"
                >
                  <div className="flex min-w-0 flex-col gap-0.5">
                    <p className="font-medium">{scenario.name}</p>
                    {scenario.description !== "" && (
                      <p className="text-muted-foreground">{scenario.description}</p>
                    )}
                    {amount !== null ? (
                      <p className="font-mono tabular-nums">
                        {t("Expects {0}", formatCurrency(amount))}
                      </p>
                    ) : (
                      <p className="text-warning-foreground flex items-center gap-1">
                        <AlertTriangleIcon className="size-3 shrink-0" />
                        {scenario.error || t("Could not be priced")}
                      </p>
                    )}
                  </div>
                  <Button
                    type="button"
                    variant={isAdded ? "ghost" : "outline"}
                    size="xs"
                    disabled={!templateId || amount === null || isAdded || addScenario.isPending}
                    onClick={() => void addScenario.mutateAsync(scenario).catch(() => undefined)}
                    className="shrink-0"
                  >
                    {isAdded ? <CheckIcon className="size-3" /> : <PlusIcon className="size-3" />}
                    {isAdded ? t("Added") : t("Add")}
                  </Button>
                </div>
              );
            })}
          </div>
          <p className="text-2xs text-muted-foreground">
            {templateId
              ? t(
                  "Expected amounts were computed by the engine from the proposed formula. Scenarios you add will run against the template as you edit it.",
                )
              : t("Save the template first, then add these as scenarios.")}
          </p>
        </div>
      )}

      <div className="flex items-center gap-2">
        <Button type="button" size="sm" onClick={onInsert}>
          {t("Insert into editor")}
        </Button>
      </div>
    </div>
  );
}
