import type { TriageHandOff, TriageUser } from "@/lib/graphql/inbox";
import { senderFirstName, senderName } from "@/lib/triage/message-state";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
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
 * The end of the day: what is left becomes what carries over to tomorrow, soonest
 * due first, with the hand-off to whoever is on call. Once handed off each
 * row wears their mark, and the briefing says they will see them first thing.
 */
export function CarryingOver({
  items,
  onCall,
  now,
  onOpenMessage,
  onHandOff,
  canHandOff,
}: {
  items: readonly DeckItem[];
  onCall: TriageUser | null;
  now: number;
  onOpenMessage: (id: string) => void;
  onHandOff: (to: TriageUser) => Promise<TriageHandOff>;
  /** Whether the reader may give messages to someone else. */
  canHandOff: boolean;
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
    <div className="flex flex-col overflow-hidden rounded-[8px] border border-dsk-b bg-dsk-card">
      {(done ? owner !== null : onCall !== null && rows.length > 0) && (
        <div className="flex min-h-11 items-center justify-end gap-2 border-b border-dsk-b-sub px-3 py-2">
          {done && owner !== null ? (
            <span role="status" className="text-dsk-base text-dsk-success-fg">
              {t(
                "Handed to {0} · {1, plural, one {they'll see it first thing} other {they'll see them first thing}}",
                senderFirstName({ fromName: owner.name, fromAddress: owner.username }),
                rows.length,
              )}
            </span>
          ) : canHandOff && onCall !== null && rows.length > 0 ? (
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
      )}
      {refusal !== null && (
        <p
          role="alert"
          className="border-b border-dsk-b-sub px-3 py-2 text-dsk-base text-dsk-error"
        >
          {refusal}
        </p>
      )}
      {onCall === null && rows.length > 0 && (
        <p className="border-b border-dsk-b-sub px-3 py-2 text-dsk-base text-dsk-subtle">
          {t("Nobody is on call tonight. Set someone in Triage settings to hand these off.")}
        </p>
      )}
      {rows.length === 0 && (
        <p className="px-3 py-2.5 text-dsk-base text-dsk-subtle">{t("Nothing carries over.")}</p>
      )}
      <ul className="flex flex-col p-1 empty:hidden">
        {rows.map((row) => {
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
                className={cn(
                  "grid w-full grid-cols-[8px_minmax(0,11rem)_minmax(0,1fr)_auto_auto] items-center gap-2.5 rounded-[6px] ui-focus-ring px-2 py-1.5 text-left text-dsk-base transition-opacity hover:bg-dsk-hover",
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
                <b className="truncate font-medium text-dsk-fg">{senderName(row)}</b>
                <span className="truncate text-dsk-sm text-dsk-subtle @max-2xl:hidden">{next}</span>
                <em
                  className={cn(
                    "text-xs not-italic tabular-nums",
                    due?.state === "over" ? "text-dsk-warn-fg" : "text-dsk-subtle",
                  )}
                >
                  {due === null ? "" : dueText(t, due)}
                </em>
                <span className="flex w-4 justify-end">
                  {done && rowOwner !== null && <UserMark user={rowOwner} size="xs" />}
                </span>
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
