import { SourceDiff } from "@/components/tariff-studio/source-diff";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";

/** The fields a Studio's person edits, as the comparison reads them. */
export type ComparedContent = {
  name: string;
  description: string;
  source: string;
};

type NewerVersionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  saved: ComparedContent;
  mine: ComparedContent;
  /** The saved version's number, when the record has one. */
  savedVersionNumber: number | null;
  onLoadSaved: () => void;
  onKeepEdits: () => void;
};

/**
 * The person's edits beside a version saved after they began, and the two ways out: load the
 * saved version and drop the edits, or keep the edits to save over it.
 */
export function NewerVersionDialog({
  open,
  onOpenChange,
  saved,
  mine,
  savedVersionNumber,
  onLoadSaved,
  onKeepEdits,
}: NewerVersionDialogProps) {
  const t = useT();
  const savedTitle =
    savedVersionNumber === null ? t("Saved version") : t("Saved version, v{0}", savedVersionNumber);
  const fields = [
    { key: "name", label: t("Name") },
    { key: "description", label: t("Description") },
    { key: "source", label: t("Source") },
  ] as const;
  const changed = fields.filter((field) => saved[field.key] !== mine[field.key]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg" className="max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>{t("Compare with the saved version")}</DialogTitle>
          <DialogDescription>
            {t(
              "This was saved again after you started editing. Keep your edits to save them over it, or load the saved version and lose your edits.",
            )}
          </DialogDescription>
        </DialogHeader>
        {changed.length === 0 ? (
          <p className="text-foreground-muted text-sm">
            {t("Your edits match the saved version.")}
          </p>
        ) : (
          <ScrollArea maskVariant="card" className="h-[50vh]">
            <div className="flex flex-col gap-4">
              {changed.map((field) => (
                <section key={field.key} className="flex flex-col gap-1.5">
                  <h3 className="text-sm font-medium">{field.label}</h3>
                  <SourceDiff
                    before={saved[field.key]}
                    after={mine[field.key]}
                    beforeTitle={savedTitle}
                    afterTitle={t("Your edits")}
                  />
                </section>
              ))}
            </div>
          </ScrollArea>
        )}
        <DialogFooter>
          <DialogClose render={<Button variant="outline">{t("Keep editing")}</Button>} />
          <Button variant="outline" onClick={onLoadSaved}>
            {t("Load the saved version")}
          </Button>
          <Button onClick={onKeepEdits}>{t("Keep my edits")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
