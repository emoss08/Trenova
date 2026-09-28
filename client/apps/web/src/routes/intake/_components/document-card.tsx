import {
  ControlledCaptureRecordAutocompleteField,
  ControlledDocumentTypeAutocompleteField,
} from "@/components/autocomplete-fields";
import {
  CAPTURE_RECORD_KINDS,
  captureDocumentCategory,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { CombineIcon, UndoDotIcon } from "lucide-react";
import { withKind, type Destination } from "./destination";
import type { LayoutGroup } from "./page-layout";
import { PageThumbnail, type PageActions, type PageMoveTarget } from "./page-thumbnail";

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
        <span className="text-foreground font-medium">
          {captureSuggestionSourceLabel(t, item.suggestionSource)}
        </span>
        {item.suggestionReason !== "" && <> · {item.suggestionReason}</>}
        {confidence !== null && <span className="text-foreground-subtle"> · {confidence}</span>}
      </span>
    </p>
  );
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
  moveTargets,
  pageActions,
  canEdit,
  canFile,
  canDiscard,
  dirty,
  onMergeWithNext,
  onFile,
  filing,
  onSetAside,
  settingAside,
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
  moveTargets: PageMoveTarget[];
  pageActions: Omit<PageActions, "splitAfter"> & { splitAfter: (pageId: string) => void };
  canEdit: boolean;
  canFile: boolean;
  canDiscard: boolean;
  dirty: boolean;
  onMergeWithNext?: () => void;
  onFile: () => void;
  filing: boolean;
  onSetAside: () => void;
  settingAside: boolean;
}) {
  const t = useT();
  const { setNodeRef, isOver } = useDroppable({ id: group.key, disabled: !canEdit });
  const statusAttrs = item ? captureItemStatusAttrs(t)[item.status] : null;
  const failureText = failure ?? (item?.status === "Failed" ? item.failureMessage : "");
  const fileBlocker = dirty
    ? t("Save the split before filing")
    : item === undefined
      ? t("Save the split before filing")
      : destination.recordId === ""
        ? t("Choose a record to file onto")
        : null;

  return (
    <article
      aria-label={t("Document {0}", number)}
      className="border-border bg-card flex flex-col rounded-lg border"
    >
      <header className="border-border-subtle flex flex-wrap items-center gap-2 border-b px-3 py-2">
        <h3 className="text-sm font-semibold">{t("Document {0}", number)}</h3>
        <span className="text-foreground-subtle text-xs tabular-nums">
          {t("{0, plural, one {# page} other {# pages}}", group.pageIds.length)}
        </span>
        {statusAttrs !== null && item?.status !== "Proposed" && (
          <Badge variant={phaseTone(statusAttrs.phase)}>{statusAttrs.text}</Badge>
        )}
        {item?.detectedKind && item.detectedKind !== "" && (
          <Badge variant="neutral" appearance="outline">
            {item.detectedKind}
          </Badge>
        )}
        <div className="ml-auto flex items-center gap-1">
          {canEdit && onMergeWithNext && (
            <Button type="button" size="xs" variant="ghost" onClick={onMergeWithNext}>
              <CombineIcon className="size-3.5" />
              {t("Join with next")}
            </Button>
          )}
          {canDiscard && item !== undefined && !dirty && (
            <Button
              type="button"
              size="xs"
              variant="ghost"
              onClick={onSetAside}
              isLoading={settingAside}
            >
              <UndoDotIcon className="size-3.5" />
              {t("Set aside")}
            </Button>
          )}
        </div>
      </header>

      <div className="flex flex-col gap-3 p-3">
        {item && <SuggestionLine item={item} />}

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
              const last = index === group.pageIds.length - 1;
              return (
                <PageThumbnail
                  key={pageId}
                  page={page}
                  rotation={rotations[pageId] ?? 0}
                  number={sequence[pageId] ?? page.sequence}
                  moveTargets={moveTargets.filter((target) => target.key !== group.key)}
                  disabled={!canEdit}
                  actions={{
                    ...pageActions,
                    splitAfter: last ? undefined : pageActions.splitAfter,
                  }}
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
          <div className="flex flex-col gap-1">
            <span className="text-foreground-subtle text-xs font-medium">{t("File onto")}</span>
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
              <SelectTrigger aria-label={t("Kind of record")}>
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
            category={captureDocumentCategory(destination.kind)}
            value={destination.documentTypeId}
            onValueChange={(documentTypeId) =>
              onDestinationChange({ ...destination, documentTypeId })
            }
            disabled={!canFile}
          />
        </div>

        {canFile && (
          <div className="flex items-center justify-end gap-2">
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
    </article>
  );
}
