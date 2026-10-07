import { translate } from "@trenova/shared/i18n/runtime";
import type { AssistantMessageAttachment } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
} from "react";
import { DeskIcon } from "../desk-icons";
import {
  extensionOf,
  MAX_ATTACHMENT_MB,
  MAX_ATTACHMENTS,
  type DeskAttachment,
  type DeskAttachments,
} from "./desk-attachments";

const FILE_KINDS: Record<string, [string, "pdf" | "img" | "sheet" | "doc"]> = {
  pdf: ["PDF", "pdf"],
  png: ["PNG", "img"],
  jpg: ["JPG", "img"],
  jpeg: ["JPG", "img"],
  heic: ["HEIC", "img"],
  tif: ["TIFF", "img"],
  tiff: ["TIFF", "img"],
  csv: ["CSV", "sheet"],
  xlsx: ["XLSX", "sheet"],
  xls: ["XLS", "sheet"],
  docx: ["DOCX", "doc"],
  txt: ["TXT", "doc"],
  eml: ["EML", "doc"],
};

export function formatFileSize(bytes: number): string {
  return bytes < 1024 * 1024
    ? translate("{0} KB", Math.max(1, Math.round(bytes / 1024)))
    : translate("{0} MB", (bytes / 1024 / 1024).toFixed(1));
}

/** An object URL for an image file, revoked when the file goes. */
function usePreviewUrl(file: File | undefined): string | null {
  const url = useMemo(
    () => (file && file.type.startsWith("image/") ? URL.createObjectURL(file) : null),
    [file],
  );
  useEffect(
    () => () => {
      if (url) {
        URL.revokeObjectURL(url);
      }
    },
    [url],
  );
  return url;
}

/** A file's kind as a small tile, or the picture itself for an image picked here. */
export function DeskFileIcon({
  name,
  file,
  size = 32,
}: {
  name: string;
  file?: File;
  size?: number;
}) {
  const extension = extensionOf(name);
  const [label, kind] = FILE_KINDS[extension] ?? [
    extension.toUpperCase().slice(0, 4) || "FILE",
    "bad",
  ];
  const preview = usePreviewUrl(kind === "img" ? file : undefined);
  if (preview) {
    return (
      <span
        className="dk-fi dk-img"
        style={{ width: size, height: size, backgroundImage: `url(${preview})` }}
      />
    );
  }
  return (
    <span className={cn("dk-fi", `dk-k-${kind}`)} style={{ width: size, height: size }}>
      <span>{label}</span>
    </span>
  );
}

function statusOf(item: DeskAttachment): "uploading" | "ready" | "error" | "scanning" {
  return item.scan ? "scanning" : item.status;
}

