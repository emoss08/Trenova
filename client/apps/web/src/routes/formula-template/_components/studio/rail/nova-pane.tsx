import { PageAssistant, pageAssistantThreadKey } from "@/components/assistant/page-assistant";
import type { PageBinding } from "@/components/assistant/use-thread-model";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { pageDraftEditSchema, type FormulaProposal } from "@/types/page-draft";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { useCallback, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { formulaDraftFromEditor, type ProposalTarget } from "../ai/formula-draft";
import { ProposalCard, proposalTargetOf, type ProposalState } from "../ai/proposal-card";
import { useStudio } from "../studio-context";

type InsertRecord = { state: ProposalState; before: ProposalTarget | null };

const TARGET_KEYS = [
  "name",
  "description",
  "type",
  "expression",
  "variableDefinitions",
  "breakdownDefinitions",
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
    <span className="dk-pc" title={t("Nova reads the editor as it is, saved or not")}>
      <span className="text-dsk-muted flex h-7 min-w-0 items-center gap-1.5 rounded-full pr-2 text-sm">
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
  onInsert,
  onUndo,
  onDismiss,
}: {
  threadId: string;
  messageId: string;
  records: ReadonlyMap<string, InsertRecord>;
  onInsert: (artifactId: string, next: ProposalTarget) => void;
  onUndo: (artifactId: string) => void;
  onDismiss: (artifactId: string) => void;
}) {
  const { data: summary } = useQuery({
    ...queries.assistant.artifacts(threadId)._ctx.summary,
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
          onInsert={(next) => onInsert(artifact.id, next)}
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
  onInsert,
  onUndo,
  onDismiss,
}: {
  threadId: string;
  artifactId: string;
  record: InsertRecord | undefined;
  onInsert: (next: ProposalTarget) => void;
  onUndo: () => void;
  onDismiss: () => void;
}) {
  const { data, isPending } = useQuery({
    ...queries.assistant.artifacts(threadId)._ctx.lineage(artifactId),
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
      before={record?.before?.expression ?? null}
      onInsert={onInsert}
      onUndo={onUndo}
      onDismiss={onDismiss}
    />
  );
}

/**
 * Nova in the rail: the formula assistant's conversation about this
 * template, in the Desk's compact thread with a composer that names only the
 * editor and the model. It reads the editor at every send, and what it
 * proposes lands under the reply as a diff to insert or dismiss.
 */
export function NovaPane() {
  const t = useT();
  const { template, novaRequest, clearNovaRequest, setNovaBusy, analysis, askNova } = useStudio();
  const form = useFormContext<FormulaTemplateFormValues>();
  const [unsavedKey] = useState(() => nanoid());
  const [records, setRecords] = useState<ReadonlyMap<string, InsertRecord>>(() => new Map());
  const conversationKey = useMemo(
    () => ["formula", template?.id ?? unsavedKey] as const,
    [template?.id, unsavedKey],
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
      expression: values.expression ?? "",
      variables: values.variableDefinitions ?? [],
    });
  }, [form, template?.id]);

  const page = useMemo<PageBinding>(
    () => ({ surface: "formula", readDraft, onDraftEdit: () => undefined }),
    [readDraft],
  );

  const setRecord = useCallback((artifactId: string, record: InsertRecord) => {
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

  const insert = useCallback(
    (artifactId: string, next: ProposalTarget) => {
      const before = proposalTargetOf(form.getValues());
      restore(next);
      setRecord(artifactId, { state: "inserted", before });
      toast(t("Inserted into the editor. Test it, then save."), {
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

  const presets = analysis.empty
    ? [
        t("Per mile with an 18% fuel surcharge and a $250 minimum"),
        t("Per CWT, weight rounded up to the next 100 lb"),
        t("$75 per stop after the first two"),
      ]
    : [
        t("Explain this formula, term by term"),
        t("Add a weekend pickup fee"),
        t("Suggest edge cases to test"),
        t("Why did the short haul bill the minimum?"),
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
        presets,
        disclaimer: t("Nova can be wrong. Every amount it shows is the formula engine's."),
        emptyState: (dock) => (
          <div className="flex flex-col items-center gap-2 px-4 pt-10 pb-4 text-center">
            <NovaMark className="size-5.5" />
            <h3 className="text-xl font-semibold">
              {analysis.empty ? t("How should this price a load?") : t("What should we change?")}
            </h3>
            <p className="text-foreground-muted max-w-xs text-sm leading-relaxed">
              {t(
                "Nova reads the editor as it is, saved or not, and prices every draft with the formula engine. Nothing lands in the editor until you insert it.",
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
              onInsert={insert}
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
