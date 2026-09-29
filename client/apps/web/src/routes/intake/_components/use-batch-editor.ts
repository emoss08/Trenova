import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  discardCaptureBatch,
  discardCaptureItem,
  editCaptureItems,
  fileCaptureItem,
  fileCaptureItems,
  type CaptureBatchDetail,
  type CaptureItem,
} from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import {
  destinationKey,
  filingEntry,
  initialDestination,
  isFileable,
  type Destination,
} from "./destination";
import {
  isDirty,
  layoutFromBatch,
  toEditInput,
  type LayoutGroup,
  type PageLayout,
} from "./page-layout";

/** The item a document on screen is, when the layout it belongs to is saved. */
function itemForGroup(batch: CaptureBatchDetail, group: LayoutGroup): CaptureItem | undefined {
  return (
    batch.items.find((item) => item.id === group.key) ??
    batch.items.find(
      (item) =>
        (item.status === "Proposed" || item.status === "Failed") &&
        item.pageIds[0] === group.pageIds[0],
    )
  );
}

/** The open item holding exactly these pages, in this order. */
function itemWithPages(
  batch: CaptureBatchDetail,
  pageIds: readonly string[],
): CaptureItem | undefined {
  return batch.items.find(
    (item) =>
      (item.status === "Proposed" || item.status === "Failed") &&
      item.pageIds.length === pageIds.length &&
      item.pageIds.every((id, index) => id === pageIds[index]),
  );
}

/**
 * Everything a person does to one open stack, in one place: rearranging its
 * pages, choosing where each document goes, and filing or throwing away.
 *
 * The rearranged layout is a draft until it is saved. A document is filed as
 * the server holds it, so filing from a changed draft saves the draft first
 * and then files the documents as they now stand on the server: what the
 * person sees is what is filed, without a separate save.
 */
