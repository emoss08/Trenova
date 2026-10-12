import type { InboundMessageDetail, TriageAction } from "@/lib/graphql/inbox";
import { senderFirstName } from "@/lib/triage/message-state";
import { CheckIcon, ChevronDownIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { lazy, Suspense, useEffect, useRef, useState, type ReactNode } from "react";
import { USER_FORMATTERS, actionCopy } from "../action-copy";
import { useReaderPanel } from "./reader-panel";

const FileStep = lazy(() =>
  import("./steps/file-step").then((module) => ({ default: module.FileStep })),
);
const LoadStep = lazy(() =>
  import("./steps/load-step").then((module) => ({ default: module.LoadStep })),
);
const ReplyComposer = lazy(() =>
  import("./steps/reply-composer").then((module) => ({ default: module.ReplyComposer })),
);
const StopsStep = lazy(() =>
  import("./steps/stops-step").then((module) => ({ default: module.StopsStep })),
);
const ExceptionStep = lazy(() =>
  import("./steps/confirm-steps").then((module) => ({ default: module.ExceptionStep })),
);
const BlockStep = lazy(() =>
  import("./steps/confirm-steps").then((module) => ({ default: module.BlockStep })),
);

/** How long a just-finished step keeps its tick's landing. */
const JUST_DONE_MS = 1100;

function StepFallback() {
  return (
    <div className="pt-0.5 pb-3.5 pl-7.75">
      <Skeleton className="h-16 w-full" />
    </div>
  );
}

function StepWork({ message, action }: { message: InboundMessageDetail; action: TriageAction }) {
  const panel = useReaderPanel();
  switch (action.kind) {
    case "File": {
      const attachment = message.attachments.find((row) => row.id === action.attachment?.id);
      return attachment === undefined ? null : (
        <FileStep
          message={message}
          attachment={attachment}
          action={action}
          initialSplit={panel.splitRequested === attachment.id}
        />
      );
    }
    case "BuildLoad":
      return <LoadStep message={message} action={action} />;
    case "Reply":
      return <ReplyComposer message={message} action={action} />;
    case "CheckStops":
      return (
        <StopsStep
          message={message}
          action={action}
          replyStepId={message.actions.find((step) => step.kind === "Reply")?.id ?? null}
        />
      );
    case "LogException":
      return <ExceptionStep message={message} action={action} />;
    case "BlockSender":
      return <BlockStep message={message} action={action} />;
    case "Link":
      return null;
  }
}

function StepRow({
  stepId,
  number,
  title,
  detail,
  done,
  busy,
  onStart,
  children,
}: {
  stepId: string;
  number: number;
  title: string;
  detail: string;
  done: boolean;
  busy: boolean;
  onStart: () => void;
  children: ReactNode;
}) {
  const panel = useReaderPanel();
  const reduceMotion = useReducedMotion() ?? false;
  const open = panel.state.open === stepId;
  const ref = useRef<HTMLLIElement>(null);
  const [wasDone, setWasDone] = useState(done);
  const [justDone, setJustDone] = useState(false);
  if (done !== wasDone) {
    setWasDone(done);
    setJustDone(done);
  }

  useEffect(() => {
    if (!justDone) {
      return;
    }
    const timer = setTimeout(() => setJustDone(false), JUST_DONE_MS);
    return () => clearTimeout(timer);
  }, [justDone]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const frame = requestAnimationFrame(() =>
      ref.current?.scrollIntoView({ block: "nearest", behavior: reduceMotion ? "auto" : "smooth" }),
    );
    return () => cancelAnimationFrame(frame);
  }, [open, reduceMotion]);

  return (
    <li ref={ref} className="border-border-subtle animate-rise border-b last:border-b-0">
      <button
        type="button"
        aria-expanded={open}
        onClick={onStart}
        className="hover:bg-surface-hover ui-focus-ring -mx-1 flex w-[calc(100%+0.5rem)] items-center gap-2.75 rounded-lg px-1 py-2.75 text-left"
      >
        <span
          aria-hidden
          className={cn(
            "flex size-5 shrink-0 items-center justify-center rounded-full border-[1.5px] transition-colors",
            done
              ? "bg-success border-success text-foreground-on-solid"
              : busy
                ? "border-accent-violet text-accent-violet"
                : "border-border-strong text-foreground-subtle",
            justDone && "animate-confirm",
          )}
        >
          {done ? (
            <CheckIcon className="size-2.5" strokeWidth={3} />
          ) : busy ? (
            <Spinner className="size-2.5" />
          ) : (
            <span className="font-mono text-2xs tabular-nums">{number}</span>
          )}
        </span>
        <span className="flex min-w-0 flex-1 flex-col leading-snug">
          <span
            className={cn("truncate text-base font-medium", done && "text-foreground-muted")}
          >
            {title}
          </span>
          <span className={cn("truncate text-xs", done ? "text-success" : "text-foreground-subtle")}>
            {detail}
          </span>
        </span>
        {open ? (
          <ChevronDownIcon aria-hidden className="text-foreground-subtle size-3 shrink-0" />
        ) : (
          <ChevronRightIcon aria-hidden className="text-foreground-subtle size-3 shrink-0" />
        )}
      </button>
      {open && <Suspense fallback={<StepFallback />}>{children}</Suspense>}
    </li>
  );
}

/** One step of the checklist: its number or tick, what it is and where it stands, and its work. */
export function ActionStep({
  message,
  action,
  number,
}: {
  message: InboundMessageDetail;
  action: TriageAction;
  number: number;
}) {
  const t = useT();
  const panel = useReaderPanel();
  const copy = actionCopy(t, action, message, USER_FORMATTERS);
  const busy =
    (action.kind === "BuildLoad" && panel.state.drafting.load) ||
    (action.kind === "Reply" && panel.state.drafting.reply);

  return (
    <StepRow
      stepId={action.id}
      number={number}
      title={copy.title}
      detail={busy ? t("Nova is drafting…") : copy.detail}
      done={action.done}
      busy={busy}
      onStart={() => panel.start(action)}
    >
      <StepWork message={message} action={action} />
    </StepRow>
  );
}

/** A reply the message did not ask for, started with R or "Reply". */
export function AdhocReplyStep({
  message,
  number,
}: {
  message: InboundMessageDetail;
  number: number;
}) {
  const t = useT();
  const panel = useReaderPanel();
  return (
    <StepRow
      stepId="reply"
      number={number}
      title={t("Reply to {0}", senderFirstName(message))}
      detail={
        panel.state.drafting.reply
          ? t("Nova is drafting…")
          : t("Write back from the source's address")
      }
      done={false}
      busy={panel.state.drafting.reply}
      onStart={() => panel.dispatch({ type: "toggle", id: "reply" })}
    >
      <ReplyComposer message={message} action={null} />
    </StepRow>
  );
}
