import { recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { usePermission } from "@/hooks/use-permission";
import {
  captureBatchStatusAttrs,
  captureBatchTitle,
  captureRetention,
  captureSourceLabel,
} from "@/lib/capture";
import type { CaptureBatchDetail } from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { sortableKeyboardCoordinates } from "@dnd-kit/sortable";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBox, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import {
  createDropdownMenuHandle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate, formatUnixDateTime } from "@trenova/shared/lib/date";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArrowLeftIcon, LinkIcon, MoreHorizontalIcon, Trash2Icon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { ConfirmDiscardDialog } from "./confirm-discard-dialog";
import { DocumentCard, type DiscardEffect } from "./document-card";
import { FiledDocuments } from "./filed-documents";
import { LoosePages } from "./loose-pages";
import {
  dropPosition,
  leaveOut,
  mergeWithNext,
  movePage,
  newDocumentFrom,
  pageNumber,
  rotate,
  splitAfter,
} from "./page-layout";
import { PageMenu, type PageMenuActions } from "./page-menu";
import { PagePreviewDialog } from "./page-preview-dialog";
import type { PageMenuPayload } from "./page-thumbnail";
import { useBatchEditor } from "./use-batch-editor";

/** One stack, opened: its documents to check, file or rearrange. */
export function BatchWorkspace({
  batchId,
  now,
  onClose,
}: {
  batchId: string;
  now: number;
  onClose: () => void;
}) {
  const t = useT();
  const query = useQuery(queries.capture.batch(batchId));

  if (query.isError) {
    const back = (
      <Button type="button" variant="outline" size="sm" onClick={onClose}>
        <ArrowLeftIcon />
        {t("Back to the queue")}
      </Button>
    );
    if (describeError(query.error).kind === "not-found") {
      return (
        <EmptySheet
          className="flex-1 justify-center"
          title={t("This stack is gone")}
          description={t(
            "It was discarded or deleted when its retention ran out, or you can no longer see it.",
          )}
          action={back}
          sketch={<DocumentSketch />}
        />
      );
    }
    return (
      <div className="flex flex-1 flex-col justify-center p-4">
        <ErrorState
          error={query.error}
          onRetry={() => void query.refetch()}
          title={t("Could not load this stack")}
          actions={back}
        />
      </div>
    );
  }
  if (query.data === undefined) {
    return (
      <div className="flex flex-col gap-4 p-4" aria-busy="true">
        <Skeleton className="h-6 w-1/3" />
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  return <OpenBatch batch={query.data} now={now} onClose={onClose} />;
}

/** A faint document card, for a stack with none to show. */
function DocumentSketch() {
  return (
    <div className="border-border mx-auto flex w-full max-w-sm flex-col gap-3 rounded-lg border p-3">
      <GhostLine className="w-1/3" />
      <div className="flex gap-3">
        {[0, 1, 2].map((index) => (
          <GhostBox key={index} className="h-16 w-12 rounded-sm" />
        ))}
      </div>
      <GhostLine className="w-2/3" />
    </div>
  );
}

/** A stack not yet split: every page it has is on its way to being read. */
const ARRIVING_STATUSES: readonly CaptureBatchDetail["status"][] = [
  "Receiving",
  "Sealed",
  "Processing",
];

function discardEffectFor(batch: CaptureBatchDetail): DiscardEffect {
  const open = batch.items.filter((item) => item.status === "Proposed" || item.status === "Failed");
  if (open.length > 1) {
    return "set-aside";
  }
  return batch.items.some((item) => item.status === "Filed" || item.status === "Filing")
    ? "closes-stack"
    : "discards-stack";
}

function OpenBatch({
  batch,
  now,
  onClose,
}: {
  batch: CaptureBatchDetail;
  now: number;
  onClose: () => void;
}) {
  const t = useT();
  const editor = useBatchEditor(batch);
  const { allowed: canUpdate } = usePermission(Resource.CaptureBatch, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.CaptureBatch, Operation.Delete);
  const [previewId, setPreviewId] = useState<string | null>(null);
  const [pageMenu] = useState(() => createDropdownMenuHandle<PageMenuPayload>());
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  const { copy } = useCopyToClipboard();

  const canEdit = canUpdate && batch.isEditable;
  const pages = useMemo(() => new Map(batch.pages.map((page) => [page.id, page])), [batch.pages]);
  const layout = editor.layout;
  const statusAttrs = captureBatchStatusAttrs(t)[batch.status];
  const retention = captureRetention(batch.retainUntil, now);
  const settled = batch.items.filter((item) => item.status === "Filed" || item.status === "Filing");
  const arriving = batch.items.length === 0 && ARRIVING_STATUSES.includes(batch.status);
  const discardEffect = discardEffectFor(batch);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const update = editor.update;
  const preview = useCallback((pageId: string) => setPreviewId(pageId), []);
  const pageActions = useMemo<PageMenuActions>(
    () => ({
      preview,
      rotate: (pageId, turns) => update((l) => rotate(l, pageId, turns)),
      splitAfter: (groupKey, pageId) => update((l) => splitAfter(l, groupKey, pageId)),
      leaveOut: (pageId) => update((l) => leaveOut(l, pageId)),
      newDocument: (pageId) => update((l) => newDocumentFrom(l, pageId)),
      moveTo: (pageId, target) =>
        update((l) => movePage(l, pageId, target, Number.MAX_SAFE_INTEGER)),
    }),
    [preview, update],
  );

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (over === null || active.id === over.id) {
      return;
    }
    editor.update((current) => {
      const position = dropPosition(current, String(over.id));
      return position === null
        ? current
        : movePage(current, String(active.id), position.target, position.index);
    });
  };

  const previewPage = previewId === null ? null : (pages.get(previewId) ?? null);
  const fileCount = editor.readyToFile.length;
  const fileBlocker =
    !canEdit || editor.dirty || fileCount > 0
      ? null
      : layout.groups.length === 0
        ? t("Nothing to file: every page is set aside")
        : t("Choose a record for each document you want to file");

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="border-border flex flex-col gap-2 border-b px-4 py-3">
        <div className="flex items-start gap-2">
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            className="lg:hidden"
            aria-label={t("Back to the queue")}
            onClick={onClose}
          >
            <ArrowLeftIcon className="size-4" />
          </Button>
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="truncate text-base font-semibold">{captureBatchTitle(t, batch)}</h2>
              <Badge variant={phaseTone(statusAttrs.phase)} title={statusAttrs.description}>
                {statusAttrs.text}
              </Badge>
            </div>
            <p className="text-foreground-subtle text-xs">
              {[
                captureSourceLabel(t, batch.source),
                batch.device?.name,
                batch.user?.name,
                formatUnixDateTime(batch.createdAt),
                t("{0, plural, one {# page} other {# pages}}", batch.receivedPageCount),
              ]
                .filter(Boolean)
                .join(" · ")}
            </p>
            {fileBlocker !== null && (
              <p className="text-foreground-subtle text-xs">{fileBlocker}</p>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {editor.dirty ? (
              <>
                <Button type="button" size="sm" variant="outline" onClick={editor.discardChanges}>
                  {t("Undo changes")}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  onClick={editor.save}
                  isLoading={editor.saving}
                  loadingText={t("Saving")}
                >
                  {t("Save split")}
                </Button>
              </>
            ) : (
              canEdit && (
                <Button
                  type="button"
                  size="sm"
                  onClick={editor.fileAll}
                  disabled={fileCount === 0}
                  isLoading={editor.filingAll}
                  loadingText={t("Filing")}
                >
                  {fileCount === 0
                    ? t("File documents")
                    : t("{0, plural, one {File # document} other {File # documents}}", fileCount)}
                </Button>
              )
            )}
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="outline"
                    aria-label={t("Stack actions")}
                  >
                    <MoreHorizontalIcon className="size-4" />
                  </Button>
                }
              />
              <DropdownMenuContent align="end" className="w-64">
                <DropdownMenuItem
                  title={t("Copy link")}
                  description={t("Opens this stack in Intake for anyone who can see it")}
                  descriptionClassProps="whitespace-normal"
                  startContent={<LinkIcon className="size-3.5" />}
                  onClick={() =>
                    void copy(
                      new URL(recordPath("capture_batch", batch.id), window.location.origin).href,
                      { withToast: true },
                    )
                  }
                />
                {canDelete && batch.isEditable && (
                  <DropdownMenuItem
                    title={t("Discard the stack")}
                    description={t("Deletes every page not already filed")}
                    descriptionClassProps="whitespace-normal"
                    startContent={<Trash2Icon className="size-3.5" />}
                    color="danger"
                    onClick={() => setConfirmDiscard(true)}
                  />
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-4 p-4">
          {batch.failureMessage !== "" && (
            <Alert variant="destructive" size="sm">
              <AlertTitle>{t("The stack could not be read")}</AlertTitle>
              <AlertDescription>{batch.failureMessage}</AlertDescription>
            </Alert>
          )}
          {!batch.isEditable &&
            !["Filed", "Discarded", "Expired", "Failed"].includes(batch.status) && (
              <Alert variant="info" size="sm">
                <AlertDescription>
                  {batch.status === "Receiving"
                    ? t(
                        "Pages are still arriving. The documents appear here once every page is in.",
                      )
                    : t("Trenova is splitting the stack and suggesting where each part goes.")}
                </AlertDescription>
              </Alert>
            )}
          {batch.isEditable && retention.state === "soon" && (
            <Alert variant="warning" size="sm">
              <AlertDescription>
                {t(
                  "{0, plural, one {Unfiled pages are deleted in # day} other {Unfiled pages are deleted in # days}}, on {1}.",
                  retention.daysLeft,
                  formatUnixDate(batch.retainUntil),
                )}
              </AlertDescription>
            </Alert>
          )}
          {editor.dirty && (
            <Alert variant="info" size="sm">
              <AlertDescription>
                {t("You changed how the pages divide. Save the split to file these documents.")}
              </AlertDescription>
            </Alert>
          )}

          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
            {layout.groups.map((group, index) => {
              const item = editor.itemFor(group);
              return (
                <DocumentCard
                  key={group.key}
                  number={index + 1}
                  group={group}
                  item={item}
                  pages={pages}
                  rotations={layout.rotations}
                  sequence={layout.sequence}
                  destination={editor.destinationFor(group)}
                  onDestinationChange={(destination) => editor.setDestination(group, destination)}
                  failure={item ? editor.failures[item.id] : undefined}
                  menu={pageMenu}
                  onPreview={preview}
                  canEdit={canEdit}
                  canFile={canEdit}
                  canDiscard={canDelete && batch.isEditable}
                  dirty={editor.dirty}
                  onMergeWithNext={
                    index < layout.groups.length - 1
                      ? () => editor.update((l) => mergeWithNext(l, group.key))
                      : undefined
                  }
                  onFile={() => {
                    if (item !== undefined) {
                      editor.fileOne({ item, destination: editor.destinationFor(group) });
                    }
                  }}
                  filing={item !== undefined && editor.filingOne === item.id}
                  discardEffect={discardEffect}
                  onDiscard={() =>
                    item === undefined ? Promise.resolve() : editor.discardItem(item)
                  }
                />
              );
            })}

            {layout.groups.length === 0 && batch.isEditable && (
              <EmptySheet
                title={t("No documents to file")}
                description={t(
                  "Every page is set aside. Make a document from one, or drag pages into a new one, to file it.",
                )}
                sketch={<DocumentSketch />}
              />
            )}

            {(batch.isEditable || arriving || layout.loose.length > 0) && (
              <LoosePages
                arriving={arriving}
                pageIds={layout.loose}
                pages={pages}
                rotations={layout.rotations}
                sequence={layout.sequence}
                menu={pageMenu}
                onPreview={preview}
                canEdit={canEdit}
              />
            )}
          </DndContext>
          <PageMenu
            handle={pageMenu}
            groups={layout.groups}
            canEdit={canEdit}
            actions={pageActions}
          />

          <FiledDocuments items={settled} />
        </div>
      </ScrollArea>

      <PagePreviewDialog
        page={previewPage}
        number={previewPage === null ? 0 : pageNumber(layout, previewPage)}
        rotation={previewPage ? (layout.rotations[previewPage.id] ?? 0) : 0}
        onOpenChange={(open) => {
          if (!open) {
            setPreviewId(null);
          }
        }}
      />

      <ConfirmDiscardDialog
        open={confirmDiscard}
        onOpenChange={setConfirmDiscard}
        title={t("Discard this stack?")}
        description={
          settled.length > 0
            ? t(
                "{0, plural, one {The # document already filed stays on its record.} other {The # documents already filed stay on their records.}} Every other page is deleted.",
                settled.length,
              )
            : t("Every page in it is deleted. This cannot be undone.")
        }
        confirmLabel={t("Discard stack")}
        failureTitle={t("The stack was not discarded")}
        onConfirm={editor.discardBatch}
      />
    </div>
  );
}
