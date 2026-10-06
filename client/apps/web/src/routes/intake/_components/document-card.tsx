import {
  ControlledCaptureRecordAutocompleteField,
  ControlledDocumentTypeAutocompleteField,
} from "@/components/autocomplete-fields";
import { SectionPanel } from "@/components/section-panel";
import {
  CAPTURE_RECORD_KINDS,
  captureDetectedKindLabel,
  captureItemStatusAttrs,
  captureRecordKindLabel,
  captureSuggestionSourceLabel,
  isCaptureRecordKind,
} from "@/lib/capture";
import type { CaptureItem, CapturePage } from "@/lib/graphql/capture";
import { useDroppable } from "@dnd-kit/core";
import { SortableContext, rectSortingStrategy } from "@dnd-kit/sortable";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { GitMergeIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { useId, useState } from "react";
import { ConfirmDiscardDialog } from "./confirm-discard-dialog";
import { withKind, type Destination } from "./destination";
import { pageNumber, type LayoutGroup } from "./page-layout";
import { PageThumbnail, type PageMenu } from "./page-thumbnail";

/** The element id of a document's card, for bringing it into view. */
export function documentCardId(groupKey: string): string {
  return `capture-document-${groupKey}`;
}

/**
 * What discarding a document does to its stack. The server closes a stack
 * once nothing in it is left to file, so throwing away the last open document
 * is not the small thing throwing away one of several is.
 */
export type DiscardEffect = "set-aside" | "closes-stack" | "discards-stack";

function SuggestionLine({ item }: { item: CaptureItem }) {
  const t = useT();
  if (item.suggestionSource === null) {
    return null;
  }

  const confidence =
    item.suggestionConfidence !== null && item.suggestionSource === "Classifier"
      ? t("{0}% sure", Math.round(item.suggestionConfidence * 100))
      : null;

  return (
    <p className="text-foreground-muted flex items-start gap-1.5 text-xs">
      <AssistMark className="text-brand mt-0.5 size-3.5 shrink-0" aria-hidden />
      <span>
        <span className="text-foreground">
          {captureSuggestionSourceLabel(t, item.suggestionSource)}
        </span>
        {item.suggestionReason !== "" && <> · {item.suggestionReason}</>}
        {confidence !== null && <span className="text-foreground-subtle"> · {confidence}</span>}
      </span>
    </p>
  );
}

function DocumentMarks({ item }: { item: CaptureItem }) {
  const t = useT();
  const statusAttrs = captureItemStatusAttrs(t)[item.status];
  const detected = captureDetectedKindLabel(t, item.detectedKind);
  const showStatus = item.status !== "Proposed";

  if (!showStatus && detected === null && item.suggestionSource === null) {
    return null;
  }

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
      {showStatus && (
        <Badge variant={phaseTone(statusAttrs.phase)} title={statusAttrs.description}>
          {statusAttrs.text}
        </Badge>
      )}
      {detected !== null && (
        <Badge
          variant="neutral"
          appearance="outline"
          title={t("What Trenova read this document as")}
        >
          <AssistMark aria-hidden />
          {detected}
        </Badge>
      )}
      <SuggestionLine item={item} />
    </div>
  );
}

function discardDescription(t: TranslateFn, effect: DiscardEffect, pageCount: number): string {
  switch (effect) {
    case "set-aside":
      return t(
        "{0, plural, one {Its page moves} other {Its # pages move}} to Set aside. From there you can drag them into another document, or leave them to be deleted with the stack's other unfiled pages.",
        pageCount,
      );
    case "closes-stack":
      return t(
        "It is the last document left to file, so the stack is done once it goes. Its pages are not filed and are deleted with the stack's other unfiled pages.",
      );
    case "discards-stack":
      return t(
        "It is the only document left in the stack, so the whole stack is discarded and every page in it is deleted. This cannot be undone.",
      );
  }
}

/**
 * One document proposed from the stack: its pages, what Trenova thought it
 * was, and where the person is filing it.
 */
