import { billingQueue } from "@/lib/queries/billing-queue";
import { billingQueuePacket } from "@/lib/queries/billing-queue-packet";
import { PAGE_IMAGE_CACHE, type PageImageSource } from "@/lib/queries/document-page";
import { billingQueuePacketEndpoint } from "@/services/billing-queue";
import { File06Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import type {
  BillingQueueItem,
  BillingQueuePacketChecks,
  BillingQueuePacketPageRotation,
  BillingQueuePacketPreview as PacketSummary,
} from "@trenova/shared/types/billing-queue";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { BillingQueueDocumentPages } from "./billing-queue-document-pages";
import { BillingQueueDocumentPagesSkeleton } from "./billing-queue-document-pages-skeleton";
import type { PageFlag, PartEditing, ViewerPart } from "./billing-queue-page-viewer-types";
import { invoiceDetailsEditable, useInvoiceDetails } from "./use-invoice-details";

type Translate = ReturnType<typeof useT>;

/**
 * The one packet approving the item would send, built by the same planning
 * approval runs: the invoice it would make, then each document in its packet.
 * It is assembled afresh each time the dialog opens and shown in the page viewer
 * the document review uses, each document under its name. Pages that look blank
 * and documents missing a signature they need are flagged. While the invoice can
 * still change, a document is taken out or moved from the rail, or one of its pages
 * turned, which saves the item's packet as approving will send it and assembles the
 * packet again; the invoice's own pages don't turn.
 */
export function BillingQueuePacketPreview({
  itemId,
  open,
  onOpenChange,
}: {
  itemId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="flex h-[88vh] flex-col">
        <DialogHeader>
          <DialogTitle>{t("Invoice packet")}</DialogTitle>
          <DialogDescription>
            {t("What approving sends: the invoice, then each document in its packet.")}
          </DialogDescription>
        </DialogHeader>
        <div className="border-border min-h-0 flex-1 overflow-hidden rounded-md border">
          {open ? <PacketViewer itemId={itemId} /> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function PacketViewer({ itemId }: { itemId: string }) {
  const t = useT();
  const [page, setPage] = useState(1);
  const item = useQuery(billingQueue.get(itemId));
  const packet = useQuery({
    ...billingQueuePacket.preview(itemId),
    staleTime: 0,
    gcTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
  });

  if (packet.isError) {
    const problem = describeError(packet.error);
    return (
      <div role="alert" className="bg-muted flex h-full items-center justify-center p-6">
        <div className="text-foreground-subtle flex max-w-sm flex-col items-center gap-2 text-center text-sm">
          <File06Icon className="size-6" />
          <p className="text-foreground font-medium">{t("The packet couldn't be built.")}</p>
          {problem.detail ? <p>{problem.detail}</p> : null}
          {problem.retryable ? (
            <Button size="sm" variant="outline" onClick={() => void packet.refetch()}>
              {t("Try again")}
            </Button>
          ) : null}
        </div>
      </div>
    );
  }

  if (!packet.data) return <BillingQueueDocumentPagesSkeleton />;

  const props = {
    itemId,
    packet: packet.data,
    page: Math.min(page, packet.data.pageCount),
    onPage: setPage,
    refreshing: packet.isFetching,
  };
  return item.data && invoiceDetailsEditable(item.data) ? (
    <EditablePacketPages key={packet.data.build} {...props} item={item.data} />
  ) : (
    <PacketPages key={packet.data.build} {...props} editing={null} />
  );
}

type PacketPagesProps = {
  itemId: string;
  packet: PacketSummary;
  page: number;
  onPage: (page: number) => void;
  refreshing: boolean;
};

function EditablePacketPages({ item, ...props }: PacketPagesProps & { item: BillingQueueItem }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { save, isSaving } = useInvoiceDetails(item);
  const documentIds = props.packet.parts.flatMap((part) =>
    part.kind === "Document" && part.documentId ? [part.documentId] : [],
  );

  const rotations = item.packetPageRotations ?? [];
  // One change at a time: a second click while the packet saves and rebuilds would
  // send the same change again on top of it.
  const changing = useRef(false);

  // A failed save is reported by the save itself and leaves the packet as it was.
  const keep = async (changes: {
    packetDocumentIds: string[];
    packetPageRotations?: BillingQueuePacketPageRotation[];
  }) => {
    if (changing.current) return false;
    changing.current = true;
    try {
      await save({ ...changes, packetPicked: true });
      await queryClient.invalidateQueries({
        queryKey: billingQueuePacket.preview(props.itemId).queryKey,
      });
      return true;
    } catch {
      return false;
    } finally {
      changing.current = false;
    }
  };

  const editing: PartEditing = {
    busy: isSaving || props.refreshing,
    onRemove: async (key) => {
      const before = documentIds;
      const after = before.filter((id) => id !== key);
      if (after.length === before.length) return;
      const name = props.packet.parts.find((part) => part.documentId === key)?.name ?? "";
      const turnsBefore = rotations;
      await keep({ packetDocumentIds: after }).then((kept) => {
        if (!kept) return;
        toast(t("{0} removed from the packet", name), {
          action: {
            label: t("Undo"),
            onClick: () =>
              void keep({ packetDocumentIds: before, packetPageRotations: turnsBefore }),
          },
        });
      });
    },
    onMove: (key, index) => {
      const order = documentIds.filter((id) => id !== key);
      order.splice(Math.max(0, Math.min(order.length, index - 1)), 0, key);
      if (order.every((id, at) => id === documentIds[at])) return;
      void keep({ packetDocumentIds: order });
    },
    rotatable: (page) => documentPageAt(props.packet, page) !== null,
    onRotate: (page) => {
      const at = documentPageAt(props.packet, page);
      if (at === null) return;
      void keep({
        packetDocumentIds: documentIds,
        packetPageRotations: turnedOnce(rotations, at.documentId, at.page),
      });
    },
  };

  return <PacketPages {...props} editing={editing} />;
}

function PacketPages({
  itemId,
  packet,
  page,
  onPage,
  editing,
}: PacketPagesProps & { editing: PartEditing | null }) {
  const t = useT();
  const [source] = useState<PageImageSource>(() => ({
    image: (number) => billingQueuePacket.page(itemId, packet.build, number),
    thumbnail: (number) => billingQueuePacket.thumbnail(itemId, packet.build, number),
    downloadEndpoint: billingQueuePacketEndpoint(itemId),
  }));
  const checks = useQuery({
    ...billingQueuePacket.checks(itemId, packet.build),
    staleTime: PAGE_IMAGE_CACHE.staleTime,
    gcTime: PAGE_IMAGE_CACHE.gcTime,
    retry: 1,
  });
  const documents = packet.parts.filter((part) => part.kind === "Document").length;
  const parts = viewerPartsOf(packet, editing !== null, documents, t);

  return (
    <BillingQueueDocumentPages
      source={source}
      fileName={packet.fileName}
      pageCount={packet.pageCount}
      page={page}
      onPage={onPage}
      parts={parts}
      flags={packetFlags(packet, checks.data, parts, editing, t)}
      editing={editing}
    />
  );
}

function viewerPartsOf(
  packet: PacketSummary,
  editable: boolean,
  documents: number,
  t: Translate,
): ViewerPart[] {
  return packet.parts.map((part) => {
    const carried = part.kind === "Document" && Boolean(part.documentId);
    return {
      key: part.documentId ?? "invoice",
      name: part.kind === "Invoice" ? t("Invoice") : part.name,
      firstPage: part.firstPage,
      lastPage: part.lastPage,
      removable: editable && carried,
      movable: editable && carried && documents > 1,
    };
  });
}

/**
 * What a biller should look at before the packet goes out: a document missing the signature its
 * type needs, on its first page, and each document page that looks blank. Checks from another
 * assembly of the packet than the one shown are ignored.
 */
export function packetFlags(
  packet: PacketSummary,
  checks: BillingQueuePacketChecks | undefined,
  parts: readonly ViewerPart[],
  editing: PartEditing | null,
  t: Translate,
): ReadonlyMap<number, PageFlag> {
  const flags = new Map<number, PageFlag>();
  const removal = (part: ViewerPart | undefined) =>
    editing !== null && part?.removable
      ? {
          label: t("Remove from packet"),
          ariaLabel: t("Remove {0} from the packet", part.name),
          onAction: () => editing.onRemove(part.key),
          disabled: editing.busy,
        }
      : undefined;
  const partAt = (page: number) =>
    parts.find((part) => page >= part.firstPage && page <= part.lastPage);

  if (checks && checks.build === packet.build) {
    for (const page of checks.blankPages) {
      flags.set(page, {
        title: t("This page looks blank"),
        detail: t("Almost no ink was found on it."),
        action: removal(partAt(page)),
      });
    }
  }
  for (const part of packet.parts) {
    if (!part.unsigned) continue;
    const viewerPart = partAt(part.firstPage);
    flags.set(part.firstPage, {
      title: t("No signature found"),
      detail: t("{0} needs a signature to count.", part.name),
      action: removal(viewerPart),
    });
  }

  return flags;
}

/** The document and its own page a packet page belongs to; null for the invoice's pages. */
export function documentPageAt(
  packet: PacketSummary,
  page: number,
): { documentId: string; page: number } | null {
  const part = packet.parts.find(
    (candidate) =>
      candidate.kind === "Document" &&
      candidate.documentId &&
      page >= candidate.firstPage &&
      page <= candidate.lastPage,
  );
  return part?.documentId ? { documentId: part.documentId, page: page - part.firstPage + 1 } : null;
}

/** The packet's turns with one page given another quarter turn; a full turn drops the page. */
export function turnedOnce(
  rotations: readonly BillingQueuePacketPageRotation[],
  documentId: string,
  page: number,
): BillingQueuePacketPageRotation[] {
  const current =
    rotations.find((turn) => turn.documentId === documentId && turn.page === page)?.rotation ?? 0;
  const next = (current + 90) % 360;
  const others = rotations.filter(
    (turn) => !(turn.documentId === documentId && turn.page === page),
  );
  return next === 0 ? others : [...others, { documentId, page, rotation: next }];
}
