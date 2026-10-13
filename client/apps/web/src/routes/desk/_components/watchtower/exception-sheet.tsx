import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  acknowledgeWatchtowerItem,
  commentOnWatchtowerItem,
  startWatchtowerItem,
  type WatchtowerItem,
} from "@/lib/graphql/watchtower";
import { attention } from "@/lib/queries/attention";
import { watchtower } from "@/lib/queries/watchtower";
import { toneVar } from "@/components/kpi/tone";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { useCallback, useMemo, useState } from "react";
import { Link } from "react-router";
import {
  dueLabel,
  eventActor,
  eventSentence,
  exceptionDueState,
  exceptionStatusAttrs,
} from "./exception";
import { ExceptionActionDialog } from "./exception-action-dialog";
import { MAX_EXCEPTION_COMMENT, type ExceptionAction } from "./exception-forms";
import { SEVERITY_TONE } from "./severity";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * One exception, worked from the side of the feed: who owns it, when it is
 * due, what can be done next, and everything that has happened to it. The
 * history is the audit trail itself, read back, never a summary of it.
 */
export function ExceptionSheet({
  itemId,
  onOpenChange,
}: {
  itemId: string | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  return (
    <Sheet open={itemId !== null} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-[min(30rem,calc(100vw-2rem))] gap-0 overflow-hidden sm:max-w-none"
      >
        {itemId === null ? null : <ExceptionPanel key={itemId} itemId={itemId} />}
        <span className="sr-only">{t("Exception")}</span>
      </SheetContent>
    </Sheet>
  );
}

function ExceptionPanel({ itemId }: { itemId: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [now] = useState(nowInSeconds);
  const [action, setAction] = useState<ExceptionAction | null>(null);
  const [comment, setComment] = useState("");

  const itemQuery = useQuery(watchtower.item(itemId));
  const historyQuery = useQuery(watchtower.history(itemId));
  const codesQuery = useQuery({ ...watchtower.resolutionCodes(), staleTime: Infinity });
  const codes = useMemo(() => codesQuery.data ?? [], [codesQuery.data]);
  const codeLabels = useMemo(
    () => new Map(codes.map((option) => [option.code as string, option.label])),
    [codes],
  );

  const refresh = useCallback(
    async (saved?: WatchtowerItem) => {
      if (saved) {
        queryClient.setQueryData(watchtower.item(itemId).queryKey, saved);
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: watchtower._def }),
        queryClient.invalidateQueries({ queryKey: attention._def }),
      ]);
    },
    [queryClient, itemId],
  );

  const acknowledge = useApiMutation({
    resourceName: "Exception",
    mutationFn: (item: WatchtowerItem) =>
      acknowledgeWatchtowerItem({ id: item.id, version: item.version }),
    onSuccess: refresh,
  });
  const start = useApiMutation({
    resourceName: "Exception",
    mutationFn: (item: WatchtowerItem) =>
      startWatchtowerItem({ id: item.id, version: item.version }),
    onSuccess: refresh,
  });
  const addComment = useApiMutation({
    resourceName: "Exception",
    mutationFn: (body: string) => commentOnWatchtowerItem(itemId, body),
    onSuccess: async () => {
      setComment("");
      await refresh();
    },
  });

  const item = itemQuery.data;
  if (itemQuery.isLoading || item === undefined) {
    return (
      <div className="flex flex-col gap-3 p-5">
        <Skeleton className="h-6 w-2/3" />
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-32" />
      </div>
    );
  }

  const statusAttrs = exceptionStatusAttrs(t)[item.status];
  const due = dueLabel(t, exceptionDueState(item, now));
  const resolved = item.status === "Resolved";
  const busy = acknowledge.isPending || start.isPending || addComment.isPending;
  const trimmedComment = comment.trim();

  return (
    <>
      <SheetHeader className="border-border-subtle gap-1.5 border-b p-5 pr-12">
        <div className="flex items-center gap-2">
          <span
            aria-hidden
            className="size-1.5 shrink-0 rounded-full"
            style={{ backgroundColor: toneVar(SEVERITY_TONE[item.severity]) }}
          />
          <span className="text-muted-foreground text-xs">{item.kindLabel}</span>
          <Badge variant={phaseTone(statusAttrs.phase)} title={statusAttrs.description}>
            {statusAttrs.text}
          </Badge>
        </div>
        <SheetTitle>{item.title}</SheetTitle>
        {item.summary !== "" && <SheetDescription>{item.summary}</SheetDescription>}
      </SheetHeader>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-5 p-5">
          {item.overdue && (
            <Alert size="sm" variant="destructive">
              <AlertDescription>
                {item.escalationLevel > 0
                  ? t(
                      "Overdue and escalated {0, plural, one {# time} other {# times}}.",
                      item.escalationLevel,
                    )
                  : t("Overdue. It escalates to its owner and the escalation contact.")}
              </AlertDescription>
            </Alert>
          )}

          <DescriptionList layout="inline">
            <DescriptionItem label={t("Owner")}>
              {item.assignee?.name ?? <DescriptionEmpty />}
            </DescriptionItem>
            <DescriptionItem label={t("Due")}>
              {item.dueAt == null ? (
                <DescriptionEmpty />
              ) : (
                <span className="tabular-nums">
                  {formatUnixDateTime(item.dueAt)}
                  {due !== null && <span className="text-muted-foreground"> · {due}</span>}
                </span>
              )}
            </DescriptionItem>
            <DescriptionItem label={t("Raised")}>
              <span className="tabular-nums">{formatUnixDateTime(item.occurredAt)}</span>
            </DescriptionItem>
            {item.escalationLevel > 0 && (
              <DescriptionItem label={t("Escalations")} numeric>
                {item.escalationLevel}
              </DescriptionItem>
            )}
            {resolved && (
              <DescriptionItem label={t("Resolution")}>
                {item.resolutionCode == null
                  ? t("The record cleared")
                  : (codeLabels.get(item.resolutionCode) ?? item.resolutionCode)}
                {item.resolvedBy && (
                  <span className="text-muted-foreground"> · {item.resolvedBy.name}</span>
                )}
              </DescriptionItem>
            )}
            {resolved && item.resolutionNote && (
              <DescriptionItem label={t("Note")}>{item.resolutionNote}</DescriptionItem>
            )}
          </DescriptionList>

          <div className="flex flex-wrap gap-2">
            {item.status === "Open" && (
              <Button size="sm" onClick={() => acknowledge.mutate(item)} disabled={busy}>
                {t("Acknowledge")}
              </Button>
            )}
            {!resolved && item.status !== "InProgress" && (
              <Button
                size="sm"
                variant={item.status === "Open" ? "outline" : "default"}
                onClick={() => start.mutate(item)}
                disabled={busy}
              >
                {t("Start")}
              </Button>
            )}
            {!resolved && (
              <>
                <Button size="sm" variant="outline" onClick={() => setAction("assign")}>
                  {t("Assign")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setAction("snooze")}>
                  {t("Snooze")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setAction("due")}>
                  {t("Change due time")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setAction("resolve")}>
                  {t("Resolve")}
                </Button>
              </>
            )}
            {resolved && (
              <Button size="sm" variant="outline" onClick={() => setAction("reopen")}>
                {t("Reopen")}
              </Button>
            )}
            {item.path !== "" && (
              <Button
                size="sm"
                variant="ghost"
                nativeButton={false}
                render={<Link to={item.path} />}
              >
                {t("Open the record")}
                <ArrowRightIcon className="size-4" />
              </Button>
            )}
          </div>

          <form
            className="flex flex-col gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              if (trimmedComment !== "") {
                addComment.mutate(trimmedComment);
              }
            }}
          >
            <Textarea
              aria-label={t("Comment")}
              placeholder={t("Add a comment for the history")}
              value={comment}
              maxLength={MAX_EXCEPTION_COMMENT}
              onChange={(event) => setComment(event.target.value)}
            />
            <div className="flex justify-end">
              <Button
                type="submit"
                size="sm"
                variant="outline"
                disabled={trimmedComment === "" || busy}
                isLoading={addComment.isPending}
              >
                {t("Comment")}
              </Button>
            </div>
          </form>

          <section aria-label={t("History")} className="flex flex-col gap-2">
            <h3 className="text-sm font-semibold">{t("History")}</h3>
            {historyQuery.isLoading ? (
              <Skeleton className="h-16" />
            ) : (historyQuery.data ?? []).length === 0 ? (
              <p className="text-muted-foreground text-xs">{t("Nothing has happened yet.")}</p>
            ) : (
              <ol className="divide-border-subtle flex flex-col divide-y">
                {(historyQuery.data ?? []).map((event) => (
                  <li key={event.id} className="flex flex-col gap-0.5 py-2 text-sm">
                    <div className="flex items-baseline justify-between gap-3">
                      <span>{eventSentence(t, event, codeLabels)}</span>
                      <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                        {formatUnixDateTime(event.createdAt)}
                      </span>
                    </div>
                    <span className="text-muted-foreground text-xs">{eventActor(t, event)}</span>
                    {event.note !== "" && (
                      <p className="text-foreground text-xs leading-relaxed whitespace-pre-wrap">
                        {event.note}
                      </p>
                    )}
                  </li>
                ))}
              </ol>
            )}
          </section>
        </div>
      </ScrollArea>

      <ExceptionActionDialog
        item={item}
        action={action}
        codes={codes}
        onClose={() => setAction(null)}
        onSaved={refresh}
      />
    </>
  );
}
