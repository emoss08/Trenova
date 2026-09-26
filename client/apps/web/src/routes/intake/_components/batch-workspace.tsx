import { usePermission } from "@/hooks/use-permission";
import {
  captureBatchStatusAttrs,
  captureBatchTitle,
  captureRetention,
  captureSourceLabel,
} from "@/lib/capture";
import type { CaptureBatchDetail, CapturePage } from "@/lib/graphql/capture";
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
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate, formatUnixDateTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArrowLeftIcon, MoreHorizontalIcon, Trash2Icon } from "lucide-react";
import { useMemo, useState } from "react";
import { DocumentCard } from "./document-card";
import { FiledDocuments } from "./filed-documents";
import { LoosePages } from "./loose-pages";
import {
  LOOSE,
  dropPosition,
  leaveOut,
  mergeWithNext,
  movePage,
  newDocumentFrom,
  rotate,
  splitAfter,
} from "./page-layout";
import { PagePreviewDialog } from "./page-preview-dialog";
import type { PageMoveTarget } from "./page-thumbnail";
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

  if (query.isLoading) {
    return (
      <div className="flex flex-col gap-4 p-4" aria-busy="true">
        <Skeleton className="h-6 w-1/3" />
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }
  if (query.isError || query.data === undefined) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
        <p className="text-sm font-medium">{t("This stack could not be opened")}</p>
        <p className="text-foreground-subtle max-w-xs text-xs">
          {t("It may have been discarded, or you may no longer be able to see it.")}
        </p>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onClose}>
            {t("Back to the queue")}
          </Button>
          <Button size="sm" onClick={() => void query.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      </div>
    );
  }

  return <OpenBatch batch={query.data} now={now} onClose={onClose} />;
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
  const [confirmDiscard, setConfirmDiscard] = useState(false);

  const canEdit = canUpdate && batch.isEditable;
  const pages = useMemo(() => new Map(batch.pages.map((page) => [page.id, page])), [batch.pages]);
  const layout = editor.layout;
  const statusAttrs = captureBatchStatusAttrs(t)[batch.status];
  const retention = captureRetention(batch.retainUntil, now);
  const settled = batch.items.filter((item) => item.status === "Filed" || item.status === "Filing");

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const moveTargets: PageMoveTarget[] = layout.groups.map((group, index) => ({
    key: group.key,
    label: t("Document {0}", index + 1),
  }));

  const pageActions = {
    preview: (page: CapturePage) => setPreviewId(page.id),
    rotate: (pageId: string, turns: number) => editor.update((l) => rotate(l, pageId, turns)),
    moveTo: (pageId: string, target: string) =>
      editor.update((l) => movePage(l, pageId, target, Number.MAX_SAFE_INTEGER)),
  };

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
                  title={
                    fileCount === 0
                      ? t("Choose a record for each document you want to file")
                      : undefined
                  }
                >
                  {fileCount === 0
                    ? t("File documents")
                    : t("{0, plural, one {File # document} other {File # documents}}", fileCount)}
                </Button>
              )
            )}
            {canDelete && batch.isEditable && (
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
                    title={t("Discard the stack")}
                    description={t("Deletes every page not already filed")}
                    descriptionClassProps="whitespace-normal"
                    startContent={<Trash2Icon className="size-3.5" />}
                    color="danger"
                    onClick={() => setConfirmDiscard(true)}
                  />
                </DropdownMenuContent>
              </DropdownMenu>
            )}
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
                  "{0, plural, one {Unfiled pages are deleted in # day, on {1}.} other {Unfiled pages are deleted in # days, on {1}.}}",
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
                  moveTargets={[...moveTargets, { key: LOOSE, label: t("Set aside") }]}
                  pageActions={{
                    ...pageActions,
                    splitAfter: (pageId) => editor.update((l) => splitAfter(l, group.key, pageId)),
                    leaveOut: (pageId) => editor.update((l) => leaveOut(l, pageId)),
                  }}
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
                  onSetAside={() => {
                    if (item !== undefined) {
                      editor.discardItem(item);
                    }
                  }}
                  settingAside={item !== undefined && editor.discardingItem === item.id}
                />
              );
            })}

            {layout.groups.length === 0 && batch.isEditable && (
              <p className="text-foreground-subtle text-sm">
                {t("Every page is set aside. Make a document from one to file it.")}
              </p>
            )}

            {(batch.isEditable || layout.loose.length > 0) && (
              <LoosePages
                pageIds={layout.loose}
                pages={pages}
                rotations={layout.rotations}
                moveTargets={moveTargets}
                canEdit={canEdit}
                pageActions={{
                  ...pageActions,
                  newDocument: (pageId) => editor.update((l) => newDocumentFrom(l, pageId)),
                }}
              />
            )}
          </DndContext>

          <FiledDocuments items={settled} />
        </div>
      </ScrollArea>

      <PagePreviewDialog
        page={previewPage}
        number={previewPage?.sequence ?? 0}
        rotation={previewPage ? (layout.rotations[previewPage.id] ?? 0) : 0}
        onOpenChange={(open) => {
          if (!open) {
            setPreviewId(null);
          }
        }}
      />

      <AlertDialog open={confirmDiscard} onOpenChange={setConfirmDiscard}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia className="bg-danger-subtle text-destructive">
              <Trash2Icon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Discard this stack?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {settled.length > 0
                ? t(
                    "{0, plural, one {The # document already filed stays on its record.} other {The # documents already filed stay on their records.}} Every other page is deleted.",
                    settled.length,
                  )
                : t("Every page in it is deleted. This cannot be undone.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={editor.discardingBatch}>{t("Keep it")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={editor.discardingBatch}
              onClick={() => {
                editor.discardBatch();
                setConfirmDiscard(false);
              }}
            >
              {t("Discard stack")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
