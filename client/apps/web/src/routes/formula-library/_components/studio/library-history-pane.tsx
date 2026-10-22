import { ReviewHistoryView } from "@/components/tariff-studio/review-history-view";
import type { ReviewEntryData } from "@/components/tariff-studio/review-rounds";
import { SourceDiff } from "@/components/tariff-studio/source-diff";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import type {
  FormulaLibraryRecord,
  FormulaLibraryRecordVersion,
} from "@/lib/graphql/formula-library";
import { invalidateFormulaLibrary } from "@/lib/queries/formula-library";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { ClockRewindIcon, GitCompareIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaLibrary } from "@trenova/shared/types/formula-library";
import { formulaReviewDecisionSchema } from "@trenova/shared/types/formula-template";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { formatDistanceToNow } from "date-fns";
import { useMemo, useState } from "react";
import { useLibraryStudio } from "./library-studio-context";
import { StatusLabel } from "./status-label";

/** The record's review rounds in the shape the history view reads; unknown decisions are left out. */
export function reviewEntriesOf(library: FormulaLibraryRecord): ReviewEntryData[] {
  return library.reviews.flatMap((review) => {
    const decision = formulaReviewDecisionSchema.safeParse(review.decision);
    if (!decision.success) return [];
    return [
      {
        id: review.id,
        round: review.round,
        decision: decision.data,
        comment: review.comment,
        reviewerNote: review.reviewerNote,
        baseVersionNumber: review.baseVersionNumber,
        createdAt: review.createdAt,
        actor: review.actor,
        requestedReviewer: review.requestedReviewer,
      },
    ];
  });
}

