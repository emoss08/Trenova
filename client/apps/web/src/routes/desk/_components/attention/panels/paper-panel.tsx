import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import type { WatchtowerItem, WatchtowerPaperPanel } from "@/lib/graphql/watchtower";
import { resolveWatchtowerItem } from "@/lib/graphql/watchtower";
import { deskAttention } from "@/lib/queries/desk-attention";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { watchtower } from "@/lib/queries/watchtower";
import { documentService } from "@/services/document";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, ClockIcon, UploadCloud01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useRef } from "react";
import { formatFactTime } from "../../watchtower/lanes";
import { paperRows, type PaperRow } from "./paper-rows";
import { PanelBody, PanelCard } from "./panel-parts";
import { agoOrIn } from "./panel-time";

/** What a paper can be filed as: what a scanner, a phone or a portal produces. */
const ACCEPT = ".pdf,.jpg,.jpeg,.png,.webp";

/**
 * Paperwork a delivered load is waiting on, worked where it is found. The
 * checklist is the shipment's own (what billing asks for, and what is on
 * file), and a paper that is missing is filed from its row: choose the file
 * and it goes on the shipment as that kind of document. No case is opened
 * for what is one upload. When the last one is filed the item is closed.
 */
export function PaperPanel({
  item,
  paper,
  now,
  timezone,
  lockedBy = "",
}: {
  item: Pick<WatchtowerItem, "id" | "version" | "subjectType" | "subjectId" | "record">;
  paper: WatchtowerPaperPanel;
  now: number;
  timezone: string;
  /** Why the reader cannot file on this item, in a sentence; "" when they can. */
  lockedBy?: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const shipmentId = item.subjectType === "Shipment" ? (item.subjectId ?? "") : "";
  const { allowed: mayFile } = usePermission(Resource.Document, Operation.Create);
  const { allowed: mayClose } = usePermission(Resource.Watchtower, Operation.Update);
  // Someone else's case is theirs to file on and to close. The panel says
  // whose it is where the upload would be, so nobody is left looking for it.
  const locked = lockedBy !== "";
  const canFile = mayFile && !locked;
  const canClose = mayClose && !locked;
  const readiness = useQuery({
    ...shipmentQueries.billingReadiness(shipmentId),
    enabled: shipmentId !== "",
  });
  const rows = paperRows(paper.papers, readiness.data?.requirements);
  const missing = rows.filter((row) => row.state !== "onFile").length;

  // Filing the last paper answers the item, so it is closed with the reason a
  // person would have picked; the source would clear it at its next check anyway.
  const close = useApiMutation({
    mutationFn: () =>
      resolveWatchtowerItem({ id: item.id, version: item.version, code: "DocumentReceived" }),
    resourceName: "Item",
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: deskAttention._def }),
        queryClient.invalidateQueries({ queryKey: watchtower.counts().queryKey }),
      ]),
  });

  return (
    <PanelCard
      eyebrow={paper.customer ? t("Paperwork · {0}", paper.customer) : t("Paperwork")}
      title={
        paper.deliveredAt > 0
          ? t(
              "Delivered {0}, {1}",
              formatFactTime(t, paper.deliveredAt, now, timezone),
              agoOrIn(t, paper.deliveredAt, now),
            )
          : t("Delivered")
      }
      badge={
        <Badge variant={missing > 0 ? "danger" : "success"} dot>
          {missing > 0
            ? t("{0, plural, one {# missing} other {# missing}}", missing)
            : t("All on file")}
        </Badge>
      }
    >
      <PanelBody>
        <ul className="flex flex-col divide-y divide-dsk-b-sub">
          {rows.map((row) => (
            <PaperRowItem
              key={row.key}
              row={row}
              shipmentId={canFile ? shipmentId : ""}
              onFiled={async (name) => {
                const fresh = await queryClient.fetchQuery({
                  ...shipmentQueries.billingReadiness(shipmentId),
                  staleTime: 0,
                });
                await queryClient.invalidateQueries({
                  queryKey: ["documents", "shipment", shipmentId],
                });
                if (canClose && fresh.missingRequirements.length === 0) {
                  await close.mutateAsync();
                  // The item leaves the list the moment it closes, so the list
                  // cannot say what happened to it; this does.
                  toast.success(
                    item.record?.label
                      ? t(
                          "{0} filed on {1}. Nothing else is missing, so it is cleared from Needs you.",
                          name,
                          item.record.label,
                        )
                      : t(
                          "{0} filed. Nothing else is missing, so it is cleared from Needs you.",
                          name,
                        ),
                  );
                }
              }}
            />
          ))}
        </ul>
        {paper.holds !== "" && <p className="mt-2 text-dsk-sm text-dsk-subtle">{paper.holds}</p>}
        {locked && missing > 0 && (
          <p className="mt-2 flex items-start gap-2 text-dsk-sm text-dsk-subtle" role="status">
            <DeskIcon name="lock" size={13} className="mt-0.5 shrink-0" />
            {lockedBy}
          </p>
        )}
      </PanelBody>
    </PanelCard>
  );
}

