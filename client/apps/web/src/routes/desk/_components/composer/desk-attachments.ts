import type { ComposerAttachment } from "@/components/assistant/composer";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useMemo, useRef, useState } from "react";

/** The most files one message may carry; the server refuses more. */
export const MAX_ATTACHMENTS = 5;
/** The largest file the Desk takes, in megabytes. */
export const MAX_ATTACHMENT_MB = 25;

const READABLE_EXTENSIONS = new Set([
  "pdf",
  "png",
  "jpg",
  "jpeg",
  "heic",
  "tif",
  "tiff",
  "csv",
  "xlsx",
  "xls",
  "docx",
  "txt",
  "eml",
]);

export function extensionOf(name: string): string {
  const dot = name.lastIndexOf(".");
  return dot === -1 ? "" : name.slice(dot + 1).toLowerCase();
}

/** A file on the message as the composer draws it: what is uploading, ready, refused or being scanned. */
export type DeskAttachment = ComposerAttachment & {
  /** Refused before it was sent anywhere: nothing to retry, only to remove. */
  refused?: boolean;
  /** A scan still arriving from a Capture device. */
  scan?: { device: string; pages: number };
};

/** Where a message's files go: the thread's own uploads, or files held for a thread not yet started. */
export type AttachmentSource = {
  attachments: readonly ComposerAttachment[];
  attachFiles: (files: File[]) => void;
  removeAttachment: (id: string) => void;
  retryAttachment?: (id: string) => void;
};

/**
 * Files picked before there is a conversation to upload them to. They are
 * held as picked and handed to the conversation the message starts, which
 * uploads them before the question goes.
 */
export function useHeldAttachments(): AttachmentSource & {
  files: () => File[];
  clear: () => void;
} {
  const [held, setHeld] = useState<Array<{ id: string; file: File }>>([]);
  const counter = useRef(0);

  const attachments = useMemo<ComposerAttachment[]>(
    () =>
      held.map(({ id, file }) => ({
        id,
        name: file.name,
        size: file.size,
        status: "ready",
        progress: 1,
        contentType: file.type,
        file,
      })),
    [held],
  );

  return {
    attachments,
    attachFiles: useCallback((files: File[]) => {
      setHeld((current) => [
        ...current,
        ...files.map((file) => {
          counter.current += 1;
          return { id: `held-${counter.current}`, file };
        }),
      ]);
    }, []),
    removeAttachment: useCallback(
      (id: string) => setHeld((current) => current.filter((item) => item.id !== id)),
      [],
    ),
    files: useCallback(() => held.map((item) => item.file), [held]),
    clear: useCallback(() => setHeld([]), []),
  };
}

/** Why a file cannot go with the message, before it is uploaded. */
function refusal(file: File, t: ReturnType<typeof useT>): string | null {
  const extension = extensionOf(file.name);
  if (!READABLE_EXTENSIONS.has(extension)) {
    return t("Desk can't read .{0} files", extension || "?");
  }
  if (file.size > MAX_ATTACHMENT_MB * 1024 * 1024) {
    return t("Too large · max {0} MB", MAX_ATTACHMENT_MB);
  }
  return null;
}

/** How much of each end of a PDF is read to look for its encryption entry. */
const PDF_PROBE_BYTES = 64 * 1024;

/**
 * Whether a PDF is locked with a password. A locked PDF names its
 * encryption dictionary in the trailer, at the end of the file or, for a
 * PDF laid out for the web, near its start; both ends are read, never the
 * whole file.
 */
export async function isPasswordProtectedPdf(file: Blob): Promise<boolean> {
  try {
    const head = file.slice(0, PDF_PROBE_BYTES);
    const tail = file.slice(Math.max(0, file.size - PDF_PROBE_BYTES));
    const decoder = new TextDecoder("latin1");
    const text =
      decoder.decode(await head.arrayBuffer()) + decoder.decode(await tail.arrayBuffer());
    return /\/Encrypt\s*(\d+\s+\d+\s+R|<<)/u.test(text);
  } catch {
    return false;
  }
}

