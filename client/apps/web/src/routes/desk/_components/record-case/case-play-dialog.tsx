import { handleMutationError } from "@/hooks/use-api-mutation";
import {
  rewordWatchtowerPlay,
  sendWatchtowerPlay,
  type WatchtowerPlay,
} from "@/lib/graphql/watchtower";
import { watchtower } from "@/lib/queries/watchtower";
import { useQuery } from "@tanstack/react-query";
import type { WatchtowerPlayKind } from "@trenova/graphql/generated/graphql";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
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
import { Input } from "@trenova/shared/components/ui/input";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { newIdempotencyKey } from "@trenova/shared/lib/idempotency";
import { useId, useRef, useState } from "react";
import { casePlayChannelLabel, casePlayLabel } from "./case-play";

/** A message's size as the server counts it: bytes, so an accent or an emoji is more than one. */
function sizeOf(text: string): number {
  return new TextEncoder().encode(text).length;
}

/**
 * A condition's first move, sent by the person themselves: asking for a proof
 * of delivery, asking where a load is, offering a move to carriers again.
 * The server plans it from the record (who it goes to and how) each time the
 * dialog opens, and the person reads it, changes the words if they like and
 * sends it under their own name. No agent has to be set up for any of it.
 * When it cannot be sent, the reason is what the dialog says.
 */
