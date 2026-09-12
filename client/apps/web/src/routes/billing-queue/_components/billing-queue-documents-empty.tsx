import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { UploadIcon } from "lucide-react";

/**
 * The file list as it will look with documents on it: the file mark, the name,
 * and the size, age and type along the line under it, the way every attached
 * document reads.
 */
const GHOST_FILES: readonly { name: string; meta: string }[] = [
  { name: "w-36", meta: "w-28" },
  { name: "w-28", meta: "w-32" },
  { name: "w-32", meta: "w-24" },
];

type BillingQueueDocumentsEmptyProps = {
  title: string;
  description: string;
  /** Offered only while the item can still take documents. */
  onUpload?: () => void;
  className?: string;
};

export function BillingQueueDocumentsEmpty({
  title,
  description,
  onUpload,
  className,
}: BillingQueueDocumentsEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-sm"
      title={title}
      description={description}
      action={
        onUpload ? (
          <Button variant="outline" size="sm" onClick={onUpload}>
            <UploadIcon className="size-3.5" />
            {t("Upload a document")}
          </Button>
        ) : null
      }
      sketch={
        <div className="border-border/70 bg-card divide-border/60 divide-y divide-dashed rounded-md border text-left">
          {GHOST_FILES.map((file, index) => (
            <div key={index} className="flex items-center gap-3 px-3 py-2.5">
              <span className="bg-muted size-6 shrink-0 rounded-sm" />
              <div className="flex min-w-0 flex-1 flex-col gap-2">
                <GhostLine className={`h-2 ${file.name}`} />
                <GhostLine className={file.meta} />
              </div>
            </div>
          ))}
        </div>
      }
    />
  );
}