function PaperRowItem({
  row,
  shipmentId,
  onFiled,
}: {
  row: PaperRow;
  /** The shipment a file goes on; empty when this reader cannot file one here. */
  shipmentId: string;
  /** Told which paper was filed, once it is on the shipment. */
  onFiled: (name: string) => Promise<void>;
}) {
  const t = useT();
  const input = useRef<HTMLInputElement>(null);
  const upload = useApiMutation({
    mutationFn: async (file: File) => {
      await documentService.upload({
        file,
        resourceId: shipmentId,
        resourceType: "shipment",
        documentTypeId: row.documentTypeId ?? undefined,
        description: row.documentTypeId === null ? row.name : undefined,
      });
      await onFiled(row.name);
    },
    resourceName: "Document",
    scope: { id: `file-paper:${shipmentId}:${row.key}` },
  });

  return (
    <li className="flex min-h-10 items-center gap-2.5 py-1.5 text-dsk-base">
      <span
        aria-hidden
        className={cn(
          "grid size-4.5 flex-none place-items-center rounded-full border",
          row.state === "onFile" && "border-dsk-success bg-dsk-success text-dsk-on-solid",
          row.state === "waiting" && "border-dsk-warn text-dsk-warn-fg",
          row.state === "missing" && "border-dashed border-dsk-b-strong",
        )}
      >
        {row.state === "onFile" && <CheckIcon className="size-2.5" strokeWidth={3} />}
        {row.state === "waiting" && <ClockIcon className="size-2.5" />}
      </span>
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-dsk-fg">{row.name}</span>
        <span
          className={cn(
            "text-xs",
            row.state === "missing"
              ? "text-dsk-error"
              : row.state === "waiting"
                ? "text-dsk-warn-fg"
                : "text-dsk-subtle",
          )}
        >
          {row.state === "onFile"
            ? t("On file")
            : row.state === "waiting"
              ? t("Filed, waiting to be accepted")
              : t("Not on file")}
        </span>
      </span>
      {row.state === "missing" && shipmentId !== "" && (
        <>
          <input
            ref={input}
            type="file"
            accept={ACCEPT}
            className="sr-only"
            tabIndex={-1}
            aria-label={t("Choose a file for {0}", row.name)}
            onChange={(event) => {
              const file = event.target.files?.[0];
              event.target.value = "";
              if (file) {
                upload.mutate(file);
              }
            }}
          />
          <Button
            type="button"
            size="sm"
            isLoading={upload.isPending}
            loadingText={t("Filing…")}
            onClick={() => input.current?.click()}
          >
            <UploadCloud01Icon aria-hidden className="size-3.5" />
            {t("Add the file")}
          </Button>
        </>
      )}
    </li>
  );
}
