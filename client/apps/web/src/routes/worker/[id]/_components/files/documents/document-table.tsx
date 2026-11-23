import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { DocumentReviewBadge } from "@/components/documents/document-review-badge";
import { DocumentSignatureBadge } from "@/components/documents/document-signature-badge";
import { formatFileSize } from "@/components/documents/document-upload-zone";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { documentNeedsSignature } from "@/lib/document-signature";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import type { Ref } from "react";
import {
  BOUNDED_CELL_CLASS,
  BoundedTable,
  type BoundedColumn,
  type BoundedRowProps,
  type BoundedTableHandle,
} from "../../bounded-table";
import { ROW_CHANGED_CLASS, type FlashPhase } from "../../use-row-flash";
import { documentTypeName } from "./documents-model";

type DocumentTableProps = {
  documents: readonly Document[];
  /** Selection mode: a checkbox column, and a click on the row still opens the document. */
  selecting: boolean;
  selected: ReadonlySet<string>;
  onToggle: (id: string) => void;
  onToggleAll: () => void;
  flashOf: (key: string) => FlashPhase | undefined;
  uploaderName: (userId: string) => string | null;
  actionsFor: (document: Document) => RowAction[];
  onRead: (document: Document) => void;
  handleRef: Ref<BoundedTableHandle>;
  /** Warms the code of what a row opens; called when a row is hovered or focused. */
  rowIntent: { onPointerEnter: () => void; onFocus: () => void };
};

/**
 * The worker's documents as one bounded table: name, type, who added it, when, and its size. A
 * row opens the document beside the table, and its menu holds every other thing the shared
 * documents tab can do to it. Long lists mount only the rows in view.
 */
export function DocumentTable({
  documents,
  selecting,
  selected,
  onToggle,
  onToggleAll,
  flashOf,
  uploaderName,
  actionsFor,
  onRead,
  handleRef,
  rowIntent,
}: DocumentTableProps) {
  const t = useT();
  const columns: BoundedColumn[] = [
    ...(selecting
      ? [
          {
            key: "select",
            label: (
              <Checkbox
                checked={documents.length > 0 && selected.size === documents.length}
                onCheckedChange={onToggleAll}
                aria-label={t("Select every document shown")}
              />
            ),
            shrink: true,
          },
        ]
      : []),
    { key: "name", label: t("Name") },
    { key: "type", label: t("Type") },
    { key: "added-by", label: t("Added by") },
    { key: "added", label: t("Added") },
    { key: "size", label: t("Size"), align: "end" },
    { key: "actions", label: <span className="sr-only">{t("Actions")}</span>, shrink: true },
  ];

  return (
    <BoundedTable
      columns={columns}
      label={t("Documents")}
      maxHeightClassName="max-h-128"
      estimateRowHeight={41}
      handleRef={handleRef}
      rows={documents.map((document) => ({
        key: document.id,
        render: (props) => (
          <DocumentRow
            props={props}
            document={document}
            selecting={selecting}
            checked={selected.has(document.id)}
            onToggle={onToggle}
            flash={flashOf(document.id)}
            uploaderName={uploaderName(document.uploadedById)}
            actions={actionsFor(document)}
            onRead={onRead}
            intent={rowIntent}
          />
        ),
      }))}
    />
  );
}

function DocumentRow({
  props,
  document,
  selecting,
  checked,
  onToggle,
  flash,
  uploaderName,
  actions,
  onRead,
  intent,
}: {
  props: BoundedRowProps;
  document: Document;
  selecting: boolean;
  checked: boolean;
  onToggle: (id: string) => void;
  flash: FlashPhase | undefined;
  uploaderName: string | null;
  actions: RowAction[];
  onRead: (document: Document) => void;
  intent: { onPointerEnter: () => void; onFocus: () => void };
}) {
  const t = useT();
  return (
    <tr
      {...props}
      {...intent}
      data-glide-target
      data-testid={`document-${document.id}`}
      data-changed={flash}
      tabIndex={0}
      aria-label={document.originalName}
      onClick={() => onRead(document)}
      onKeyDown={(event) => {
        if (event.target !== event.currentTarget) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onRead(document);
        }
      }}
      className={cn(
        "cursor-pointer outline-none focus-visible:[&>td]:bg-surface-hover",
        ROW_CHANGED_CLASS,
      )}
    >
      {selecting ? (
        <td
          className={BOUNDED_CELL_CLASS}
          onClick={(event) => event.stopPropagation()}
          onKeyDown={(event) => event.stopPropagation()}
        >
          <Checkbox
            checked={checked}
            onCheckedChange={() => onToggle(document.id)}
            aria-label={t("Select {0}", document.originalName)}
          />
        </td>
      ) : null}
      <td className={cn(BOUNDED_CELL_CLASS, "w-full max-w-0")}>
        <span className="flex min-w-0 items-center gap-2">
          <DocumentFileTypeIcon
            fileType={document.fileType}
            fileName={document.originalName}
            size="sm"
            className="size-6 shrink-0 rounded-md"
          />
          <span className="truncate font-medium" title={document.originalName}>
            {document.originalName}
          </span>
          <DocumentReviewBadge document={document} />
          {documentNeedsSignature(document) ? <DocumentSignatureBadge document={document} /> : null}
        </span>
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "max-w-40 truncate")}>{documentTypeName(document)}</td>
      <td className={cn(BOUNDED_CELL_CLASS, "max-w-44 truncate")}>
        {uploaderName ?? <span className="text-muted-foreground">{t("Not known")}</span>}
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "font-mono text-xs whitespace-nowrap tabular-nums")}>
        {formatUnixDateMedium(document.createdAt)}
      </td>
      <td
        className={cn(
          BOUNDED_CELL_CLASS,
          "text-right font-mono text-xs whitespace-nowrap tabular-nums",
        )}
      >
        {formatFileSize(document.fileSize)}
      </td>
      <td
        className={BOUNDED_CELL_CLASS}
        onClick={(event) => event.stopPropagation()}
        onKeyDown={(event) => event.stopPropagation()}
      >
        <RowActionsMenu label={t("Actions for {0}", document.originalName)} actions={actions} />
      </td>
    </tr>
  );
}
