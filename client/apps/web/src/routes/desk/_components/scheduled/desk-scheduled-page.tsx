import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { ConversationSchedule } from "@/types/assistant";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { toast } from "@trenova/shared/components/ui/toast";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import {
  orderSchedules,
  useScheduleActions,
  useScheduleClock,
} from "../conversation/desk-schedules";
import { DeskScheduleEditDialog } from "../conversation/desk-schedule-edit-dialog";
import { DeskScheduledRow, type DeskScheduledRowActions } from "./desk-scheduled-row";
import { DeskScheduledSkeleton } from "./desk-scheduled-skeleton";
import { allSchedulesQuery } from "./schedules-query";
import "./desk-scheduled.css";

/**
 * The Scheduled page: every request the person asked the Desk to repeat,
 * across all of their conversations, the ones that run soonest first and
 * the paused ones after. Each says where its answers post and when it last
 * ran, and can be run now, edited, paused, resumed or deleted here. The rest of the
 * list is read as the reader nears its end.
 */
export function DeskScheduledPage() {
  const t = useT();
  const rt = useRichT();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useScheduleClock();
  const { allowed: canRun } = usePermission(Resource.Assistant, Operation.Create);
  const scrollRef = useRef<HTMLDivElement>(null);
  const moreRef = useRef<HTMLDivElement>(null);
  const [confirming, setConfirming] = useState<string | null>(null);
  const [editing, setEditing] = useState<ConversationSchedule | null>(null);

  const query = useInfiniteQuery(allSchedulesQuery());
  const { fetchNextPage, hasNextPage, isFetchingNextPage } = query;
  // A schedule paused or resumed while the list is read a page at a time
  // moves between pages on the server, so a later page can bring it again.
  const schedules = orderSchedules([
    ...new Map(
      (query.data?.pages ?? []).flatMap((page) => page.items).map((item) => [item.id, item]),
    ).values(),
  ]);
  const upcoming = schedules.filter((schedule) => schedule.enabled);
  const paused = schedules.filter((schedule) => !schedule.enabled);

  // A run starts in another conversation, out of sight of this page, so it
  // is announced with the way there.
  const actions = useScheduleActions({
    onRan: (schedule: ConversationSchedule) =>
      toast.success(t("Running now"), {
        description: schedule.threadTitle
          ? t("The answer will post in {0}", schedule.threadTitle)
          : t("The answer will post in its conversation"),
        action: {
          label: t("Open"),
          onClick: () => void navigate(conversationPath(schedule.threadId)),
        },
      }),
  });

  const rowActions: DeskScheduledRowActions = {
    edit: setEditing,
    toggle: actions.toggle,
    run: actions.run,
    confirmDelete: setConfirming,
    cancelDelete: () => setConfirming(null),
    remove: (schedule) => {
      setConfirming(null);
      actions.remove(schedule);
    },
  };

  useEffect(() => {
    const sentinel = moreRef.current;
    if (!sentinel || !hasNextPage) {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting) && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { root: scrollRef.current, rootMargin: "0px 0px 600px 0px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  const group = (heading: string, items: readonly ConversationSchedule[]) => (
    <section className="dk-sd-g" aria-label={heading}>
      <h2>{heading}</h2>
      <ul className="dk-sd-list">
        {items.map((schedule) => (
          <DeskScheduledRow
            key={schedule.id}
            schedule={schedule}
            now={now}
            timezone={timezone}
            canRun={canRun}
            changing={actions.isChanging(schedule.id)}
            confirming={confirming === schedule.id}
            actions={rowActions}
          />
        ))}
      </ul>
    </section>
  );

  const body = () => {
    if (query.isPending) {
      return <DeskScheduledSkeleton />;
    }
    if (query.isError && schedules.length === 0) {
      return (
        <div className="dk-sd-empty" role="alert">
          <b>{t("Couldn't load your scheduled requests")}</b>
          <Button
            variant="bare"
            size="bare"
            className="h-7.5 rounded-lg px-3 text-sm font-medium text-dsk-fg ring-1 ring-dsk-b transition-colors duration-150 hover:bg-dsk-hover"
            onClick={() => void query.refetch()}
          >
            {t("Try again")}
          </Button>
        </div>
      );
    }
    if (schedules.length === 0) {
      return (
        <div className="dk-sd-empty">
          <span className="dk-sd-empty-i">
            <DeskIcon name="clock" size={18} />
          </span>
          <b>{t("Nothing scheduled yet")}</b>
          <p>
            {rt(
              "In any conversation, start a message with when it should run, such as <q>every weekday at 7:30am, list shipments not yet dispatched</q>, or type <kbd/> first.",
              {
                q: (children) => <q>{children}</q>,
                kbd: () => <span className="dk-kbd">/schedule</span>,
              },
            )}
          </p>
          <Button className="mt-1 rounded-lg px-3.25" onClick={() => void navigate("/desk")}>
            {t("New conversation")}
          </Button>
        </div>
      );
    }

    return (
      <>
        {upcoming.length > 0 && group(t("Coming up"), upcoming)}
        {paused.length > 0 && group(t("Paused"), paused)}
        {isFetchingNextPage && <DeskScheduledSkeleton />}
      </>
    );
  };

  return (
    <div className="dk-pg" ref={scrollRef}>
      <div className="dk-pg-in dk-narrow">
        <header className="dk-pg-h">
          <span className="dk-pg-k">{t("Scheduled")}</span>
          <h1>{t("Scheduled requests")}</h1>
          <p>
            {t(
              "Requests you asked Desk to repeat. Each run asks the agent again with your access at the time, and its answer posts into the conversation the request was made in.",
            )}
          </p>
        </header>
        {body()}
        <div ref={moreRef} className="dk-sd-more" aria-hidden />
        <DeskScheduleEditDialog schedule={editing} onClose={() => setEditing(null)} />
      </div>
    </div>
  );
}