export function useBatchEditor(batch: CaptureBatchDetail) {
  const t = useT();
  const queryClient = useQueryClient();
  const saved = useMemo(() => layoutFromBatch(batch), [batch]);

  // The draft follows the batch: a new version from the server (a save, a
  // filing finishing, another person's edit) replaces it, which is what the
  // version check on save would force anyway.
  const [draft, setDraft] = useState<{ version: number; layout: PageLayout }>({
    version: batch.version,
    layout: saved,
  });
  if (draft.version !== batch.version) {
    setDraft({ version: batch.version, layout: saved });
  }
  const layout = draft.version === batch.version ? draft.layout : saved;
  const dirty = isDirty(layout, saved);

  const update = useCallback(
    (change: (current: PageLayout) => PageLayout) =>
      setDraft((current) => ({ ...current, layout: change(current.layout) })),
    [],
  );
  const discardChanges = useCallback(
    () => setDraft({ version: batch.version, layout: saved }),
    [batch.version, saved],
  );

  const [destinations, setDestinations] = useState<Record<string, Destination>>({});
  const destinationFor = useCallback(
    (group: LayoutGroup): Destination =>
      destinations[destinationKey(group.pageIds)] ??
      initialDestination(itemForGroup(batch, group), batch),
    [batch, destinations],
  );
  const setDestination = useCallback((group: LayoutGroup, destination: Destination) => {
    setDestinations((current) => ({ ...current, [destinationKey(group.pageIds)]: destination }));
  }, []);

  // Why a document did not file, from the last filing, until it changes.
  const [failures, setFailures] = useState<Record<string, string>>({});

  const refresh = useCallback(
    (next?: CaptureBatchDetail) => {
      if (next !== undefined) {
        queryClient.setQueryData(queries.capture.batch(next.id).queryKey, next);
      }
      return Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.capture.batch(batch.id).queryKey }),
        queryClient.invalidateQueries({ queryKey: queries.capture.batches._def }),
        queryClient.invalidateQueries({ queryKey: queries.capture.batchCount._def }),
      ]);
    },
    [batch.id, queryClient],
  );

  /**
   * The batch to file from: this one when nothing changed, otherwise the one
   * the server returns once the draft is saved.
   */
  const settle = useCallback(async (): Promise<CaptureBatchDetail> => {
    if (!dirty) {
      return batch;
    }
    const next = await editCaptureItems(batch.id, toEditInput(layout, saved, batch.version));
    queryClient.setQueryData(queries.capture.batch(next.id).queryKey, next);
    return next;
  }, [batch, dirty, layout, queryClient, saved]);

  /** The item a document on screen is on `current`, which may be newly saved. */
  const itemOn = useCallback(
    (current: CaptureBatchDetail, group: LayoutGroup) =>
      dirty ? itemWithPages(current, group.pageIds) : itemForGroup(current, group),
    [dirty],
  );

  const saveMutation = useApiMutation({
    mutationFn: () => editCaptureItems(batch.id, toEditInput(layout, saved, batch.version)),
    onSuccess: async (next) => {
      toast.success(t("Split saved"));
      await refresh(next);
    },
    resourceName: "Capture batch",
  });

  const fileOneMutation = useApiMutation({
    mutationFn: async (group: LayoutGroup) => {
      const destination = destinationFor(group);
      const current = await settle();
      const item = itemOn(current, group);
      if (item === undefined) {
        throw new Error(t("The stack changed while you were working. Reload it and file again."));
      }
      const entry = filingEntry(item, destination);
      return fileCaptureItem(item.id, {
        targetType: entry.targetType,
        targetId: entry.targetId,
        documentTypeId: entry.documentTypeId,
        version: entry.version,
      });
    },
    onSuccess: async (filed) => {
      setFailures((current) =>
        Object.fromEntries(Object.entries(current).filter(([itemId]) => itemId !== filed.id)),
      );
      toast.success(t("Filing the document"));
      await refresh();
    },
    resourceName: "Document",
  });

  const fileAllMutation = useApiMutation({
    mutationFn: async (ready: { group: LayoutGroup; destination: Destination }[]) => {
      const current = await settle();
      const entries = ready.flatMap(({ group, destination }) => {
        const item = itemOn(current, group);
        return item === undefined ? [] : [filingEntry(item, destination)];
      });
      return fileCaptureItems(entries);
    },
    onSuccess: async (result) => {
      setFailures(Object.fromEntries(result.failures.map((row) => [row.itemId, row.message])));
      if (result.failures.length === 0) {
        toast.success(
          t("{0, plural, one {Filing # document} other {Filing # documents}}", result.filed.length),
        );
      } else {
        toast.warning(
          t(
            "{0, plural, one {# document could not be filed} other {# documents could not be filed}}",
            result.failures.length,
          ),
          { description: t("The reason is shown on each one.") },
        );
      }
      await refresh();
    },
    resourceName: "Documents",
  });

  // The discards are confirmed in a dialog that stays open until they settle
  // and says there why one failed, so they report nothing on their own.
  const discardItemMutation = useMutation({
    mutationFn: (item: CaptureItem) => discardCaptureItem(item.id, item.version),
    onSuccess: async (settled) => {
      toast.success(t("Document discarded"), {
        description:
          settled.status === "Discarded"
            ? t("It was the last document in the stack, so the stack is discarded too.")
            : settled.status === "Filed"
              ? t("It was the last document left to file, so the stack is done.")
              : t("Its pages are under Set aside, where you can drag them into another document."),
      });
      await refresh();
    },
  });

  const discardBatchMutation = useMutation({
    mutationFn: () => discardCaptureBatch(batch.id, batch.version),
    onSuccess: async () => {
      toast.success(t("Stack discarded"), {
        description: t("Its unfiled pages are deleted. Filed documents stay on their records."),
      });
      await refresh();
    },
  });

  /**
   * Every document on screen with a record chosen, as it would be filed now.
   * A changed draft counts every one of its documents: filing saves it first.
   */
  const readyToFile = useMemo(
    () =>
      layout.groups.flatMap((group) => {
        const destination = destinationFor(group);
        const known = dirty || itemForGroup(batch, group) !== undefined;
        return known && group.pageIds.length > 0 && isFileable(destination)
          ? [{ group, destination }]
          : [];
      }),
    [batch, destinationFor, dirty, layout.groups],
  );

  return {
    layout,
    saved,
    dirty,
    update,
    discardChanges,
    itemFor: (group: LayoutGroup) => itemForGroup(batch, group),
    destinationFor,
    setDestination,
    failures,
    readyToFile,
    save: () => saveMutation.mutate(undefined),
    saving: saveMutation.isPending,
    fileOne: fileOneMutation.mutate,
    filingOne: fileOneMutation.isPending ? fileOneMutation.variables?.key : undefined,
    fileAll: () => fileAllMutation.mutate(readyToFile),
    filingAll: fileAllMutation.isPending,
    discardItem: discardItemMutation.mutateAsync,
    discardBatch: () => discardBatchMutation.mutateAsync(),
  };
}

export type BatchEditor = ReturnType<typeof useBatchEditor>;
