import { PageAssistant } from "@/components/assistant/page-assistant";
import { pageAssistantThreadKey } from "@/components/assistant/page-assistant-thread-key";
import type { PageBinding, PageRequest } from "@/components/assistant/use-thread-model";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { assistant } from "@/lib/queries/assistant";
import type { PageThread } from "@/types/assistant";
import {
  pageDraftEditSchema,
  type FormulaProposal,
  type PageDraftAction,
  type PageDraftEdit,
} from "@/types/page-draft";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { toast } from "@trenova/shared/components/ui/toast";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useMemo, useState, type ReactNode } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { applyProposal, formulaDraftFromEditor, type ProposalTarget } from "./formula-draft";
import {
  ProposalCard,
  proposalTargetOf,
  type ProposalFormValues,
  type ProposalState,
} from "./proposal-card";

type ProposalRecord = { state: ProposalState; before: ProposalTarget | null };

const TARGET_KEYS = ["name", "description", "source"] as const;

const PROPOSE_ACTION: Record<"tariff" | "library", PageDraftAction> = {
  tariff: "formula_propose",
  library: "formula_library_propose",
};

/** The file, as Nova's composer names it: read-only, read at every send. */
function FilePageChip({ kind }: { kind: "tariff" | "library" }) {
  const t = useT();
  const { control } = useFormContext<ProposalFormValues>();
  const name = useWatch({ control, name: "name" });
  const source = useWatch({ control, name: "source" }) ?? "";
  const lines = source.split("\n").filter((line) => line.trim() !== "").length;

  return (
    <span className="fs-page-chip" title={t("Nova reads the editor as it is, saved or not")}>
      <span className="text-dsk-muted flex h-7 min-w-0 items-center gap-1.5 rounded-badge pr-2 text-sm">
        <span className="dk-pc-ic">
          <DeskIcon name="file" size={12} stroke={2} />
        </span>
        <span className="min-w-0 truncate">
          {name?.trim() || (kind === "library" ? t("Library editor") : t("Tariff editor"))}
        </span>
        <span className="text-foreground-subtle shrink-0 text-xs">
          {lines === 0 ? t("empty") : t("{0, plural, one {# line} other {# lines}}", lines)}
        </span>
      </span>
    </span>
  );
}

function ReplyProposal({
  kind,
  threadId,
  artifactId,
  record,
  onApply,
  onUndo,
  onDismiss,
}: {
  kind: "tariff" | "library";
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
    if (!parsed.success || parsed.data.action !== PROPOSE_ACTION[kind]) return null;
    return parsed.data.formula ?? null;
  }, [artifactId, data, kind]);

  if (isPending) return <Skeleton className="h-40 w-full rounded-lg" />;
  if (!proposal) return null;

  return (
    <ProposalCard
      kind={kind}
      proposal={proposal}
      state={record?.state ?? "pending"}
      before={record?.before ?? null}
      onApply={onApply}
      onUndo={onUndo}
      onDismiss={onDismiss}
    />
  );
}

export type TariffNovaPaneProps = {
  kind: "tariff" | "library";
  /** The saved template's or library's ID; null while it is being created. */
  recordId: string | null;
  /** Names the conversation about a file not yet created, for the life of the page. */
  unsavedKey: string;
  openThread: (recordId: string | null) => Promise<PageThread>;
  request: PageRequest | null;
  onRequestSent: (key: string) => void;
  onWorkingChange: (busy: boolean) => void;
  ask: (text?: string) => void;
  /** Offered once, in the empty conversation. */
  presets: readonly string[];
  emptyTitle: string;
  emptyBody: ReactNode;
};

/**
 * Nova in the Studio's rail: the conversation about this file, in the Desk's compact thread
 * with a composer that names only the editor and the model. It reads the file at every send;
 * what it proposes lands in the editor the moment it arrives (unless the server's check refused
 * it), with the diff, the checks and the ratings under the reply and one click to undo it.
 * Nothing is saved until the person saves.
 */
