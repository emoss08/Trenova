import { useUploadWithProgress } from "@/hooks/use-upload-with-progress";
import { describeApiError } from "@/lib/api-error-message";
import type { InboundMessageDetail, TriageAction, TriageReplyTemplate } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { inboxReferenceQuery } from "@/lib/queries/inbox";
import {
  fileFromDocument,
  filesKey,
  overBudget,
  planUploads,
  remainingBytes,
  sameFiles,
  totalBytes,
  uploadTarget,
  withFile,
  withoutFile,
  type ReplyFile,
} from "@/lib/triage/reply-files";
import type { Document } from "@trenova/shared/types/document";
import { useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { LayersThree01Icon, PlusIcon, Send01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useMemo, useReducer, useState } from "react";
import { PersonAvatar } from "../../person-avatar";
import { useAnnouncePresence, useMessagePresence } from "../../use-triage-presence";
import { useTriageActions } from "../../use-triage-actions";
import {
  canSend,
  composerReducer,
  initialComposer,
  type ComposerState,
  type ReplyOutcome,
} from "../composer-state";
import { useReaderPanel, useStepSubmit } from "../reader-panel";
import {
  AttachMenu,
  RefusedFileChip,
  ReplyFileChip,
  UploadingChip,
} from "./reply-files";
import { StepBody, StepError, StepFooter } from "./step-parts";

/** How long typing pauses before the draft is saved for the team. */
const SAVE_AFTER_MS = 1200;
const PREVIEW_LENGTH = 46;
const REPLY_PREFIX = /^re:\s/i;

type Reply = InboundMessageDetail["replies"][number];

/** The newest reply that went out or was refused; drafts are the composer's own. */
function latestOutcome(replies: readonly Reply[]): ReplyOutcome | null {
  for (let index = replies.length - 1; index >= 0; index -= 1) {
    if (replies[index].status !== "Draft") {
      return replies[index];
    }
  }

  return null;
}

function replySubject(t: TranslateFn, message: InboundMessageDetail): string {
  const subject = message.subject.trim();
  if (subject === "") {
    return t("Re: your email");
  }

  return REPLY_PREFIX.test(subject) ? subject : t("Re: {0}", subject);
}

const NO_FILES: readonly ReplyFile[] = [];

type RefusedUpload = { key: string; name: string; reason: string };

/** The files a reply carries, the ones on their way up, and the ones refused before upload. */
function ReplyFiles({
  files,
  uploads,
  refused,
  onRemove,
  onCancelUpload,
  onRetryUpload,
  onDismissRefusal,
}: {
  files: readonly ReplyFile[];
  uploads: ReturnType<typeof useUploadWithProgress>["uploads"];
  refused: readonly RefusedUpload[];
  onRemove?: (documentId: string) => void;
  onCancelUpload: (id: string) => void;
  onRetryUpload: (id: string) => void;
  onDismissRefusal: (key: string) => void;
}) {
  const t = useT();
  const pending = uploads.filter((upload) => upload.status !== "success");
  if (files.length === 0 && pending.length === 0 && refused.length === 0) {
    return null;
  }

  return (
    <div role="list" aria-label={t("Files")} className="flex flex-wrap gap-2">
      {files.map((file) => (
        <ReplyFileChip key={file.documentId} file={file} onRemove={onRemove} />
      ))}
      {pending.map((upload) => (
        <UploadingChip
          key={upload.id}
          upload={upload}
          onCancel={onCancelUpload}
          onRetry={onRetryUpload}
        />
      ))}
      {refused.map((refusal) => (
        <RefusedFileChip
          key={refusal.key}
          name={refusal.name}
          reason={refusal.reason}
          onDismiss={() => onDismissRefusal(refusal.key)}
        />
      ))}
    </div>
  );
}

/** What the files still uploading will add when they land. */
function inFlightBytes(uploads: ReturnType<typeof useUploadWithProgress>["uploads"]): number {
  return uploads
    .filter((upload) => upload.status !== "success" && upload.status !== "error")
    .reduce((sum, upload) => sum + upload.file.size, 0);
}

/** Someone else is writing a reply to this message right now. */
function WritingToo({ messageId }: { messageId: string }) {
  const rt = useRichT();
  const writers = useMessagePresence(messageId).filter((member) => member.activity === "writing");
  const first = writers[0];
  if (first === undefined) {
    return null;
  }

  return (
    <div
      role="status"
      className="bg-warning-subtle text-warning-subtle-foreground animate-rise flex items-center gap-2 rounded-md px-2.5 py-2 text-sm"
    >
      <PersonAvatar name={first.name} size="xs" present />
      <span className="min-w-0 flex-1 truncate">
        {rt(
          "<b>{0}</b> is writing a reply too",
          { b: (chunks) => <span className="font-semibold">{chunks}</span> },
          first.name.split(/\s+/)[0] ?? first.name,
        )}
      </span>
      <span aria-hidden className="flex gap-0.75">
        <span className="animate-breathe size-1 rounded-full bg-current" />
        <span className="animate-breathe size-1 rounded-full bg-current [animation-delay:150ms]" />
        <span className="animate-breathe size-1 rounded-full bg-current [animation-delay:300ms]" />
      </span>
    </div>
  );
}

function TemplatesMenu({
  messageId,
  body,
  subject,
  onPick,
}: {
  messageId: string;
  body: string;
  subject: string;
  onPick: (text: string, template: TriageReplyTemplate) => void;
}) {
  const t = useT();
  const actions = useTriageActions();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const templates = useQuery({
    ...inboxReferenceQuery(queries.inbox.templates()),
    enabled: open,
  });
  const list = templates.data ?? [];
  const rendered = useQueries({
    queries: list.map((template) => ({
      ...queries.inbox.renderedTemplate(template.id, messageId),
      enabled: open,
      staleTime: 60_000,
    })),
  });

  const pick = (template: TriageReplyTemplate) => {
    setError(null);
    void queryClient
      .fetchQuery({ ...queries.inbox.renderedTemplate(template.id, messageId), staleTime: 60_000 })
      .then(
        (text) => {
          onPick(text, template);
          setOpen(false);
        },
        (failure: unknown) => setError(describeApiError(failure)),
      );
  };

  const save = () => {
    setSaving(true);
    setError(null);
    actions
      .saveTemplate(subject.replace(REPLY_PREFIX, "").slice(0, 60) || t("Reply"), body)
      .then(
        () => setSaved(true),
        (failure: unknown) => setError(describeApiError(failure)),
      )
      .finally(() => setSaving(false));
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) {
          setSaved(false);
          setError(null);
        }
      }}
    >
      <PopoverTrigger
        render={<Button size="sm" variant="ghost" className="data-popup-open:bg-surface-hover" />}
      >
        <LayersThree01Icon aria-hidden className="size-3" />
        {t("Templates")}
      </PopoverTrigger>
      <PopoverContent side="top" align="start" className="w-80 gap-0 p-1">
        <div className="text-foreground-subtle px-2 pt-1 pb-1.5 text-xs">
          {t("Fill in from this message")}
        </div>
        <div className="flex max-h-72 flex-col overflow-y-auto">
          {templates.isLoading && (
            <span className="text-foreground-subtle px-2 py-1.5 text-sm">{t("Loading…")}</span>
          )}
          {templates.isSuccess && list.length === 0 && (
            <span className="text-foreground-subtle px-2 py-1.5 text-sm">
              {t("No templates yet. Save a reply as one below.")}
            </span>
          )}
          {list.map((template, index) => {
            const filled = rendered[index]?.data ?? template.body;
            const firstLine = filled.split("\n")[0] ?? "";
            return (
              <button
                key={template.id}
                type="button"
                onClick={() => pick(template)}
                className="hover:bg-surface-hover ui-focus-ring flex flex-col items-start gap-0.5 rounded-md px-2 py-1.5 text-left"
              >
                <span className="text-sm font-medium">{template.name}</span>
                <span className="text-foreground-subtle w-full truncate text-xs">
                  {firstLine.length > PREVIEW_LENGTH
                    ? t("{0}…", firstLine.slice(0, PREVIEW_LENGTH))
                    : firstLine}
                </span>
              </button>
            );
          })}
        </div>
        {error !== null && <p className="text-danger px-2 py-1 text-xs">{error}</p>}
        <div className="border-border-subtle mt-1 border-t pt-1">
          <button
            type="button"
            disabled={body.trim() === "" || saving || saved}
            onClick={save}
            className="hover:bg-surface-hover ui-focus-ring flex h-7 w-full items-center gap-2 rounded-md px-2 text-sm disabled:opacity-50"
          >
            <PlusIcon aria-hidden className="size-3" />
            {saved ? t("Saved for the team") : t("Save this as a template")}
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function RefusedRetry({
  state,
  onSend,
  disabled,
}: {
  state: Extract<ComposerState, { phase: "failed" }>;
  onSend: (to: string) => void;
  disabled: boolean;
}) {
  const t = useT();
  const [address, setAddress] = useState("");
  const suggested = state.failure.suggestedAddress;

  if (suggested !== null) {
    return (
      <StepFooter note={t("To {0}", suggested)}>
        <Button size="sm" disabled={disabled} onClick={() => onSend(suggested)}>
          {t("Send to {0}@ instead", suggested.split("@")[0] ?? suggested)}
        </Button>
      </StepFooter>
    );
  }

  return (
    <form
      className="flex items-center gap-1.5"
      onSubmit={(event) => {
        event.preventDefault();
        if (address.trim() !== "") {
          onSend(address.trim());
        }
      }}
    >
      <Input
        type="email"
        value={address}
        onChange={(event) => setAddress(event.target.value)}
        placeholder={t("Another address at the sender")}
        aria-label={t("Send to another address")}
        className="h-7 text-sm"
        inputContainerClassName="min-w-0 flex-1"
      />
      <Button size="sm" type="submit" disabled={disabled || address.trim() === ""}>
        {t("Send there")}
      </Button>
    </form>
  );
}