export type DeskAttachments = ReturnType<typeof useDeskAttachments>;

/**
 * The files on a message as the composer shows them. Files the Desk cannot
 * read, or that are too large, are refused on the spot and shown as refused
 * rather than uploaded and turned away; past five a note says how many did
 * not fit.
 */
/** Files arriving some other way than an upload, such as a scan from Capture. */
export type ExtraAttachments = {
  items: readonly DeskAttachment[];
  remove: (id: string) => void;
  owns: (id: string) => boolean;
  clear?: () => void;
};

export function useDeskAttachments(source: AttachmentSource, extra?: ExtraAttachments | null) {
  const t = useT();
  const [refused, setRefused] = useState<DeskAttachment[]>([]);
  const [note, setNote] = useState<string | null>(null);
  const counter = useRef(0);

  const extraItems = extra?.items;
  const items = useMemo<DeskAttachment[]>(
    () => [...source.attachments, ...refused, ...(extraItems ?? [])],
    [extraItems, refused, source.attachments],
  );

  const add = useCallback(
    (picked: FileList | File[]) => {
      const files = Array.from(picked);
      const room = Math.max(0, MAX_ATTACHMENTS - items.length);
      const taken = files.slice(0, room);
      setNote(
        files.length > room
          ? t("Up to {0} files per message · {1} not added", MAX_ATTACHMENTS, files.length - room)
          : null,
      );
      const accepted: File[] = [];
      const turnedAway: DeskAttachment[] = [];
      for (const file of taken) {
        const reason = refusal(file, t);
        if (reason === null) {
          accepted.push(file);
          continue;
        }
        counter.current += 1;
        turnedAway.push({
          id: `refused-${counter.current}`,
          name: file.name,
          size: file.size,
          status: "error",
          progress: 0,
          error: reason,
          refused: true,
        });
      }
      if (turnedAway.length > 0) {
        setRefused((current) => [...current, ...turnedAway]);
      }
      if (accepted.length === 0) {
        return;
      }
      // A PDF locked with a password would upload and then be unreadable,
      // so it is turned away here, before anything is sent.
      void Promise.all(
        accepted.map((file) =>
          extensionOf(file.name) === "pdf" ? isPasswordProtectedPdf(file) : Promise.resolve(false),
        ),
      ).then((locked) => {
        const open = accepted.filter((_, index) => !locked[index]);
        const shut = accepted.filter((_, index) => locked[index]);
        if (shut.length > 0) {
          setRefused((current) => [
            ...current,
            ...shut.map((file) => {
              counter.current += 1;
              return {
                id: `refused-${counter.current}`,
                name: file.name,
                size: file.size,
                status: "error" as const,
                progress: 0,
                error: t("Password-protected · Desk can't open it"),
                refused: true,
              };
            }),
          ]);
        }
        if (open.length > 0) {
          source.attachFiles(open);
        }
      });
    },
    [items.length, source, t],
  );

  const remove = useCallback(
    (id: string) => {
      setNote(null);
      if (extra?.owns(id)) {
        extra.remove(id);
        return;
      }
      if (refused.some((item) => item.id === id)) {
        setRefused((current) => current.filter((item) => item.id !== id));
        return;
      }
      source.removeAttachment(id);
    },
    [extra, refused, source],
  );

  const clearExtra = extra?.clear;
  const clear = useCallback(() => {
    setRefused([]);
    setNote(null);
    clearExtra?.();
  }, [clearExtra]);

  const uploading = items.some((item) => item.status === "uploading" || item.scan !== undefined);
  const ready = items.filter((item) => item.status === "ready");
  const failed = items.filter((item) => item.status === "error").length;

  return {
    items,
    add,
    remove,
    retry: source.retryAttachment,
    clear,
    note,
    uploading,
    ready,
    failed,
    full: items.length >= MAX_ATTACHMENTS,
  };
}
