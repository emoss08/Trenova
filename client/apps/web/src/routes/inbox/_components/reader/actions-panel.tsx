import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { useTriageStore } from "@/stores/triage-store";
import {
  ArrowRightIcon,
  EyeIcon,
  MessageChatCircleIcon,
} from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ActionStep, AdhocReplyStep } from "./action-step";
import { ConflictBanners } from "./conflict-banner";
import { Information } from "./information";
import { useReaderPanel } from "./reader-panel";

function Checklist({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const panel = useReaderPanel();
  const steps = message.actions;
  const done = steps.filter((step) => step.done).length;
  const adhoc = panel.adhocReply && !steps.some((step) => step.kind === "Reply");
  if (steps.length === 0 && !adhoc) {
    return null;
  }

  return (
    <section aria-label={t("Actions")}>
      {steps.length > 0 && (
        <>
          <div className="text-foreground-subtle mb-2 flex items-baseline justify-between text-xs font-medium">
            <h2>{t("Actions")}</h2>
            <span className="font-mono text-2xs tabular-nums">{t("{0}/{1}", done, steps.length)}</span>
          </div>
          <div
            role="progressbar"
            aria-label={t("Steps done")}
            aria-valuemin={0}
            aria-valuemax={steps.length}
            aria-valuenow={done}
            className="bg-border-subtle mb-1.5 h-0.5 overflow-hidden rounded-full"
          >
            <div
              className="bg-success ease-settle h-full origin-left rounded-full transition-transform duration-500"
              style={{ transform: `scaleX(${steps.length === 0 ? 0 : done / steps.length})` }}
            />
          </div>
        </>
      )}
      <ol className="flex flex-col">
        {steps.map((step, index) => (
          <ActionStep key={step.id} message={message} action={step} number={index + 1} />
        ))}
        {adhoc && <AdhocReplyStep message={message} number={steps.length + 1} />}
      </ol>
    </section>
  );
}

/**
 * The column beside the message: what clashes with what is on file, the
 * steps the message asks for (each finished in place), what it tells you,
 * and the way to hand it to an agent in Desk.
 */
export function ActionsPanel({
  message,
  placement,
}: {
  message: InboundMessageDetail;
  /** "rail" beside the message, "inline" above its body, "sheet" on a phone. */
  placement: "rail" | "inline" | "sheet";
}) {
  const t = useT();
  const openSteps = message.actions.filter((step) => !step.done).map((step) => step.id);

  return (
    <div
      className={cn(
        "flex min-w-0 flex-col",
        placement === "rail" && "sticky top-0 gap-6.5 self-start pt-1.5",
        placement === "inline" &&
          "bg-surface-hover border-border-subtle gap-4.5 rounded-xl border px-4 py-3.5",
        placement === "sheet" && "gap-6",
      )}
    >
      <ConflictBanners message={message} />
      <Checklist message={message} />
      <Information message={message} />
      {!message.novaRead && (
        <div className="bg-warning-subtle border-warning-border flex flex-col gap-1.5 rounded-lg border px-3 py-2.75">
          <div className="text-warning-subtle-foreground flex items-center gap-1.75 text-base">
            <EyeIcon aria-hidden className="size-3.25" />
            <span className="font-semibold">{t("Not read yet")}</span>
          </div>
          <p className="text-foreground-muted text-sm leading-normal">
            {t(
              "Nova is off, so nothing was pulled out of this message. You can still reply, link and handle it by hand.",
            )}
          </p>
        </div>
      )}
      {placement !== "inline" && (
        <button
          type="button"
          onClick={() =>
            useTriageStore
              .getState()
              .openOverlay({ kind: "desk", messageId: message.id, openActionIds: openSteps })
          }
          className="border-border hover:border-accent-teal-border hover:bg-accent-teal-subtle ui-focus-ring flex items-center gap-2.5 rounded-xl border px-3 py-2.5 text-left transition-colors"
        >
          <span
            aria-hidden
            className="bg-surface-hover text-foreground-muted flex size-6.5 shrink-0 items-center justify-center rounded-md"
          >
            <MessageChatCircleIcon className="size-3.25" />
          </span>
          <span className="flex min-w-0 flex-1 flex-col leading-snug">
            <span className="text-base font-medium">{t("Work on it in Desk")}</span>
            <span className="text-foreground-subtle text-xs">
              {t("Hand the message and its actions to one of your agents")}
            </span>
          </span>
          <ArrowRightIcon aria-hidden className="text-foreground-subtle size-3.25 shrink-0" />
        </button>
      )}
    </div>
  );
}