export function DocumentCard({
  number,
  group,
  item,
  pages,
  rotations,
  sequence,
  destination,
  onDestinationChange,
  failure,
  menu,
  onPreview,
  canEdit,
  canFile,
  canDiscard,
  dirty,
  onMergeWithNext,
  onFile,
  filing,
  discardEffect,
  onDiscard,
  active,
  onActivate,
}: {
  number: number;
  group: LayoutGroup;
  /** Absent for a document the person carved out and has not saved yet. */
  item: CaptureItem | undefined;
  pages: Map<string, CapturePage>;
  rotations: Readonly<Record<string, number>>;
  sequence: Readonly<Record<string, number>>;
  destination: Destination;
  onDestinationChange: (destination: Destination) => void;
  failure: string | undefined;
  menu: PageMenu;
  onPreview: (pageId: string) => void;
  canEdit: boolean;
  canFile: boolean;
  canDiscard: boolean;
  dirty: boolean;
  onMergeWithNext?: () => void;
  onFile: () => void;
  filing: boolean;
  discardEffect: DiscardEffect;
  /** Throws the document away; settles when the server has answered. */
  onDiscard: () => Promise<unknown>;
  /** The document the stack's shortcuts act on. */
  active: boolean;
  onActivate: () => void;
}) {
  const t = useT();
  const kindId = useId();
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  const { setNodeRef, isOver } = useDroppable({ id: group.key, disabled: !canEdit });
  const failureText = failure ?? (item?.status === "Failed" ? item.failureMessage : "");
  const fileBlocker = destination.recordId === "" ? t("Choose a record to file onto") : null;
  const title = t("Document {0}", number);
  const showMerge = canEdit && onMergeWithNext !== undefined;
  const showDiscard = canDiscard && item !== undefined && !dirty;

  return (
    <div
      id={documentCardId(group.key)}
      data-active={active || undefined}
      onFocusCapture={onActivate}
      onPointerDownCapture={onActivate}
      className={cn("scroll-my-4 rounded-lg", active && "ring-brand ring-1")}
    >
      <SectionPanel
        title={title}
        hint={t("{0, plural, one {# page} other {# pages}}", group.pageIds.length)}
        className="overflow-visible"
        action={
          showMerge || showDiscard ? (
            <div className="flex items-center gap-1">
              {showMerge && (
                <Button
                  type="button"
                  size="xs"
                  variant="ghost"
                  onClick={onMergeWithNext}
                  title={t("Join with next")}
                >
                  <GitMergeIcon className="size-3.5" aria-hidden />
                  <span className="sr-only sm:not-sr-only">{t("Join with next")}</span>
                </Button>
              )}
              {showDiscard && (
                <Button
                  type="button"
                  size="xs"
                  variant="ghost"
                  onClick={() => setConfirmDiscard(true)}
                  title={t("Discard document")}
                >
                  <Trash01Icon className="size-3.5" aria-hidden />
                  <span className="sr-only sm:not-sr-only">{t("Discard document")}</span>
                </Button>
              )}
            </div>
          ) : undefined
        }
      >
        <div className="flex flex-col gap-3 p-3">
          {item && <DocumentMarks item={item} />}

          <SortableContext items={group.pageIds} strategy={rectSortingStrategy}>
            <div
              ref={setNodeRef}
              className={cn(
                "flex min-h-32 flex-wrap gap-3 rounded-md p-1 transition-colors",
                isOver && "bg-surface-selected",
              )}
            >
              {group.pageIds.map((pageId, index) => {
                const page = pages.get(pageId);
                if (page === undefined) {
                  return null;
                }
                return (
                  <PageThumbnail
                    key={pageId}
                    page={page}
                    rotation={rotations[pageId] ?? 0}
                    number={pageNumber({ sequence }, page)}
                    groupKey={group.key}
                    last={index === group.pageIds.length - 1}
                    menu={menu}
                    onPreview={onPreview}
                    disabled={!canEdit}
                  />
                );
              })}
            </div>
          </SortableContext>

          {failureText !== "" && (
            <Alert variant="destructive" size="sm">
              <AlertDescription>{failureText}</AlertDescription>
            </Alert>
          )}

          <div className="grid gap-3 sm:grid-cols-[10rem_minmax(0,1fr)_minmax(0,14rem)] sm:items-end">
            <div className="flex flex-col gap-0.5">
              <div className="mb-0.5 flex items-center">
                <Label htmlFor={kindId} className="block text-xs font-medium">
                  {t("File onto")}
                </Label>
              </div>
              <Select
                value={destination.kind}
                items={CAPTURE_RECORD_KINDS.map((kind) => ({
                  value: kind,
                  label: captureRecordKindLabel(t, kind),
                }))}
                onValueChange={(value) => {
                  if (typeof value === "string" && isCaptureRecordKind(value)) {
                    onDestinationChange(withKind(destination, value));
                  }
                }}
                disabled={!canFile}
              >
                <SelectTrigger id={kindId} className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CAPTURE_RECORD_KINDS.map((kind) => (
                    <SelectItem key={kind} value={kind}>
                      {captureRecordKindLabel(t, kind)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <ControlledCaptureRecordAutocompleteField
              key={destination.kind}
              kind={destination.kind}
              label={captureRecordKindLabel(t, destination.kind)}
              value={destination.recordId}
              onValueChange={(recordId) => onDestinationChange({ ...destination, recordId })}
              disabled={!canFile}
            />
            <ControlledDocumentTypeAutocompleteField
              label={t("Document type")}
              placeholder={t("Optional")}
              value={destination.documentTypeId}
              onValueChange={(documentTypeId) =>
                onDestinationChange({ ...destination, documentTypeId })
              }
              disabled={!canFile}
            />
          </div>

          {canFile && (
            <div className="flex flex-wrap items-center justify-end gap-x-3 gap-y-1">
              {fileBlocker !== null && (
                <span className="text-foreground-subtle text-xs">{fileBlocker}</span>
              )}
              <Button
                type="button"
                size="sm"
                onClick={onFile}
                disabled={fileBlocker !== null}
                isLoading={filing}
                loadingText={t("Filing")}
              >
                {t("File")}
              </Button>
            </div>
          )}
        </div>

        {showDiscard && (
          <ConfirmDiscardDialog
            open={confirmDiscard}
            onOpenChange={setConfirmDiscard}
            title={t("Discard document {0}?", number)}
            description={discardDescription(t, discardEffect, group.pageIds.length)}
            confirmLabel={t("Discard")}
            failureTitle={t("The document was not discarded")}
            onConfirm={onDiscard}
          />
        )}
      </SectionPanel>
    </div>
  );
}
