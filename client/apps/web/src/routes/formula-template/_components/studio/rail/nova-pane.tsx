import { PageAssistant } from "@/components/assistant/page-assistant";
import { pageAssistantThreadKey } from "@/components/assistant/page-assistant-thread-key";
import type { PageBinding } from "@/components/assistant/use-thread-model";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import { pageDraftEditSchema, type FormulaProposal, type PageDraftEdit } from "@/types/page-draft";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { applyProposal, formulaDraftFromEditor, type ProposalTarget } from "../ai/formula-draft";
import { ProposalCard, proposalTargetOf, type ProposalState } from "../ai/proposal-card";
import { useStudio } from "../studio-context";

type ProposalRecord = { state: ProposalState; before: ProposalTarget | null };

const TARGET_KEYS = [
  "name",
  "description",
  "type",
  "expression",
  "variableDefinitions",
  "breakdownDefinitions",
  "minCharge",
  "maxCharge",
  "roundingMode",
  "roundingPrecision",
] as const;

/** The editor, as Nova's composer names it: read-only, read at every send. */
function FormulaPageChip() {
  const t = useT();
  const { analysis } = useStudio();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const name = useWatch({ control, name: "name" });
  const inputs = useWatch({ control, name: "variableDefinitions" })?.length ?? 0;
  const expression = useWatch({ control, name: "expression" }) ?? "";
  const lines = expression.split("\n").filter((line) => line.trim() !== "").length;

  return (
    <span className="fs-page-chip" title={t("Nova reads the editor as it is, saved or not")}>
      <span className="text-dsk-muted flex h-7 min-w-0 items-center gap-1.5 rounded-badge pr-2 text-sm">
        <span className="dk-pc-ic">
          <DeskIcon name="file" size={12} stroke={2} />
        </span>
        <span className="min-w-0 truncate">{name?.trim() || t("Formula editor")}</span>
        <span className="text-foreground-subtle shrink-0 text-xs">
          {analysis.empty
            ? t("empty")
            : t(
                "{0, plural, one {# line} other {# lines}} · {1, plural, one {# input} other {# inputs}}",
                lines,
                inputs,
              )}
        </span>
      </span>
    </span>
  );
}

function ReplyProposals({
  threadId,
  messageId,
  records,
  onApply,
  onUndo,
  onDismiss,
}: {
  threadId: string;
  messageId: string;
  records: ReadonlyMap<string, ProposalRecord>;
  onApply: (artifactId: string, next: ProposalTarget) => void;
  onUndo: (artifactId: string) => void;
  onDismiss: (artifactId: string) => void;
}) {
  const { data: summary } = useQuery({
    ...assistant.artifacts(threadId)._ctx.summary,
    enabled: threadId !== "",
  });
  const edits = (summary?.results ?? []).filter(
    (artifact) => artifact.kind === "draft_edit" && artifact.messageId === messageId,
  );
  if (edits.length === 0) return null;

  return (
    <div className="dk-ec-after flex flex-col gap-2">
      {edits.map((artifact) => (
        <ReplyProposal
          key={artifact.id}
          threadId={threadId}
          artifactId={artifact.id}
          record={records.get(artifact.id)}
          onApply={(next) => onApply(artifact.id, next)}
          onUndo={() => onUndo(artifact.id)}
          onDismiss={() => onDismiss(artifact.id)}
        />
      ))}
    </div>
  );
}

function ReplyProposal({
  threadId,
  artifactId,
  record,
  onApply,
  onUndo,
  onDismiss,
}: {
  threadId: string;
  artifactId: string;
  record: ProposalRecord | undefined;
  onApply: (next: ProposalTarget) => void;
  onUndo: () => void;
  onDismiss: () => void;
}) {
  const { data, isPending } = useQuery({
    ...assistant.artifacts(threadId)._ctx.lineage(artifactId),
    staleTime: Number.POSITIVE_INFINITY,
  });
  const proposal = useMemo((): FormulaProposal | null => {
    const artifact = data?.results.find((item) => item.id === artifactId) ?? data?.results.at(-1);
    const parsed = pageDraftEditSchema.safeParse(artifact?.payload);
    if (!parsed.success || parsed.data.action !== "propose_formula") return null;
    return parsed.data.formula ?? null;
  }, [artifactId, data]);

  if (isPending) return <Skeleton className="h-40 w-full rounded-lg" />;
  if (!proposal) return null;

  return (
    <ProposalCard
      proposal={proposal}
      state={record?.state ?? "pending"}
      before={record?.before ?? null}
      onApply={onApply}
      onUndo={onUndo}
      onDismiss={onDismiss}
    />
  );
}

/**
 * Nova in the rail: the formula assistant's conversation about this
 * template, in the Desk's compact thread with a composer that names only the
 * editor and the model. It reads the editor at every send; what it proposes
 * lands in the editor the moment it arrives, with the diff under the reply
 * and one click to undo it. Nothing is saved until the person saves.
 */
