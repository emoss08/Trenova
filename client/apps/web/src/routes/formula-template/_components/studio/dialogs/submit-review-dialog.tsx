import { describeApiError } from "@/lib/api-error-message";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { CheckIcon, Send01Icon } from "@trenova/shared/components/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { ReadinessPanel } from "../../readiness-panel";
import { NovaButton } from "../ai/nova-button";
import { ImpactSummary } from "./impact-summary";

const MAX_NOTE = 4000;

function Label({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex items-center gap-2 pt-1">
      <span className="text-foreground-subtle text-xs font-medium">{children}</span>
      <span className="flex-1" />
      {action}
    </div>
  );
}

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word.charAt(0).toUpperCase())
    .join("");
}

/**
 * Sending the saved template to review: what review will check, who should
 * look, a note for them that Nova drafts from the change and the backtest,
 * and what the change does to recent freight. Submit waits on every blocking
 * check, the same ones the server enforces.
 */
export function SubmitReviewDialog({
  open,
  onOpenChange,
  template,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: FormulaTemplate;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const templateId = template.id ?? "";
  const [ready, setReady] = useState<boolean | null>(null);
  const [reviewerId, setReviewerId] = useState<string | null>(null);
  const [note, setNote] = useState("");
  const [noteError, setNoteError] = useState<string | null>(null);

  const reviewers = useQuery({
    ...formulaTemplate.reviewers(templateId),
    enabled: open && templateId !== "",
  });
  const impact = useQuery({
    ...formulaTemplate.approvalImpact(templateId),
    enabled: open && templateId !== "",
    staleTime: 30_000,
  });

  const draft = useMutation({
    mutationFn: () => apiService.formulaTemplateService.draftReviewNote(templateId),
    onMutate: () => setNoteError(null),
    onSuccess: (response) => setNote(response.note.slice(0, MAX_NOTE)),
    onError: (error) =>
      setNoteError(
        describeApiError(error, t("Nova could not draft the note. Write one yourself.")),
      ),
  });
  const { mutate: drafting } = draft;

  useEffect(() => {
    if (!open) {
      setNote("");
      setReviewerId(null);
      setReady(null);
      setNoteError(null);
      return;
    }
    drafting();
  }, [open, drafting]);

  useEffect(() => {
    const first = reviewers.data?.[0];
    if (open && reviewerId === null && first) setReviewerId(first.id);
  }, [open, reviewerId, reviewers.data]);

  const submit = useMutation({
    mutationFn: () =>
      apiService.formulaTemplateService.submit(templateId, {
        comment: "",
        reviewerId: reviewerId ?? undefined,
        reviewerNote: note.trim(),
      }),
    onSuccess: async () => {
      const reviewer = reviewers.data?.find((person) => person.id === reviewerId);
      toast(reviewer ? t("Sent to {0} for review", reviewer.name) : t("Submitted for review"));
      onOpenChange(false);
      await invalidateFormulaTemplate(queryClient);
    },
    onError: (error) =>
      toast.error(t("Could not submit the template"), {
        description: describeApiError(error),
      }),
  });

  const blocked = ready === false;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="xl"
        className="max-h-[88vh] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 p-0"
      >
        <DialogHeader className="border-b px-5 pt-5 pb-4">
          <DialogTitle>{t("Submit for review")}</DialogTitle>
          <DialogDescription>
            {t(
              "The reviewer approves exactly what is saved. Approval makes this version rate shipments.",
            )}
          </DialogDescription>
        </DialogHeader>
        <ScrollArea maskVariant="card" className="min-h-0">
          <div className="grid grid-cols-1 gap-5 px-5 py-4 md:grid-cols-2">
            <div className="flex flex-col gap-2">
              <Label>{t("Readiness")}</Label>
              {open && (
                <ReadinessPanel
                  templateId={templateId}
                  step="submit"
                  onReadinessChange={setReady}
                />
              )}
              <Label>{t("Reviewer")}</Label>
              {reviewers.isPending ? (
                <div className="flex flex-col gap-1.5">
                  <Skeleton className="h-11 w-full" />
                  <Skeleton className="h-11 w-full" />
                </div>
              ) : (reviewers.data?.length ?? 0) === 0 ? (
                <p className="text-foreground-muted rounded-md border px-3 py-2 text-xs">
                  {t(
                    "Nobody else here can approve formula templates. Anyone with approve permission will see it in their notifications.",
                  )}
                </p>
              ) : (
                <div role="radiogroup" aria-label={t("Reviewer")} className="flex flex-col gap-1.5">
                  {reviewers.data?.map((person) => {
                    const selected = person.id === reviewerId;
                    return (
                      <Button
                        key={person.id}
                        type="button"
                        variant="outline"
                        size="bare"
                        role="radio"
                        aria-checked={selected}
                        onClick={() => setReviewerId(person.id)}
                        className={cn(
                          "flex h-auto items-center justify-start gap-2.5 rounded-md px-2.5 py-2 text-left",
                          selected && "border-foreground bg-surface-selected",
                        )}
                      >
                        <span className="bg-surface-active flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-medium">
                          {initials(person.name)}
                        </span>
                        <span className="flex min-w-0 flex-1 flex-col">
                          <span className="truncate text-sm font-medium">{person.name}</span>
                          <span className="text-foreground-subtle truncate text-xs font-normal">
                            {person.username}
                          </span>
                        </span>
                        {selected && <CheckIcon className="size-3.5" />}
                      </Button>
                    );
                  })}
                </div>
              )}
            </div>
            <div className="flex flex-col gap-2">
              <Label
                action={
                  <NovaButton
                    busy={draft.isPending}
                    disabled={draft.isPending}
                    onClick={() => draft.mutate()}
                  >
                    {draft.isPending ? t("Writing…") : t("Rewrite")}
                  </NovaButton>
                }
              >
                {t("Note for the reviewer")}
              </Label>
              <Textarea
                value={draft.isPending ? "" : note}
                onChange={(event) => setNote(event.target.value.slice(0, MAX_NOTE))}
                disabled={draft.isPending}
                minRows={4}
                maxRows={10}
                aria-label={t("Note for the reviewer")}
                placeholder={
                  draft.isPending
                    ? t("Nova is reading the change and the backtest…")
                    : t("What changed and what it does to charges")
                }
                className={cn(draft.isPending && "ui-shimmer")}
              />
              {noteError && (
                <Alert variant="warning" size="sm">
                  <AlertDescription>{noteError}</AlertDescription>
                </Alert>
              )}
              <Label>{t("Impact on recent freight")}</Label>
              <ImpactSummary
                data={impact.data}
                isLoading={impact.isPending}
                isError={impact.isError}
              />
            </div>
          </div>
        </ScrollArea>
        <DialogFooter className="mx-0 mb-0 items-center px-5 py-3">
          <span className="text-foreground-muted mr-auto text-xs">
            {blocked
              ? t("Fix the blocking checks to submit.")
              : t("You'll be notified when it's reviewed.")}
          </span>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            className="gap-1.5"
            disabled={blocked || ready === null || draft.isPending}
            isLoading={submit.isPending}
            onClick={() => submit.mutate()}
          >
            <Send01Icon className="size-3.5" />
            {t("Submit")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