function CompareDialog({
  library,
  target,
  onClose,
}: {
  library: FormulaLibraryRecord;
  target: FormulaLibraryRecordVersion | null;
  onClose: () => void;
}) {
  const t = useT();
  const older = useMemo(
    () => library.versions.filter((version) => version.versionNumber !== target?.versionNumber),
    [library.versions, target],
  );
  const [baseNumber, setBaseNumber] = useState<number | null>(null);
  const base =
    older.find((version) => version.versionNumber === baseNumber) ??
    older.find((version) => target && version.versionNumber < target.versionNumber) ??
    older[0];

  return (
    <Dialog
      open={target !== null}
      onOpenChange={(open) => {
        if (!open) {
          setBaseNumber(null);
          onClose();
        }
      }}
    >
      <DialogContent size="xl">
        <DialogHeader>
          <DialogTitle>
            {target && base
              ? t("v{0} against v{1}", target.versionNumber, base.versionNumber)
              : t("Compare versions")}
          </DialogTitle>
          <DialogDescription>
            {t("Lines the later version added are green; lines it removed are struck out.")}
          </DialogDescription>
        </DialogHeader>
        {older.length > 0 && (
          <div className="flex flex-wrap items-center gap-1.5" role="group" aria-label={t("Compare against")}>
            <span className="text-foreground-muted text-xs">{t("Against")}</span>
            {older.map((version) => (
              <Button
                key={version.id}
                type="button"
                size="xs"
                variant={version.versionNumber === base?.versionNumber ? "secondary" : "ghost"}
                aria-pressed={version.versionNumber === base?.versionNumber}
                onClick={() => setBaseNumber(version.versionNumber)}
                className="font-mono"
              >
                {t("v{0}", version.versionNumber)}
              </Button>
            ))}
          </div>
        )}
        <div className="max-h-[60vh] overflow-auto">
          {target && base ? (
            base.source === target.source && base.name === target.name ? (
              <p className="text-foreground-muted text-sm">
                {t("The two versions hold the same source.")}
              </p>
            ) : (
              <SourceDiff before={base.source} after={target.source} />
            )
          ) : (
            <p className="text-foreground-muted text-sm">{t("This is the only version.")}</p>
          )}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RollbackDialog({
  library,
  target,
  onClose,
  onRestored,
}: {
  library: FormulaLibraryRecord;
  target: FormulaLibraryRecordVersion | null;
  onClose: () => void;
  onRestored: (library: FormulaLibrary) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [message, setMessage] = useState("");
  const { mutate, isPending } = useApiMutation({
    mutationFn: (version: number) =>
      apiService.formulaLibraryService.rollback(library.id, {
        targetVersion: version,
        changeMessage: message.trim() || undefined,
      }),
    onSuccess: async (restored) => {
      toast.success(t("Restored v{0} as a new version", target?.versionNumber ?? 0));
      setMessage("");
      onRestored(restored);
      onClose();
      await invalidateFormulaLibrary(queryClient);
    },
    resourceName: "Formula Library",
  });

  return (
    <Dialog
      open={target !== null}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Restore v{0}?", target?.versionNumber ?? 0)}</DialogTitle>
          <DialogDescription>
            {t(
              "Its source becomes a new version, v{0}, as a draft to review. Templates keep the version they pinned.",
              library.currentVersionNumber + 1,
            )}
          </DialogDescription>
        </DialogHeader>
        {target && (
          <div className="max-h-80 overflow-auto">
            <SourceDiff before={library.source} after={target.source} />
          </div>
        )}
        <Textarea
          value={message}
          onChange={(event) => setMessage(event.target.value)}
          minRows={2}
          maxLength={500}
          aria-label={t("Why restore it")}
          placeholder={t("Why restore it (optional)")}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            isLoading={isPending}
            loadingText={t("Restoring…")}
            onClick={() => target && mutate(target.versionNumber)}
          >
            {t("Restore")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/**
 * The library's versions, newest first, and its review conversation. Any version compares
 * with another; an earlier one can be restored as a new draft version.
 */
export function LibraryHistoryPane({ onRestored }: { onRestored: (library: FormulaLibrary) => void }) {
  const t = useT();
  const { library, dirty } = useLibraryStudio();
  const { allowed: canUpdate } = usePermission(Resource.FormulaLibrary, Operation.Update);
  const [comparing, setComparing] = useState<FormulaLibraryRecordVersion | null>(null);
  const [restoring, setRestoring] = useState<FormulaLibraryRecordVersion | null>(null);
  const reviews = useMemo(() => (library ? reviewEntriesOf(library) : []), [library]);

  if (!library) {
    return (
      <p className="text-foreground-muted p-4 text-sm">
        {t("Every save makes a version. The first is made when you create the library.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-4 p-3">
      <section className="flex flex-col gap-2" aria-label={t("Versions")}>
        <h3 className="text-foreground-subtle text-xs font-medium">{t("Versions")}</h3>
        {dirty && (
          <p className="text-foreground-muted text-xs">
            {t("Save or discard your changes before restoring a version.")}
          </p>
        )}
        <ol className="divide-y overflow-hidden rounded-md border">
          {library.versions.map((version) => {
            const current = version.versionNumber === library.currentVersionNumber;
            return (
              <li key={version.id} className="flex items-start gap-2 px-3 py-2">
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <span className="font-mono text-sm font-medium">
                      {t("v{0}", version.versionNumber)}
                    </span>
                    <StatusLabel status={version.status} />
                    {current && (
                      <Badge variant="neutral" appearance="outline" className="text-2xs">
                        {t("Current")}
                      </Badge>
                    )}
                    {version.versionNumber === library.approvedVersionNumber && (
                      <Badge variant="success" className="text-2xs">
                        {t("Approved")}
                      </Badge>
                    )}
                    {version.tags.map((tag) => (
                      <Badge key={tag} variant="neutral" className="text-2xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                  {version.changeMessage && (
                    <p className="text-foreground-muted line-clamp-2 text-xs">
                      {version.changeMessage}
                    </p>
                  )}
                  <span className="text-foreground-subtle text-2xs">
                    {version.createdBy?.name
                      ? t(
                          "{0} · {1}",
                          version.createdBy.name,
                          formatDistanceToNow(new Date(version.createdAt * 1000), {
                            addSuffix: true,
                          }),
                        )
                      : formatDistanceToNow(new Date(version.createdAt * 1000), {
                          addSuffix: true,
                        })}
                  </span>
                </div>
                <div className="flex shrink-0 items-center gap-0.5">
                  {library.versions.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-xs"
                      aria-label={t("Compare v{0}", version.versionNumber)}
                      onClick={() => setComparing(version)}
                    >
                      <GitCompareIcon className="size-3.5" />
                    </Button>
                  )}
                  {!current && canUpdate && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-xs"
                      aria-label={t("Restore v{0}", version.versionNumber)}
                      disabled={dirty}
                      onClick={() => setRestoring(version)}
                    >
                      <ClockRewindIcon className="size-3.5" />
                    </Button>
                  )}
                </div>
              </li>
            );
          })}
        </ol>
      </section>

      <ReviewHistoryView
        reviews={reviews}
        isLoading={false}
        isError={false}
        emptyText={t("This library has never been submitted for review.")}
      />

      <CompareDialog library={library} target={comparing} onClose={() => setComparing(null)} />
      <RollbackDialog
        library={library}
        target={restoring}
        onClose={() => setRestoring(null)}
        onRestored={onRestored}
      />
    </div>
  );
}
