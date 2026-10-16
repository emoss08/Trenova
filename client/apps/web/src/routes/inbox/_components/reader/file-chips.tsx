import type { InboundAttachment, InboundMessageDetail } from "@/lib/graphql/inbox";
import type { SelectOption } from "@/lib/graphql/select-options";
import { inbox } from "@/lib/queries/inbox";
import { inboxReferenceQuery } from "@/lib/queries/inbox";
import { documentTypeName, documentTypeNamed } from "@/lib/triage/document-types";
import { apiService } from "@/services/api";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useContextMenuTrigger } from "@trenova/shared/components/context-menu/context-menu-host";
import type {
  ContextMenuEntry,
  ContextMenuSpec,
} from "@trenova/shared/components/context-menu/menu-spec";
import {
  Copy01Icon,
  Download01Icon,
  EyeIcon,
  File04Icon,
  LayersThree01Icon,
  Truck01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { cn, formatFileSize, toSentenceFragment } from "@trenova/shared/lib/utils";
import { toast } from "@trenova/shared/components/ui/toast";
import { useTriageActions, type TriageActions } from "../use-triage-actions";
import { useTriageDocumentTypes } from "../use-triage-data";
import { FileBadge } from "./file-badge";
import { useReaderPanel } from "./reader-panel";

const REFUSED_PREFIX = /^refused:\s*/i;

function laneOf(option: SelectOption): string | undefined {
  return option.description ?? undefined;
}

/** What Nova read from a file, as lines a person can paste: "Rate: $1,240.00". */
export function readingText(t: TranslateFn, attachment: InboundAttachment): string {
  return (attachment.reading?.fields ?? [])
    .map((field) => t("{0}: {1}", field.label, field.value))
    .join("\n");
}

/** The document type option a file was read as, matched by name. */
export function readTypeId(
  attachment: InboundAttachment,
  documentTypes: readonly SelectOption[],
): string | null {
  const name = attachment.reading?.documentType ?? attachment.filedAs ?? "";

  return documentTypeNamed(documentTypes, name)?.id ?? null;
}

async function download(t: TranslateFn, documentId: string) {
  try {
    const url = await apiService.documentService.getDownloadUrl(documentId);
    window.open(url, "_blank", "noopener,noreferrer");
  } catch {
    toast.error(t("The file could not be downloaded"));
  }
}

type AttachmentMenuInput = {
  t: TranslateFn;
  actions: TriageActions;
  message: InboundMessageDetail;
  attachment: InboundAttachment;
  documentTypes: readonly SelectOption[];
  shipments: readonly SelectOption[];
  openFile: (attachmentId: string, split: boolean) => void;
};

/** The file's menu: look at it, keep it, file it, split it, copy what was read, unfile it. */
export function attachmentMenu({
  t,
  actions,
  message,
  attachment,
  documentTypes,
  shipments,
  openFile,
}: AttachmentMenuInput): ContextMenuSpec {
  const matched = message.matchedShipment;
  const documentId = attachment.documentId;
  const typeId = readTypeId(attachment, documentTypes);
  const filed = attachment.filedShipmentId !== null;
  const reading = readingText(t, attachment);
  const file = (shipmentId: string, documentTypeId: string | null) => {
    void actions
      .file(message.id, { attachmentId: attachment.id, shipmentId, documentTypeId })
      .catch(() => openFile(attachment.id, false));
  };
  const onShipment = (option: { id: string; label: string; hint?: string }): ContextMenuEntry => ({
    id: option.id,
    label: option.label,
    hint: option.hint,
    onSelect: () => file(option.id, typeId),
  });

  return {
    title: attachment.fileName,
    subtitle:
      filed && attachment.filedShipment !== null
        ? t(
            "{0} · filed on {1}",
            formatFileSize(attachment.byteSize),
            attachment.filedShipment.proNumber,
          )
        : formatFileSize(attachment.byteSize),
    entries: [
      {
        id: "preview",
        label: t("Preview"),
        icon: EyeIcon,
        onSelect: () =>
          useTriageStore
            .getState()
            .openOverlay({ kind: "doc", messageId: message.id, attachmentId: attachment.id }),
      },
      {
        id: "download",
        label: t("Download"),
        icon: Download01Icon,
        disabled: documentId === null,
        onSelect: () => {
          if (documentId !== null) {
            void download(t, documentId);
          }
        },
      },
      { type: "separator" },
      {
        id: "file-on",
        label: t("File on"),
        icon: Truck01Icon,
        disabled: documentId === null || filed,
        submenu: [
          matched !== null && { type: "header", label: t("Matched") },
          matched !== null &&
            onShipment({
              id: matched.id,
              label: matched.proNumber,
              hint: t("{0} → {1}", matched.originCity, matched.destinationCity),
            }),
          { type: "header", label: t("Other shipments") },
          ...shipments
            .filter((option) => option.id !== matched?.id)
            .map((option) =>
              onShipment({ id: option.id, label: option.label, hint: laneOf(option) }),
            ),
          { type: "separator" },
          {
            id: "find",
            label: t("Find a shipment…"),
            onSelect: () => openFile(attachment.id, false),
          },
        ],
      },
      {
        id: "file-as",
        label: t("File as"),
        icon: File04Icon,
        disabled: documentId === null || filed || matched === null || documentTypes.length === 0,
        submenu: documentTypes.map((type) => ({
          id: type.id,
          label: documentTypeName(type),
          onSelect: () => {
            if (matched !== null) {
              file(matched.id, type.id);
            }
          },
        })),
      },
      attachment.pages.length > 1 &&
        !filed && {
          id: "split",
          label: t("Split pages"),
          icon: LayersThree01Icon,
          hint: t("{0, plural, one {# page} other {# pages}}", attachment.pages.length),
          onSelect: () => openFile(attachment.id, true),
        },
      { type: "separator" },
      {
        id: "copy-reading",
        label: t("Copy what Nova read"),
        icon: Copy01Icon,
        disabled: reading === "",
        onSelect: () =>
          actions.copyText(
            reading,
            t(
              "{0, plural, one {Copied # field} other {Copied # fields}}",
              attachment.reading?.fields.length ?? 0,
            ),
          ),
      },
      filed && {
        id: "unfile",
        label: t("Unfile"),
        icon: XCloseIcon,
        destructive: true,
        onSelect: () => actions.unfile(message.id, attachment.id),
      },
    ],
  };
}

function FileChip({
  message,
  attachment,
}: {
  message: InboundMessageDetail;
  attachment: InboundAttachment;
}) {
  const t = useT();
  const actions = useTriageActions();
  const panel = useReaderPanel();
  const queryClient = useQueryClient();
  const documentTypes = useTriageDocumentTypes().data?.results ?? [];
  const refused = attachment.failureText.trim() !== "";
  const filed = attachment.filedShipment;

  const trigger = useContextMenuTrigger(() =>
    refused
      ? null
      : attachmentMenu({
          t,
          actions,
          message,
          attachment,
          documentTypes,
          shipments:
            queryClient.getQueryData<GenericLimitOffsetResponse<SelectOption>>(
              inbox.filingShipments().queryKey,
            )?.results ?? [],
          openFile: panel.openFile,
        }),
  );

  const open = () => {
    if (refused) {
      return;
    }
    if (attachment.filedShipmentId !== null) {
      useTriageStore
        .getState()
        .openOverlay({ kind: "doc", messageId: message.id, attachmentId: attachment.id });
      return;
    }
    panel.openFile(attachment.id, false);
  };

  let status: string;
  if (refused) {
    status = t(
      "Refused · {0}",
      toSentenceFragment(attachment.failureText.trim().replace(REFUSED_PREFIX, "")),
    );
  } else if (filed !== null) {
    status = t("Filed on {0}", filed.proNumber);
  } else {
    status = formatFileSize(attachment.byteSize);
  }

  return (
    <button
      type="button"
      {...trigger}
      aria-disabled={refused || undefined}
      title={refused ? attachment.failureText : attachment.fileName}
      onClick={open}
      className={cn(
        "border-border ui-focus-ring flex max-w-70 items-center gap-2.5 rounded-lg border py-1.75 pr-3 pl-1.75 text-left transition-colors",
        refused
          ? "cursor-default border-dashed"
          : "hover:border-border-strong hover:bg-surface-hover",
      )}
    >
      <FileBadge fileName={attachment.fileName} dimmed={refused} />
      <span className="flex min-w-0 flex-col leading-snug">
        <span className="truncate text-base font-medium">{attachment.fileName}</span>
        <span
          className={cn(
            "truncate text-xs",
            refused ? "text-danger" : filed !== null ? "text-success" : "text-foreground-subtle",
          )}
        >
          {status}
        </span>
      </span>
    </button>
  );
}

/** The files that came with the message, each opening its step or its document. */
export function FileChips({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  useQuery(inboxReferenceQuery(inbox.filingShipments()));
  if (message.attachments.length === 0) {
    return null;
  }

  return (
    <div role="list" aria-label={t("Files")} className="flex flex-wrap gap-2">
      {message.attachments.map((attachment) => (
        <div role="listitem" key={attachment.id} className="min-w-0">
          <FileChip message={message} attachment={attachment} />
        </div>
      ))}
    </div>
  );
}
