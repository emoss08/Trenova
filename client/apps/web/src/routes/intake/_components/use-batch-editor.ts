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
import { useQueryClient } from "@tanstack/react-query";
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

/**
 * Everything a person does to one open stack, in one place: rearranging its
 * pages, choosing where each document goes, and filing or throwing away.
 *
 * The rearranged layout is a draft until it is saved, and filing waits for
 * it: a document is filed as the server holds it, so filing one the screen
 * shows differently would file the wrong pages.
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

  const saveMutation = useApiMutation({
    mutationFn: () => editCaptureItems(batch.id, toEditInput(layout, saved, batch.version)),
    onSuccess: async (next) => {
      toast.success(t("Split saved"));
      await refresh(next);
    },
    resourceName: "Capture batch",
  });

  const fileOneMutation = useApiMutation({
    mutationFn: ({ item, destination }: { item: CaptureItem; destination: Destination }) => {
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
    mutationFn: (entries: { item: CaptureItem; destination: Destination }[]) =>
      fileCaptureItems(entries.map(({ item, destination }) => filingEntry(item, destination))),
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

  const discardItemMutation = useApiMutation({
    mutationFn: (item: CaptureItem) => discardCaptureItem(item.id, item.version),
    onSuccess: async () => {
      toast.success(t("Document set aside"));
      await refresh();
    },
    resourceName: "Document",
  });

  const discardBatchMutation = useApiMutation({
    mutationFn: () => discardCaptureBatch(batch.id, batch.version),
    onSuccess: async () => {
      toast.success(t("Stack discarded"));
      await refresh();
    },
    resourceName: "Capture batch",
  });

  /** Every open document with a record chosen, as it would be filed now. */
  const readyToFile = useMemo(() => {
    if (dirty) {
      return [];
    }
    return layout.groups.flatMap((group) => {
      const item = itemForGroup(batch, group);
      const destination = destinationFor(group);
      return item !== undefined && isFileable(destination) ? [{ item, destination }] : [];
    });
  }, [batch, destinationFor, dirty, layout.groups]);

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
    filingOne: fileOneMutation.isPending ? fileOneMutation.variables?.item.id : undefined,
    fileAll: () => fileAllMutation.mutate(readyToFile),
    filingAll: fileAllMutation.isPending,
    discardItem: discardItemMutation.mutate,
    discardingItem: discardItemMutation.isPending ? discardItemMutation.variables?.id : undefined,
    discardBatch: () => discardBatchMutation.mutate(undefined),
    discardingBatch: discardBatchMutation.isPending,
  };
}

export type BatchEditor = ReturnType<typeof useBatchEditor>;
