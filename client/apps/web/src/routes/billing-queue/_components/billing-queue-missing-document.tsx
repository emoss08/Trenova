import { UnmetRequirementNote } from "@/components/documents/unmet-requirement-note";
import { Upload01Icon } from "@trenova/shared/components/icons";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";
import { useState, type CSSProperties, type DragEvent } from "react";

/** A blank's placeholder bar and the ink that writes over it, `order`th in reading order. */
function Blank({ x, y, width, order }: { x: number; y: number; width: number; order: number }) {
  return (
    <>
      <rect className="blank" x={x} y={y} width={width} height={1.6} rx={0.8} />
      <rect
        className="ink"
        style={{ "--i": order } as CSSProperties}
        x={x}
        y={y}
        width={width}
        height={1.6}
        rx={0.8}
      />
    </>
  );
}

/**
 * The paper that belongs in the slot, drawn the way a document's thumbnail reads:
 * a heading, two party boxes, a freight table and a signature line, dog-eared.
 */
function BlankSheet() {
  return (
    <span className="bq-missing-sheet" aria-hidden>
      <svg viewBox="0 0 85 110">
        <path className="paper" d="M1 1H70L84 15V109H1Z" />
        <path className="fold" d="M70 1V12.5Q70 15 72.5 15H84Z" />
        <rect className="title" x={8} y={8} width={34} height={3.6} rx={0.8} />
        <rect className="label" x={8} y={14.5} width={20} height={1.4} rx={0.7} />
        <rect className="box" x={8} y={21} width={33} height={21} rx={1} />
        <rect className="box" x={44} y={21} width={33} height={21} rx={1} />
        <rect className="label" x={11} y={24} width={10} height={1.4} rx={0.7} />
        <rect className="label" x={47} y={24} width={13} height={1.4} rx={0.7} />
        <Blank x={11} y={29} width={24} order={0} />
        <Blank x={11} y={33} width={17} order={1} />
        <Blank x={11} y={37} width={21} order={2} />
        <Blank x={47} y={29} width={22} order={3} />
        <Blank x={47} y={33} width={26} order={4} />
        <Blank x={47} y={37} width={15} order={5} />
        <rect className="thead" x={8} y={48} width={69} height={5} rx={0.8} />
        <rect className="rule" x={8} y={60} width={69} height={0.5} />
        <rect className="rule" x={8} y={68} width={69} height={0.5} />
        <rect className="rule" x={8} y={76} width={69} height={0.5} />
        <Blank x={10} y={55.8} width={27} order={6} />
        <Blank x={62} y={55.8} width={12} order={7} />
        <Blank x={10} y={63.6} width={20} order={8} />
        <Blank x={62} y={63.6} width={9} order={9} />
        <Blank x={10} y={71.6} width={24} order={10} />
        <Blank x={62} y={71.6} width={13} order={11} />
        <rect className="label" x={8} y={97} width={18} height={1.4} rx={0.7} />
        <rect className="rule" x={46} y={98.5} width={31} height={0.6} />
        <path
          className="signature"
          pathLength={1}
          d="M48 96.5c1.5-4.5 3.5-5 3.6-1.8s-1.4 3.2 1.1.4 3-4 4.2-1.6 1.6 2.6 3.4.2 2.4-2.8 3.6-.6 2.2 1.4 4.6.2 3-1 5-.6"
        />
      </svg>
    </span>
  );
}

const hasFiles = (event: DragEvent) => event.dataTransfer.types.includes("Files");

/**
 * A document the customer requires that the shipment does not have yet, held as
 * an empty field where its card will be. Clicking it uploads that type; a file
 * dropped on it goes up as that type. While its upload runs, the sheet stays
 * written in and a bar under it fills.
 */
export function MissingDocumentSlot({
  requirement,
  uploadProgress,
  onUpload,
  onDropFiles,
}: {
  requirement: ShipmentBillingRequirement;
  /** 0–100 while a file for this requirement is going up. */
  uploadProgress?: number;
  /** Absent once the item can no longer take documents. */
  onUpload?: () => void;
  onDropFiles?: (files: File[]) => void;
}) {
  const t = useT();
  const [over, setOver] = useState(false);
  const uploading = uploadProgress !== undefined;
  const accepting = Boolean(onDropFiles) && !uploading;

  return (
    <div role="listitem" className="flex min-w-0 flex-col gap-1.5">
      <div
        className="bq-missing ui-field relative aspect-[17/22] w-full overflow-hidden"
        data-over={over || undefined}
        data-uploading={uploading || undefined}
        onDragEnter={(event) => {
          if (accepting && hasFiles(event)) setOver(true);
        }}
        onDragOver={(event) => {
          if (!accepting || !hasFiles(event)) return;
          event.preventDefault();
          event.dataTransfer.dropEffect = "copy";
        }}
        onDragLeave={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setOver(false);
        }}
        onDrop={(event) => {
          if (!accepting || !hasFiles(event)) return;
          event.preventDefault();
          setOver(false);
          onDropFiles?.([...event.dataTransfer.files]);
        }}
      >
        <span className="bq-missing-hatch" aria-hidden />
        <BlankSheet />
        {onUpload ? (
          <button
            type="button"
            disabled={uploading}
            onClick={onUpload}
            aria-label={
              uploading
                ? t("Uploading {0}", requirement.documentTypeName)
                : t("Upload or drop the {0}", requirement.documentTypeName)
            }
            className="bq-missing-action absolute inset-0 z-10 flex items-end justify-center rounded-[inherit] pb-[11%] outline-none"
          >
            <span className="bq-missing-pill border-input text-foreground inline-flex h-7.5 items-center gap-1.5 rounded-full border pr-3 pl-2.5 text-xs font-medium">
              {uploading ? (
                <Spinner className="size-3.5" />
              ) : (
                <Upload01Icon className="text-foreground-muted size-3.5" />
              )}
              {uploading ? t("Uploading") : t("Upload")}
            </span>
          </button>
        ) : null}
        {uploading ? (
          <span
            role="progressbar"
            aria-label={t("Uploading {0}", requirement.documentTypeName)}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={uploadProgress}
            className="bg-sunken absolute inset-x-0 bottom-0 z-20 h-0.75"
          >
            <span
              className="bq-missing-progress bg-brand block h-full"
              style={{ transform: `scaleX(${uploadProgress / 100})` }}
            />
          </span>
        ) : null}
      </div>
      <span className="flex min-w-0 items-center gap-1.5 text-xs font-medium">
        <span aria-hidden className="bg-warning size-1.5 shrink-0 rounded-full" />
        <span className="truncate">{requirement.documentTypeName}</span>
      </span>
      <UnmetRequirementNote requirement={requirement} className="text-foreground-muted" />
    </div>
  );
}