/** One file on the message: its kind, name, progress or problem, and a way to take it off. */
function FileChip({
  item,
  onRemove,
  onRetry,
  onFinishScan,
}: {
  item: DeskAttachment;
  onRemove: (id: string) => void;
  onRetry?: (id: string) => void;
  onFinishScan?: (id: string) => void;
}) {
  const t = useT();
  const status = statusOf(item);
  if (item.scan) {
    const { device, pages } = item.scan;
    return (
      <div
        className="dk-fp dk-s-scanning"
        data-tip={
          pages > 0
            ? t("Pages are arriving from {0}", device)
            : t("Waiting for {0} · put the pages in the scanner", device)
        }
      >
        <span className="dk-fp-scan">
          <DeskIcon name="scanner" size={12} stroke={2} />
        </span>
        <span className="dk-fp-n">{pages > 0 ? t("Scanning") : t("Waiting for scanner")}</span>
        <span className="dk-fp-pages">
          {Array.from({ length: Math.min(pages, 4) }, (_, index) => (
            <i key={index} />
          ))}
        </span>
        {pages > 0 && (
          <span className="dk-fp-pc">{t("{0, plural, one {# page} other {# pages}}", pages)}</span>
        )}
        {pages > 0 && onFinishScan && (
          <button
            type="button"
            className="dk-fp-b dk-done"
            title={t("Done scanning")}
            aria-label={t("Done scanning")}
            onClick={() => onFinishScan(item.id)}
          >
            <DeskIcon name="check" size={11} stroke={2.4} />
          </button>
        )}
        <button
          type="button"
          className="dk-fp-b"
          title={t("Cancel scan")}
          aria-label={t("Cancel scan")}
          onClick={() => onRemove(item.id)}
        >
          <DeskIcon name="x" size={10} stroke={2.4} />
        </button>
      </div>
    );
  }
  const percent = Math.round((item.progress ?? 0) * (item.progress > 1 ? 1 : 100));
  const error = item.error ?? t("Upload failed");
  const tip =
    status === "error"
      ? error
      : status === "uploading"
        ? t("Uploading · {0}%", percent)
        : `${item.name} · ${formatFileSize(item.size)}`;
  return (
    <div
      className={cn("dk-fp", `dk-s-${status}`)}
      data-tip={tip}
      style={{ "--dk-p": Math.min(1, percent / 100) } as CSSProperties}
    >
      <DeskFileIcon name={item.name} file={item.file} size={18} />
      <span className="dk-fp-n">{item.name}</span>
      {status === "uploading" && <span className="dk-fp-pc">{percent}%</span>}
      {status === "error" && <span className="dk-fp-e">{error.split(" · ")[0]}</span>}
      {status === "error" && !item.refused && onRetry && (
        <button
          type="button"
          className="dk-fp-b"
          title={t("Try again")}
          aria-label={t("Try again")}
          onClick={() => onRetry(item.id)}
        >
          <DeskIcon name="replay" size={11} />
        </button>
      )}
      <button
        type="button"
        className="dk-fp-b"
        title={t("Remove")}
        aria-label={t("Remove {0}", item.name)}
        onClick={() => onRemove(item.id)}
      >
        <DeskIcon name="x" size={10} stroke={2.4} />
      </button>
    </div>
  );
}

/** How many chips show before the rest fold into "+N more". */
const SHOWN = 3;

/** The files on the message, in a row above the text. */
export function DeskAttachRow({
  attachments,
  onFinishScan,
  lead,
}: {
  attachments: DeskAttachments;
  onFinishScan?: (id: string) => void;
  lead?: ReactNode;
}) {
  const t = useT();
  const [more, setMore] = useState(false);
  const { items, note, failed } = attachments;
  const retryable = items.some((item) => item.status === "error" && !item.refused);
  if (items.length === 0 && !note && !lead) {
    return null;
  }
  const rest = items.slice(SHOWN);
  const chip = (item: DeskAttachment) => (
    <FileChip
      key={item.id}
      item={item}
      onRemove={attachments.remove}
      onRetry={attachments.retry}
      onFinishScan={onFinishScan}
    />
  );
  return (
    <div className="dk-ar">
      <div className="dk-ar-l">
        {lead}
        {items.slice(0, SHOWN).map(chip)}
        {rest.length > 0 && (
          <span className="dk-fp-more-w">
            <button
              type="button"
              className={cn(
                "dk-fp dk-more",
                rest.some((item) => item.status === "error") && "dk-has-err",
              )}
              onClick={() => setMore((value) => !value)}
            >
              {t("+{0} more", rest.length)}
            </button>
            {more && (
              <div className="dk-fp-pop" onMouseLeave={() => setMore(false)}>
                {rest.map(chip)}
              </div>
            )}
          </span>
        )}
        {(note || failed > 0) && (
          <span className="dk-ar-note">
            {note ??
              (!retryable
                ? failed === 1
                  ? t("Remove it to send the rest")
                  : t("{0} files can't be sent · remove them to send the rest", failed)
                : failed === 1
                  ? t("Retry or remove it to send with this file")
                  : t("{0} files can't be sent · retry or remove them", failed))}
          </span>
        )}
      </div>
    </div>
  );
}