/**
 * Writing back from the source's own address: the words (a template filled
 * in from the message, or Nova's draft), a warning when someone else is
 * writing too, Send, and — when the provider refuses — why, with the address
 * to try instead. Typing stays in this component, so nothing outside it
 * redraws on a keystroke; the draft is saved for the team after a pause.
 */
export function ReplyComposer({
  message,
  action,
}: {
  message: InboundMessageDetail;
  /** The step this answers; null for a reply the message did not ask for. */
  action: TriageAction | null;
}) {
  const t = useT();
  const rt = useRichT();
  const actions = useTriageActions();
  const panel = useReaderPanel();
  const drafting = panel.state.drafting.reply;
  const draftError = panel.state.errors.reply;
  const draft = message.replyDraft;
  const latest = latestOutcome(message.replies);
  const [state, dispatch] = useReducer(composerReducer, latest, initialComposer);
  const [body, setBody] = useState(draft?.body ?? "");
  const [savedBody, setSavedBody] = useState(draft?.body ?? "");
  const [templateId, setTemplateId] = useState<string | null>(draft?.templateId ?? null);
  const [novaDrafted, setNovaDrafted] = useState(draft?.draftedByNova ?? false);
  const [files, setFiles] = useState<readonly ReplyFile[]>(draft?.attachments ?? NO_FILES);
  const [savedFiles, setSavedFiles] = useState<readonly ReplyFile[]>(
    draft?.attachments ?? NO_FILES,
  );
  const [refusedUploads, setRefusedUploads] = useState<readonly RefusedUpload[]>([]);
  const [saveFailed, setSaveFailed] = useState(false);
  const [focused, setFocused] = useState(false);
  const draftKey = draft === null ? null : `${draft.id}:${draft.updatedAt}`;
  const [syncedKey, setSyncedKey] = useState(draftKey);
  const subject = draft?.subject || replySubject(t, message);

  if (draftKey !== syncedKey) {
    setSyncedKey(draftKey);
    if (draft !== null && draft.body !== savedBody) {
      setBody(draft.body);
      setSavedBody(draft.body);
      setTemplateId(draft.templateId);
      setNovaDrafted(draft.draftedByNova);
    }
    if (draft !== null && !sameFiles(draft.attachments, savedFiles)) {
      setFiles(draft.attachments);
      setSavedFiles(draft.attachments);
    }
  }

  const attachables = useQuery(queries.inbox.replyAttachables(message.id));
  const limitBytes = attachables.data?.limitBytes ?? null;
  const target = uploadTarget(message);
  const attachablesKey = queries.inbox.replyAttachables(message.id).queryKey;
  const onUploaded = useCallback((result: unknown) => {
    const document = result as Document;
    setFiles((current) => withFile(current, fileFromDocument(document)));
  }, []);
  const upload = useUploadWithProgress({
    resourceId: target.resourceId,
    resourceType: target.resourceType,
    invalidateQueryKey: attachablesKey,
    onSuccess: onUploaded,
  });
  const uploading = upload.uploads.some(
    (state) => state.status !== "success" && state.status !== "error",
  );

  const addFiles = (chosen: File[]) => {
    if (limitBytes === null) {
      upload.uploadFiles(chosen);
      return;
    }
    const plan = planUploads(
      chosen,
      remainingBytes(files, limitBytes) - inFlightBytes(upload.uploads),
    );
    if (plan.refused.length > 0) {
      setRefusedUploads((current) => [
        ...current,
        ...plan.refused.map(({ file, remainingBytes: left }) => ({
          key: `${file.name}:${file.size}:${file.lastModified}`,
          name: file.name,
          reason: t(
            "{0} is {1}. A reply can carry {2} in all, and {3} is left.",
            file.name,
            formatFileSize(file.size),
            formatFileSize(limitBytes),
            formatFileSize(left),
          ),
        })),
      ]);
    }
    if (plan.accepted.length > 0) {
      upload.uploadFiles(plan.accepted);
    }
  };

  const toggleFile = (file: ReplyFile) => {
    setFiles((current) =>
      current.some((existing) => existing.documentId === file.documentId)
        ? withoutFile(current, file.documentId)
        : withFile(current, file),
    );
  };

  const removeFile = (documentId: string) => {
    setFiles((current) => withoutFile(current, documentId));
  };

  useAnnouncePresence(focused ? message.id : null, "writing");

  const latestKey = latest === null ? null : `${latest.id}:${latest.status}`;
  const [seenLatest, setSeenLatest] = useState(latestKey);
  if (latestKey !== seenLatest) {
    setSeenLatest(latestKey);
    dispatch({ type: "synced", latest });
  }

  useEffect(() => {
    if (drafting) {
      dispatch({ type: "draftRequested" });
    } else if (draftError !== null) {
      dispatch({ type: "draftFailed", error: draftError });
    } else {
      dispatch({ type: "draftReady" });
    }
  }, [drafting, draftError]);

  const writing = state.phase === "writing" || state.phase === "failed";
  const fileIds = filesKey(files);
  const unsavedFiles = fileIds !== filesKey(savedFiles);
  useEffect(() => {
    if (!writing || body.trim() === "" || (body === savedBody && !unsavedFiles)) {
      return;
    }
    const timer = setTimeout(() => {
      setSavedBody(body);
      setSavedFiles(files);
      actions
        .saveReplyDraft(
          message.id,
          subject,
          body,
          files.map((file) => file.documentId),
        )
        .then(
          () => setSaveFailed(false),
          () => setSaveFailed(true),
        );
    }, SAVE_AFTER_MS);
    return () => clearTimeout(timer);
  }, [writing, body, savedBody, unsavedFiles, files, actions, message.id, subject]);

  const flushDraft = useLatestCallback(() => {
    if (writing && body.trim() !== "" && (body !== savedBody || unsavedFiles)) {
      actions
        .saveReplyDraft(
          message.id,
          subject,
          body,
          files.map((file) => file.documentId),
        )
        .catch(() => undefined);
    }
  });
  useEffect(() => flushDraft, [flushDraft]);

  const tooLarge = limitBytes !== null && overBudget(files, limitBytes);
  const send = (to: string | null) => {
    if (!canSend(state, body) || tooLarge || uploading) {
      return;
    }
    dispatch({ type: "send", to });
    actions
      .sendReply({
        messageId: message.id,
        subject,
        body,
        toAddress: to,
        templateId,
        draftedByNova: novaDrafted,
        documentIds: files.map((file) => file.documentId),
      })
      .then(
        (reply) => dispatch({ type: "sendSettled", reply }),
        (failure: unknown) => dispatch({ type: "sendErrored", error: describeApiError(failure) }),
      );
  };

  const ready = canSend(state, body) && state.phase === "writing" && !tooLarge && !uploading;
  useStepSubmit(action?.id ?? "reply", ready ? () => send(null) : null);

  const sentReply = useMemo(
    () =>
      state.phase === "sent"
        ? message.replies.find((reply) => reply.id === state.replyId)
        : undefined,
    [state, message.replies],
  );

  if (state.phase === "sent") {
    const sentFiles = sentReply?.attachments ?? files;
    return (
      <StepBody className="gap-1.5">
        <p className="text-foreground-muted text-base leading-relaxed whitespace-pre-wrap">
          {sentReply?.body ?? body}
        </p>
        {sentFiles.length > 0 && (
          <div role="list" aria-label={t("Files")} className="flex flex-wrap gap-2">
            {sentFiles.map((file) => (
              <ReplyFileChip key={file.documentId} file={file} />
            ))}
          </div>
        )}
      </StepBody>
    );
  }

  const mailbox = message.mailbox?.address ?? "";
  const to = state.phase === "sending" && state.to !== null ? state.to : message.fromAddress;
  const sending = state.phase === "sending";
  const error = state.phase === "writing" || state.phase === "failed" ? state.error : null;

  return (
    <StepBody>
      <WritingToo messageId={message.id} />
      {state.phase === "drafting" ? (
        <div className="bg-sunken border-border-subtle flex min-h-37.5 flex-col gap-2 rounded-lg border px-2.75 py-2.5">
          <TextShimmer as="span" className="text-base">
            {t("Looking it up and drafting…")}
          </TextShimmer>
        </div>
      ) : (
        <Textarea
          autoFocus
          minRows={6}
          maxRows={18}
          value={body}
          disabled={sending}
          aria-label={t("Reply")}
          placeholder={t("Write to {0}", message.fromAddress)}
          className="rounded-lg px-2.75 py-2.5 text-base leading-relaxed md:text-base"
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onChange={(event) => setBody(event.target.value)}
        />
      )}
      <ReplyFiles
        files={files}
        uploads={upload.uploads}
        refused={refusedUploads}
        onRemove={sending ? undefined : removeFile}
        onCancelUpload={upload.cancelUpload}
        onRetryUpload={upload.retryUpload}
        onDismissRefusal={(key) =>
          setRefusedUploads((current) => current.filter((refusal) => refusal.key !== key))
        }
      />
      {tooLarge && limitBytes !== null && (
        <StepError>
          {t(
            "These files add up to {0}. A reply can carry {1} in all, so remove one to send it.",
            formatFileSize(totalBytes(files)),
            formatFileSize(limitBytes),
          )}
        </StepError>
      )}
      {state.phase === "failed" && (
        <StepError>
          {rt(
            "<b>Didn't send.</b> {0}",
            { b: (chunks) => <span className="font-semibold">{chunks}</span> },
            state.failure.failureText,
          )}
        </StepError>
      )}
      {error !== null && <StepError>{error}</StepError>}
      {state.phase === "failed" ? (
        <RefusedRetry state={state} disabled={!canSend(state, body)} onSend={send} />
      ) : (
        <StepFooter
          note={
            sending
              ? t("Sending to {0}…", to)
              : saveFailed
                ? t("Couldn't save the draft — it is still here")
                : uploading
                  ? t("Uploading — Send waits for the files")
                  : files.length > 0 && limitBytes !== null
                    ? t(
                        "{0, plural, one {# file} other {# files}} · {1} of {2}",
                        files.length,
                        formatFileSize(totalBytes(files)),
                        formatFileSize(limitBytes),
                      )
                    : mailbox === ""
                      ? t("To {0}", to)
                      : t("From {0} to {1}", mailbox, to)
          }
        >
          {state.phase === "writing" && (
            <AttachMenu
              messageId={message.id}
              files={files}
              limitBytes={limitBytes}
              disabled={sending}
              onToggle={toggleFile}
              onUpload={addFiles}
            />
          )}
          {state.phase === "writing" && (
            <TemplatesMenu
              messageId={message.id}
              body={body}
              subject={subject}
              onPick={(text, template) => {
                setBody(text);
                setTemplateId(template.id);
                setNovaDrafted(false);
              }}
            />
          )}
          {action?.novaCanDo === true && state.phase === "writing" && (
            <Button size="sm" variant="ghost" onClick={panel.draftReply}>
              {t("Have Nova draft it")}
            </Button>
          )}
          <Button
            size="sm"
            disabled={!ready}
            isLoading={sending}
            loadingText={t("Sending…")}
            className={cn(state.phase === "drafting" && "opacity-50")}
            onClick={() => send(null)}
          >
            <Send01Icon aria-hidden className="size-3" />
            {t("Send")}
          </Button>
        </StepFooter>
      )}
    </StepBody>
  );
}