export function CasePlayDialog({
  itemId,
  play,
  onOpenChange,
  onSent,
}: {
  itemId: string;
  play: WatchtowerPlayKind;
  onOpenChange: (open: boolean) => void;
  /** Told once it is sent, so the case reads again and says who was asked. */
  onSent: () => void;
}) {
  const t = useT();
  // Planned afresh each time: who can be reached changes with the record.
  const query = useQuery({ ...watchtower.play(itemId), gcTime: 0, staleTime: 0, retry: false });

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent size="md">
        {query.isPending ? (
          <PlaySkeleton title={casePlayLabel(play, t)} />
        ) : query.isError ? (
          <PlayUnavailable
            title={casePlayLabel(play, t)}
            reason={describeError(query.error).detail ?? describeError(query.error).message}
            onClose={() => onOpenChange(false)}
          />
        ) : (
          <PlayForm
            key={query.dataUpdatedAt}
            planned={query.data}
            onClose={() => onOpenChange(false)}
            onSent={onSent}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

function PlaySkeleton({ title }: { title: string }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Reading the load")}>
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription>{t("Reading the load to see who can be reached.")}</DialogDescription>
      </DialogHeader>
      <div className="flex flex-col gap-2">
        <i className="h-4 w-1/2 ui-shimmer rounded-[4px]" />
        <i className="h-24 w-full ui-shimmer rounded-md" />
      </div>
    </div>
  );
}

function PlayUnavailable({
  title,
  reason,
  onClose,
}: {
  title: string;
  reason: string;
  onClose: () => void;
}) {
  const t = useT();

  return (
    <div className="flex flex-col gap-4">
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription>{t("This cannot be sent from here right now.")}</DialogDescription>
      </DialogHeader>
      <p role="alert" className="text-sm text-foreground">
        {reason}
      </p>
      <DialogFooter>
        <Button type="button" onClick={onClose}>
          {t("OK")}
        </Button>
      </DialogFooter>
    </div>
  );
}

function PlayForm({
  planned,
  onClose,
  onSent,
}: {
  planned: WatchtowerPlay;
  onClose: () => void;
  onSent: () => void;
}) {
  const t = useT();
  const id = useId();
  const sending = useRef(false);
  const [message, setMessage] = useState(planned.message);
  const [guidance, setGuidance] = useState("");
  const [pending, setPending] = useState<"send" | "reword" | null>(null);
  const hasMessage = planned.message !== "";
  const size = sizeOf(message);
  const tooLong = planned.messageLimit > 0 && size > planned.messageLimit;
  const empty = hasMessage && message.trim() === "";

  const send = async () => {
    if (sending.current || tooLong || empty) {
      return;
    }
    sending.current = true;
    setPending("send");
    try {
      await sendWatchtowerPlay(planned.itemId, hasMessage ? message.trim() : "", {
        idempotencyKey: newIdempotencyKey(),
      });
      onSent();
      onClose();
    } catch (error) {
      handleMutationError({ error, resourceName: "Request" });
    } finally {
      sending.current = false;
      setPending(null);
    }
  };

  const reword = async () => {
    const asked = guidance.trim();
    if (sending.current || asked === "") {
      return;
    }
    sending.current = true;
    setPending("reword");
    try {
      setMessage(await rewordWatchtowerPlay(planned.itemId, { message, guidance: asked }));
      setGuidance("");
    } catch (error) {
      handleMutationError({ error, resourceName: "Message" });
    } finally {
      sending.current = false;
      setPending(null);
    }
  };

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void send();
      }}
    >
      <DialogHeader>
        <DialogTitle>{casePlayLabel(planned.kind, t)}</DialogTitle>
        <DialogDescription>
          {hasMessage
            ? t(
                "It goes out under your name. Who it goes to comes from the load; the words are yours to change.",
              )
            : t("It is made under your name, from what is on the load.")}
        </DialogDescription>
      </DialogHeader>

      <dl className="grid grid-cols-[auto_1fr] items-baseline gap-x-3 gap-y-1.5 text-sm">
        <dt className="text-foreground-muted">{t("To")}</dt>
        <dd className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="min-w-0 break-words">{planned.recipient}</span>
          <Badge variant="neutral">{casePlayChannelLabel(planned.channel, t)}</Badge>
        </dd>
        {planned.sender !== "" && (
          <>
            <dt className="text-foreground-muted">{t("From")}</dt>
            <dd className="min-w-0 break-words">{planned.sender}</dd>
          </>
        )}
        {planned.subject !== "" && (
          <>
            <dt className="text-foreground-muted">{t("Subject")}</dt>
            <dd className="min-w-0 break-words">{planned.subject}</dd>
          </>
        )}
      </dl>

      {planned.summary !== "" && <p className="text-sm text-foreground">{planned.summary}</p>}

      {hasMessage && (
        <div className="flex flex-col gap-1.5">
          <div className="flex items-baseline justify-between gap-3">
            <label htmlFor={id} className="text-xs font-medium text-foreground-muted">
              {t("Message")}
            </label>
            {planned.messageLimit > 0 && (
              <span
                className={
                  tooLong ? "text-xs text-danger-foreground" : "text-xs text-foreground-muted"
                }
                aria-live="polite"
              >
                {t("{0} of {1}", size, planned.messageLimit)}
              </span>
            )}
          </div>
          <Textarea
            id={id}
            value={message}
            minRows={4}
            isInvalid={tooLong || empty}
            aria-describedby={tooLong || empty ? `${id}-err` : undefined}
            onChange={(event) => setMessage(event.target.value)}
          />
          {(tooLong || empty) && (
            <span id={`${id}-err`} role="alert" className="text-xs text-danger-foreground">
              {empty
                ? t("Write the message, or cancel.")
                : t("That is too long for how it is sent. Shorten it to send.")}
            </span>
          )}
        </div>
      )}

      {hasMessage && planned.canReword && (
        <div className="flex items-center gap-2">
          <AssistMark className="size-4 shrink-0 text-foreground-muted" aria-hidden />
          <Input
            value={guidance}
            aria-label={t("How Nova should change the message")}
            placeholder={t("Have Nova reword it: firmer, shorter, mention it is the third ask…")}
            maxLength={600}
            onChange={(event) => setGuidance(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                void reword();
              }
            }}
          />
          <Button
            type="button"
            variant="secondary"
            isLoading={pending === "reword"}
            disabled={guidance.trim() === "" || pending === "send"}
            onClick={() => void reword()}
          >
            {t("Reword")}
          </Button>
        </div>
      )}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          {t("Cancel")}
        </Button>
        <Button
          type="submit"
          isLoading={pending === "send"}
          disabled={tooLong || empty || pending === "reword"}
        >
          {hasMessage ? t("Send") : t("Offer it")}
        </Button>
      </DialogFooter>
    </form>
  );
}