/** Placeholders for the files being dragged over the composer, before they land. */
export function DeskDropGhosts({ count, taken }: { count: number; taken: number }) {
  const t = useT();
  const room = Math.max(0, MAX_ATTACHMENTS - taken);
  const shown = Math.min(Math.max(1, count), room);
  return (
    <div className="dk-ar dk-ghosts">
      <div className="dk-ar-l">
        {Array.from({ length: shown }, (_, index) => (
          <span
            key={index}
            className="dk-fp dk-ghost"
            style={{ animationDelay: `${index * 50}ms` }}
          >
            <span className="dk-fi" style={{ width: 18, height: 18 }} />
            <span className="dk-fp-n" />
          </span>
        ))}
        {room === 0 ? (
          <span className="dk-ar-note">
            {t("This message is full · {0} files max", MAX_ATTACHMENTS)}
          </span>
        ) : count > room ? (
          <span className="dk-ar-note">
            {t("Only {0} more will fit · up to {1} per message", room, MAX_ATTACHMENTS)}
          </span>
        ) : null}
      </div>
    </div>
  );
}

/** What the attach button opens: a file from this computer, or a scan from Capture. */
export function DeskAttachMenu({
  onPickFiles,
  onScan,
  scanDevice,
  onClose,
  children,
}: {
  onPickFiles: (files: FileList) => void;
  onScan?: () => void;
  scanDevice?: string | null;
  onClose: () => void;
  /** The Capture panel, when it is open in place of the menu. */
  children?: ReactNode;
}) {
  const t = useT();
  const inputRef = useRef<HTMLInputElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }
      // Focus inside the menu goes back to the button that opened it, rather
      // than to the page when the menu goes.
      if (rootRef.current?.contains(document.activeElement)) {
        rootRef.current.parentElement?.querySelector<HTMLElement>("[data-attach-toggle]")?.focus();
      }
      onClose();
    };
    const onPointer = (event: PointerEvent) => {
      const target = event.target as Element | null;
      if (rootRef.current?.contains(target) || target?.closest?.("[data-attach-toggle]")) {
        return;
      }
      onClose();
    };
    window.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      window.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [onClose]);

  // The menu opens on its first item, so the arrow keys work at once.
  const hasChildren = children !== undefined && children !== null;
  useEffect(() => {
    if (!hasChildren) {
      rootRef.current?.querySelector<HTMLElement>("[role='menuitem']")?.focus();
    }
  }, [hasChildren]);
  const onMenuKey = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    const items = [...(rootRef.current?.querySelectorAll<HTMLElement>("[role='menuitem']") ?? [])];
    const at = items.indexOf(document.activeElement as HTMLElement);
    const step = event.key === "ArrowDown" ? 1 : event.key === "ArrowUp" ? -1 : 0;
    if (step !== 0 && items.length > 0) {
      event.preventDefault();
      items[(at + step + items.length) % items.length]?.focus();
    }
  };

  if (children) {
    return (
      <div className="dk-am dk-cap-w" ref={rootRef}>
        {children}
      </div>
    );
  }

  return (
    <div className="dk-am" ref={rootRef} role="menu" onKeyDown={onMenuKey}>
      <input
        ref={inputRef}
        type="file"
        multiple
        hidden
        onChange={(event) => {
          if (event.target.files) {
            onPickFiles(event.target.files);
          }
          event.target.value = "";
          onClose();
        }}
      />
      <button type="button" role="menuitem" onClick={() => inputRef.current?.click()}>
        <DeskIcon name="plus" size={14} />
        <span>
          <b>{t("Upload from computer")}</b>
          <em>
            {t(
              "PDF, images, CSV, Excel · up to {0} files, {1} MB each",
              MAX_ATTACHMENTS,
              MAX_ATTACHMENT_MB,
            )}
          </em>
        </span>
      </button>
      {onScan && (
        <button type="button" role="menuitem" onClick={onScan}>
          <DeskIcon name="scanner" size={14} stroke={2} />
          <span>
            <b>{t("Scan from Capture")}</b>
            <em>
              {scanDevice
                ? t("Scan paper on {0} straight into this message", scanDevice)
                : t("Scan paper straight into this message")}
            </em>
          </span>
        </button>
      )}
      <div className="dk-am-f">
        <span className="dk-kbd">⌘U</span> {t("or drop files anywhere")}
      </div>
    </div>
  );
}

