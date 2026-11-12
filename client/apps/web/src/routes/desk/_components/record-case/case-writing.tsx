import { withoutRecordIds } from "@/lib/record-ids";
import { describeToolCall } from "@/components/assistant/tool-presentation";
import {
  followDeskCase,
  NOTHING_WRITTEN,
  writeCaseEvent,
  type CallLedger,
  type CaseWriting as Writing,
} from "@/lib/desk-case-stream";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useState } from "react";

/** The longest subject shown beside a drafted step; past it, it is prose, not a name. */
const MAX_SUBJECT = 48;

/** How long a dropped stream waits before it is followed again. */
const RETRY_MS = 2000;
/** How often a stream that keeps dropping is followed again before the page stops asking. */
const MAX_RETRIES = 5;

/**
 * Follows what the agent on a case is writing while `active`. The stream is
 * replayed from its start each time it is followed, so a reader who arrives
 * late, or reconnects, sees the whole of it. `onEnded` is called once the run
 * says it is done, so the page reads the case again for what it filed.
 */
export function useCaseWriting(
  kind: string,
  id: string,
  active: boolean,
  onEnded: () => void,
): Writing {
  const [writing, setWriting] = useState<Writing>(NOTHING_WRITTEN);
  const ended = useLatestCallback(onEnded);

  useEffect(() => {
    if (!active) {
      return;
    }
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout> | undefined;
    let tries = 0;

    const follow = () => {
      let state = NOTHING_WRITTEN;
      const calls: CallLedger = new Map();
      setWriting(state);
      followDeskCase(
        kind,
        id,
        (event, data) => {
          state = writeCaseEvent(state, event, data, calls);
          setWriting(state);
        },
        controller.signal,
      )
        .then(() => {
          if (!controller.signal.aborted && state.ended) {
            ended();
          }
        })
        .catch(() => undefined)
        .finally(() => {
          if (controller.signal.aborted || state.ended || tries >= MAX_RETRIES) {
            return;
          }
          tries += 1;
          timer = setTimeout(follow, RETRY_MS);
        });
    };
    follow();

    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [active, ended, id, kind]);

  return active ? writing : NOTHING_WRITTEN;
}

/**
 * The plan while an agent drafts it. It is the plan card itself, filling in:
 * each change appears as a step the moment the agent drafts it, in the place
 * the finished step will stand, with its values as a skeleton until the plan
 * is filed and they can be shown as they would be approved. Above the steps
 * one line says what the agent is doing now and how much it has looked at.
 * What the agent says to itself while it works is not shown.
 */
export function CaseWritingCard({ writing, lead }: { writing: Writing; lead: string }) {
  const t = useT();
  const doing =
    writing.tool !== "" ? t(describeToolCall(writing.tool, null).title) : t("Working out the plan");

  return (
    <section className="dk-cz-plan" aria-label={t("Plan")} aria-busy data-slot="case-writing">
      <div className="dk-cz-plan-h">
        <div className="dk-cz-plan-k">{t("Plan")}</div>
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-dsk-sm text-dsk-subtle">
          <NovaBadge size="xs" />
          <span className="min-w-0">{lead}</span>
        </div>
      </div>
      <div
        className="flex min-w-0 items-center gap-2 border-t border-dsk-b-sub px-4 py-2.5 text-dsk-sm"
        role="status"
      >
        <TextShimmer as="span" className="min-w-0 truncate text-dsk-fg2">
          {t("{0}…", doing)}
        </TextShimmer>
        {writing.checked > 0 && (
          <span className="ml-auto flex-none text-dsk-subtle tabular-nums">
            {t("{0, plural, one {# thing checked} other {# things checked}}", writing.checked)}
          </span>
        )}
      </div>
      {writing.steps.map((step, index) => {
        const described = describeToolCall(step.name, step.args);
        // The subject is a name (a driver, a PRO). A tool whose first value is a
        // sentence of reasoning has no name to show, so it shows none.
        const named = withoutRecordIds(described.subject).trim();
        const subject = named.length <= MAX_SUBJECT ? named : "";

        return (
          <div key={step.name} className="dk-cz-step animate-rise">
            <span className="dk-cz-step-nd">{index + 1}</span>
            <span className="dk-cz-step-t">
              {t(described.title)}
              {subject !== "" && <span className="ml-1.5 text-dsk-subtle">{subject}</span>}
            </span>
            <span className="dk-cz-step-st">{t("Drafted")}</span>
            <div className="col-span-full flex flex-col gap-1.5 pt-1">
              <i className="h-3.5 w-2/3 ui-shimmer rounded-[4px]" />
              <i className="h-3.5 w-1/2 ui-shimmer rounded-[4px]" />
            </div>
          </div>
        );
      })}
      {writing.steps.length === 0 && (
        <div className="flex flex-col gap-2 border-t border-dsk-b-sub px-4 py-3.5">
          <i className="h-3.5 w-1/2 ui-shimmer rounded-[4px]" />
          <i className="h-3.5 w-2/3 ui-shimmer rounded-[4px]" />
        </div>
      )}
    </section>
  );
}
