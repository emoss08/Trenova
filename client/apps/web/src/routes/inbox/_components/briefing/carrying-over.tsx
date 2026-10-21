import type { TriageHandOff, TriageUser } from "@/lib/graphql/inbox";
import { senderFirstName, senderName } from "@/lib/triage/message-state";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { USER_FORMATTERS, actionCopy } from "../action-copy";
import { CLASSIFICATION_DOT } from "../classification";
import { dueSpan, dueText } from "../due";
import { inlineErrorText } from "../inline-error";
import { carryOverRows, type DeckItem } from "./briefing-model";

function UserMark({ user, size }: { user: TriageUser; size: "sm" | "xs" }) {
  return (
    <ResolvedUserAvatar
      userId={user.id}
      name={user.name}
      thumbnailUrl={user.thumbnailUrl}
      className={size === "sm" ? "size-4.5" : "size-4"}
      fallbackClassName="text-3xs"
    />
  );
}

/**
 * The end of the day: the deck becomes what carries over to tomorrow, soonest
 * due first, with the hand-off to whoever is on call. Once handed off each
 * row wears their mark, and the briefing says they will see them first thing.
 */
export function CarryingOver({
  items,
  onCall,
  now,
  onOpenMessage,
  onHandOff,
}: {
  items: readonly DeckItem[];
  onCall: TriageUser | null;
  now: number;
  onOpenMessage: (id: string) => void;
  onHandOff: (to: TriageUser) => Promise<TriageHandOff>;
}) {
  const t = useT();
  const rows = useMemo(() => carryOverRows(items), [items]);
  const [handing, setHanding] = useState(false);
  const [handed, setHanded] = useState<TriageUser | null>(null);
  const [refusal, setRefusal] = useState<string | null>(null);
  const owner = handed ?? onCall;
  const allOwned =
    onCall !== null && rows.length > 0 && rows.every((row) => row.owner?.id === onCall.id);
  const done = handed !== null || allOwned;

  const handOff = () => {
    if (onCall === null) {
      return;
    }
    setHanding(true);
    setRefusal(null);
    onHandOff(onCall).then(
      (result) => {
        setHanding(false);
        setHanded(result.to);
      },
      (error: unknown) => {
        setHanding(false);
        setRefusal(inlineErrorText(t, error));
      },
    );
  };

  return (
    <section
      aria-label={t("Carrying over")}
      className="animate-rise flex w-full max-w-140 flex-col gap-0.5"
    >
      <div className="flex items-center gap-2 pb-2.5">
        <h3 className="text-base font-semibold">{t("Carrying over")}</h3>
        <span className="text-foreground-subtle font-mono text-xs tabular-nums">{rows.length}</span>
        <span className="flex-1" />
        {done && owner !== null ? (
          <span
            role="status"
            className="animate-confirm text-success inline-flex items-center gap-1.5 text-sm"
          >
            <CheckIcon aria-hidden className="size-3" strokeWidth={2.4} />
            {t(
              "Handed to {0} · {1, plural, one {they'll see it first thing} other {they'll see them first thing}}",
              senderFirstName({ fromName: owner.name, fromAddress: owner.username }),
              rows.length,
            )}
          </span>
        ) : onCall !== null && rows.length > 0 ? (
          <Button
            size="sm"
            isLoading={handing}
            loadingText={t("Handing off…")}
            onClick={handOff}
            className="pl-1.5"
          >
            <UserMark user={onCall} size="sm" />
            {t(
              "Hand off to {0}",
              senderFirstName({ fromName: onCall.name, fromAddress: onCall.username }),
            )}
          </Button>
        ) : null}
      </div>
      {refusal !== null && (
        <p role="alert" className="text-danger pb-2 text-sm">
          {refusal}
        </p>
      )}
      {onCall === null && rows.length > 0 && (
        <p className="text-foreground-subtle pb-2 text-sm">
          {t("Nobody is on call tonight. Set someone in Triage settings to hand these off.")}
        </p>
      )}
      <ul className="flex flex-col">
        {rows.map((row, index) => {
          const due = row.dueAt === null ? null : dueSpan(row.dueAt, now);
          const next =
            row.action === null
              ? row.subject
              : actionCopy(t, row.action, row, USER_FORMATTERS).title;
          const rowOwner = done ? owner : row.owner;
          return (
            <li key={row.messageId}>
              <button
                type="button"
                onClick={() => onOpenMessage(row.messageId)}
                style={{ animationDelay: `${index * 50}ms` }}
                className={cn(
                  "animate-rise ui-focus-ring hover:bg-surface-hover -mx-3 grid w-[calc(100%+1.5rem)] grid-cols-[8px_minmax(110px,auto)_minmax(0,1fr)_auto_auto] items-center gap-3 rounded-lg px-3 py-2.5 text-left text-base transition-opacity",
                  "@max-2xl:grid-cols-[8px_minmax(0,1fr)_auto_auto]",
                  done && "opacity-70",
                )}
              >
                <span
                  aria-hidden
                  className={cn(
                    "size-1.75 rounded-full",
                    CLASSIFICATION_DOT[row.classification ?? "Other"],
                  )}
                />
                <b className="truncate font-medium">{senderName(row)}</b>
                <span className="text-foreground-subtle truncate @max-2xl:hidden">{next}</span>
                <em
                  className={cn(
                    "text-2xs font-mono not-italic tabular-nums",
                    due?.state === "over" ? "text-warning" : "text-foreground-subtle",
                  )}
                >
                  {due === null ? "" : dueText(t, due)}
                </em>
                <span className="flex w-4 justify-end">
                  {done && rowOwner !== null && (
                    <span className="animate-confirm">
                      <UserMark user={rowOwner} size="xs" />
                    </span>
                  )}
                </span>
              </button>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