/** The whole Desk while files are dragged over it. */
export function DeskDropOverlay({
  show,
  hot,
  count,
}: {
  show: boolean;
  hot: boolean;
  count: number;
}) {
  const t = useT();
  return (
    <div className={cn("dk-dz", show && "dk-on", hot && "dk-hot")} aria-hidden={!show}>
      <div className="dk-dz-in">
        <div className="dk-dz-stack">
          <span />
          <span />
          <span />
        </div>
        <b>
          {hot
            ? t("Release to attach to your message")
            : count > 0
              ? t("{0, plural, one {Drop to attach # file} other {Drop to attach # files}}", count)
              : t("Drop to attach")}
        </b>
        <span>
          {t(
            "PDF, images, CSV or Excel · up to {0} files, {1} MB each",
            MAX_ATTACHMENTS,
            MAX_ATTACHMENT_MB,
          )}
        </span>
      </div>
    </div>
  );
}

/** The files a sent question carried, under it in the conversation. */
export function DeskMessageAttachments({
  attachments,
}: {
  attachments: readonly AssistantMessageAttachment[];
}) {
  if (attachments.length === 0) {
    return null;
  }
  return (
    <div className="dk-ma">
      {attachments.map((attachment) => (
        <span
          key={attachment.documentId}
          className="dk-fp dk-s-sent"
          data-tip={formatFileSize(attachment.fileSize ?? 0)}
        >
          <DeskFileIcon name={attachment.fileName} size={18} />
          <span className="dk-fp-n">{attachment.fileName}</span>
        </span>
      ))}
    </div>
  );
}

export type DragState = { on: boolean; hot: boolean; count: number };

const NOT_DRAGGING: DragState = { on: false, hot: false, count: 0 };

function carriesFiles(event: DragEvent): boolean {
  return Boolean(event.dataTransfer && [...event.dataTransfer.types].includes("Files"));
}

/**
 * Files dragged anywhere over the Desk: whether they are over the window, and
 * whether over the composer, where letting go attaches them.
 */
export function useDeskDrop(onDrop: ((files: FileList) => void) | null): DragState {
  const [drag, setDrag] = useState<DragState>(NOT_DRAGGING);
  const onDropRef = useRef(onDrop);
  useEffect(() => {
    onDropRef.current = onDrop;
  });

  useEffect(() => {
    let depth = 0;
    const enter = (event: DragEvent) => {
      if (!carriesFiles(event) || !onDropRef.current) {
        return;
      }
      event.preventDefault();
      depth += 1;
      const count = event.dataTransfer?.items?.length ?? 1;
      setDrag((current) => ({ ...current, on: true, count }));
    };
    const over = (event: DragEvent) => {
      if (!carriesFiles(event) || !onDropRef.current) {
        return;
      }
      event.preventDefault();
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }
      const hot = Boolean((event.target as Element | null)?.closest?.(".dk-cmp"));
      setDrag((current) =>
        current.on && current.hot === hot ? current : { ...current, on: true, hot },
      );
    };
    const leave = (event: DragEvent) => {
      if (!carriesFiles(event)) {
        return;
      }
      depth = Math.max(0, depth - 1);
      if (depth === 0) {
        setDrag(NOT_DRAGGING);
      }
    };
    const drop = (event: DragEvent) => {
      if (!carriesFiles(event)) {
        return;
      }
      event.preventDefault();
      depth = 0;
      setDrag(NOT_DRAGGING);
      const files = event.dataTransfer?.files;
      if (files && files.length > 0) {
        onDropRef.current?.(files);
      }
    };
    window.addEventListener("dragenter", enter);
    window.addEventListener("dragover", over);
    window.addEventListener("dragleave", leave);
    window.addEventListener("drop", drop);
    return () => {
      window.removeEventListener("dragenter", enter);
      window.removeEventListener("dragover", over);
      window.removeEventListener("dragleave", leave);
      window.removeEventListener("drop", drop);
    };
  }, []);

  return drag;
}