export function TariffNovaPane({
  kind,
  recordId,
  unsavedKey,
  openThread,
  request,
  onRequestSent,
  onWorkingChange,
  ask,
  presets,
  emptyTitle,
  emptyBody,
}: TariffNovaPaneProps) {
  const t = useT();
  const rules = useRules();
  const form = useFormContext<ProposalFormValues>();
  const [records, setRecords] = useState<ReadonlyMap<string, ProposalRecord>>(() => new Map());
  const conversationKey = useMemo(
    () => [kind === "library" ? "formula-library" : "formula", recordId ?? unsavedKey] as const,
    [kind, recordId, unsavedKey],
  );

  const open = useCallback(() => openThread(recordId), [openThread, recordId]);
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
      kind,
      ...(recordId ? { recordId } : {}),
      name: values.name ?? "",
      description: values.description ?? "",
      source: values.source ?? "",
    });
  }, [form, kind, recordId]);

  const setRecord = useCallback((artifactId: string, record: ProposalRecord) => {
    setRecords((current) => new Map(current).set(artifactId, record));
  }, []);

  const restore = useCallback(
    (snapshot: ProposalTarget) => {
      for (const key of TARGET_KEYS) {
        form.setValue(key, snapshot[key], { shouldDirty: true, shouldValidate: true });
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

  // A change Nova makes lands in the editor as it arrives; the card under the reply shows the
  // diff and undoes it. One the server's check refused stays in the thread to read, never in the
  // editor.
  const onDraftEdit = useCallback(
    (edit: PageDraftEdit, artifactId: string) => {
      if (edit.action !== PROPOSE_ACTION[kind] || !edit.formula) return;
      const proposal = edit.formula;
      if (!proposal.check.ok) {
        setRecord(artifactId, { state: "pending", before: null });
        const problem = proposal.check.diagnostics.find((finding) => finding.severity === "error");
        toast.warning(t("Nova's change was not applied: it does not check."), {
          description: problem ? translateMessage(problem.message, t) : undefined,
        });
        return;
      }
      const current = proposalTargetOf(form.getValues());
      apply(
        artifactId,
        applyProposal(rules, current, proposal),
        current.source.trim() === ""
          ? kind === "library"
            ? t("Nova drafted the library")
            : t("Nova drafted the tariff")
          : t("Nova changed the source"),
      );
    },
    [apply, form, kind, rules, setRecord, t],
  );

  const page = useMemo<PageBinding>(
    () => ({
      surface: kind === "library" ? "formula_library" : "formula",
      readDraft,
      onDraftEdit,
    }),
    [kind, readDraft, onDraftEdit],
  );

  return (
    <PageAssistant
      className="h-full min-h-0"
      chrome="none"
      conversationKey={conversationKey}
      open={open}
      page={page}
      pageRequest={request}
      onPageRequestSent={onRequestSent}
      onWorkingChange={onWorkingChange}
      thread={{
        composerKit: "page",
        composerChip: <FilePageChip kind={kind} />,
        disclaimer: t("Nova can be wrong. Every amount it shows is the rules module's."),
        emptyState: (dock) => (
          <div className="dk-c flex flex-col items-center gap-2 px-4 pt-10 pb-4 text-center">
            <NovaBadge size="lg" />
            <h3 className="text-xl font-semibold">{emptyTitle}</h3>
            <p className="text-foreground-muted max-w-xs text-sm leading-relaxed">{emptyBody}</p>
            <div className="dk-sugg">
              {presets.map((preset) => (
                <Button
                  key={preset}
                  type="button"
                  variant="bare"
                  size="bare"
                  disabled={dock.busy}
                  onClick={() => ask(preset)}
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
              kind={kind}
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

function ReplyProposals({
  kind,
  threadId,
  messageId,
  records,
  onApply,
  onUndo,
  onDismiss,
}: {
  kind: "tariff" | "library";
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
          kind={kind}
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