export function NovaPane() {
  const t = useT();
  const {
    template,
    novaRequest,
    clearNovaRequest,
    setNovaBusy,
    analysis,
    askNova,
    unsavedNovaKey,
  } = useStudio();
  const form = useFormContext<FormulaTemplateFormValues>();
  const [records, setRecords] = useState<ReadonlyMap<string, ProposalRecord>>(() => new Map());
  const conversationKey = useMemo(
    () => ["formula", template?.id ?? unsavedNovaKey] as const,
    [template?.id, unsavedNovaKey],
  );

  const open = useCallback(
    () => apiService.formulaTemplateService.openAssistantThread(template?.id ?? null),
    [template?.id],
  );
  const opened = useQuery({
    queryKey: pageAssistantThreadKey(conversationKey),
    queryFn: () => open(),
    enabled: false,
    staleTime: Number.POSITIVE_INFINITY,
  });
  const threadId = opened.data?.thread.id ?? "";

  const readDraft = useCallback(() => {
    const values = form.getValues();
    return formulaDraftFromEditor({
      templateId: template?.id ?? null,
      schemaId: values.schemaId || "shipment",
      templateType: values.type ?? "FreightCharge",
      name: values.name ?? "",
      description: values.description ?? "",
      expression: values.expression ?? "",
      variables: values.variableDefinitions ?? [],
      breakdown: values.breakdownDefinitions ?? [],
      policy: {
        minCharge: values.minCharge,
        maxCharge: values.maxCharge,
        roundingMode: values.roundingMode,
        roundingPrecision: values.roundingPrecision,
      },
    });
  }, [form, template?.id]);

  const setRecord = useCallback((artifactId: string, record: ProposalRecord) => {
    setRecords((current) => new Map(current).set(artifactId, record));
  }, []);

  const restore = useCallback(
    (snapshot: ProposalTarget) => {
      for (const key of TARGET_KEYS) {
        form.setValue(key, snapshot[key] as never, { shouldDirty: true, shouldValidate: true });
      }
    },
    [form],
  );

  const undo = useCallback(
    (artifactId: string) => {
      const record = records.get(artifactId);
      if (record?.before) restore(record.before);
      setRecord(artifactId, { state: "pending", before: null });
    },
    [records, restore, setRecord],
  );

  const apply = useCallback(
    (artifactId: string, next: ProposalTarget, announce: string) => {
      const before = proposalTargetOf(form.getValues());
      restore(next);
      setRecord(artifactId, { state: "applied", before });
      toast(announce, {
        description: t("Test it, then save. Nothing is saved yet."),
        action: {
          label: t("Undo"),
          onClick: () => {
            restore(before);
            setRecord(artifactId, { state: "pending", before: null });
          },
        },
      });
    },
    [form, restore, setRecord, t],
  );

  // A change Nova makes lands in the editor as it arrives; the card under
  // the reply shows the diff and undoes it. One the engine could not run
  // stays in the thread for the person to read, never in the editor.
  const onDraftEdit = useCallback(
    (edit: PageDraftEdit, artifactId: string) => {
      if (edit.action !== "propose_formula" || !edit.formula) return;
      const proposal = edit.formula;
      if (!proposal.check.valid) {
        setRecord(artifactId, { state: "pending", before: null });
        toast.warning(t("Nova's change was not applied: the engine could not run it."), {
          description: proposal.check.error || undefined,
        });
        return;
      }
      const current = proposalTargetOf(form.getValues());
      apply(
        artifactId,
        applyProposal(current, proposal),
        current.expression.trim() === ""
          ? t("Nova drafted the formula")
          : t("Nova changed the formula"),
      );
    },
    [apply, form, setRecord, t],
  );

  const page = useMemo<PageBinding>(
    () => ({ surface: "formula", readDraft, onDraftEdit }),
    [readDraft, onDraftEdit],
  );

  const presets = analysis.empty
    ? [
        t("Per mile with an 18% fuel surcharge and a $250 minimum"),
        t("Per CWT, weight rounded up to the next 100 lb"),
        t("$75 per stop after the first two"),
      ]
    : [
        t("Explain this formula, term by term"),
        t("Add a weekend pickup fee"),
        t("Set a $250 minimum charge"),
        t("Suggest edge cases to test"),
      ];

  return (
    <PageAssistant
      className="h-full min-h-0"
      chrome="none"
      conversationKey={conversationKey}
      open={open}
      page={page}
      pageRequest={novaRequest}
      onPageRequestSent={clearNovaRequest}
      onWorkingChange={setNovaBusy}
      thread={{
        composerKit: "page",
        composerChip: <FormulaPageChip />,
        // The presets are offered once, in the empty state; typed out in the
        // composer they would loop a caret on a working screen.
        disclaimer: t("Nova can be wrong. Every amount it shows is the formula engine's."),
        emptyState: (dock) => (
          <div className="dk-c flex flex-col items-center gap-2 px-4 pt-10 pb-4 text-center">
            <NovaBadge size="lg" />
            <h3 className="text-xl font-semibold">
              {analysis.empty ? t("How should this price a load?") : t("What should we change?")}
            </h3>
            <p className="text-foreground-muted max-w-xs text-sm leading-relaxed">
              {t(
                "Nova reads the editor as it is and makes the changes you ask for — the formula, its inputs, the charge policy. Every change can be undone, and nothing is saved until you save.",
              )}
            </p>
            <div className="dk-sugg">
              {presets.map((preset) => (
                <Button
                  key={preset}
                  type="button"
                  variant="bare"
                  size="bare"
                  disabled={dock.busy}
                  onClick={() => askNova(preset)}
                >
                  {preset}
                </Button>
              ))}
            </div>
          </div>
        ),
        renderReplyExtras: (messageId) =>
          threadId ? (
            <ReplyProposals
              threadId={threadId}
              messageId={messageId}
              records={records}
              onApply={(artifactId, next) => apply(artifactId, next, t("Applied to the editor"))}
              onUndo={undo}
              onDismiss={(artifactId) =>
                setRecord(artifactId, { state: "dismissed", before: null })
              }
            />
          ) : null,
      }}
    />
  );
}
