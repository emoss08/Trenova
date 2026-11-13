import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import type {
  FreightArea,
  FreightMove,
  FreightOutlook,
  FreightPositioning,
} from "@/lib/graphql/freight-outlook";
import { CheckIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatCentsWhole } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { moveKey, windowOfSlot } from "./outlook-model";
import { PanelSection, slotLabel, slotRange } from "./outlook-parts";
import { useOutlookActions } from "./use-outlook-actions";

const CONSOLE_PATH = "/dispatch/console";
/** How long the pointer rests on a move before its card opens, and lingers after it leaves. */
const CARD_DELAY_MS = 250;
const CARD_CLOSE_MS = 80;

function consolePath(moveId: string | null | undefined, workerId: string | null | undefined) {
  const params = new URLSearchParams();
  if (moveId) params.set("move", moveId);
  if (workerId) params.set("worker", workerId);
  return `${CONSOLE_PATH}?${params.toString()}`;
}

function signedCents(cents: number): string {
  if (cents === 0) return formatCentsWhole(0);
  return cents < 0 ? `−${formatCentsWhole(-cents)}` : `+${formatCentsWhole(cents)}`;
}

type MoveText = {
  title: ReactNode;
  plain: string;
  meta: string;
  detail: string;
  parts: { label: string; cents: number }[];
  foot: string;
};

function useMoveText(onOpenArea: (cell: number, slotStart: number) => void) {
  const t = useT();
  const rt = useRichT();
  const worker = (chunks: ReactNode) => <span className="font-medium">{chunks}</span>;
  const area = (move: FreightMove) => (chunks: ReactNode) => (
    <button
      type="button"
      onClick={(event) => {
        event.stopPropagation();
        onOpenArea(move.toCell, move.slotStart);
      }}
      className="cursor-pointer rounded-sm ui-focus-ring underline decoration-border-strong underline-offset-4 hover:decoration-brand"
    >
      {chunks}
    </button>
  );

  return (move: FreightMove): MoveText => {
    const when = slotLabel(move.slotStart);
    const short = Math.max(1, Math.round(move.short));
    const miles = Math.round(move.miles);
    switch (move.kind) {
      case "Send":
        return {
          title: rt(
            "Send <w>{0}</w> to <a>{1}</a>",
            { w: worker, a: area(move) },
            move.workerName,
            move.toLabel,
          ),
          plain: t("Send {0} to {1}", move.workerName, move.toLabel),
          meta: move.fromLabel
            ? t(
                "From {0} · free {1} · {2} mi",
                move.fromLabel,
                slotLabel(move.freeAt ?? move.slotStart),
                miles,
              )
            : t("Free {0} · {1} mi", slotLabel(move.freeAt ?? move.slotStart), miles),
          detail: t(
            "Free {0}, {1, plural, one {# mile} other {# miles}} empty, there by {2}. {3} is {4, plural, one {# truck} other {# trucks}} short from {5}.",
            slotLabel(move.freeAt ?? move.slotStart),
            miles,
            slotLabel(move.arriveAt ?? move.slotStart),
            move.toLabel,
            short,
            when,
          ),
          parts: [
            { label: t("{0} gains a driver", move.toLabel), cents: move.dropCents },
            {
              label: move.fromLabel ? t("{0} loses one", move.fromLabel) : t("Where they are now"),
              cents: -move.stayCents,
            },
            {
              label: t("{0, plural, one {# empty mile} other {# empty miles}}", miles),
              cents: -move.emptyCostCents,
            },
          ],
          foot: t("Sending tells the driver in the app and counts them at {0}.", move.toLabel),
        };
      case "Book":
        return {
          title: rt(
            "Book <w>{0}</w> toward <a>{1}</a>",
            { w: worker, a: area(move) },
            move.workerName,
            move.toLabel,
          ),
          plain: t("Book {0} toward {1}", move.workerName, move.toLabel),
          meta: t(
            "{0, plural, one {# load} other {# loads}} from {1} end nearby by {2}",
            Math.max(1, move.laneMoveIds.length),
            move.fromLabel,
            when,
          ),
          detail: t(
            "{0} picks up {1, plural, one {# mile} other {# miles}} away and delivers near {2} before {3}, saving {4} against running empty.",
            move.proNumber,
            miles,
            move.toLabel,
            when,
            formatCentsWhole(move.savedCents),
          ),
          parts: [
            { label: t("{0} gains a driver", move.toLabel), cents: move.dropCents },
            {
              label: move.fromLabel ? t("{0} loses one", move.fromLabel) : t("Where they are now"),
              cents: -move.stayCents,
            },
            {
              label: t("{0, plural, one {# mile} other {# miles}} to the pickup", miles),
              cents: -move.emptyCostCents,
            },
          ],
          foot: t("Opens the console with the load and driver picked."),
        };
      case "Keep":
        return {
          title: rt(
            "Keep <w>{0}</w> in <a>{1}</a>",
            { w: worker, a: area(move) },
            move.workerName,
            move.toLabel,
          ),
          plain: t("Keep {0} in {1}", move.workerName, move.toLabel),
          meta: t("{0} takes their next load, {1}", move.partnerName, move.proNumber),
          detail: move.partnerLabel
            ? t(
                "Their next load, {0}, would leave {1} {2, plural, one {# truck} other {# trucks}} short from {3}. Hand it to {4}, free in {5} {6}.",
                move.proNumber,
                move.toLabel,
                short,
                when,
                move.partnerName,
                move.partnerLabel,
                slotLabel(move.partnerFreeAt ?? move.slotStart),
              )
            : t(
                "Their next load, {0}, would leave {1} {2, plural, one {# truck} other {# trucks}} short from {3}. Hand it to {4}, free {5}.",
                move.proNumber,
                move.toLabel,
                short,
                when,
                move.partnerName,
                slotLabel(move.partnerFreeAt ?? move.slotStart),
              ),
          parts: [
            { label: t("{0} keeps a driver", move.toLabel), cents: move.dropCents },
            {
              label: move.partnerLabel
                ? t("{0} loses {1}", move.partnerLabel, move.partnerName)
                : t("Where {0} is now", move.partnerName),
              cents: -move.stayCents,
            },
            {
              label: t(
                "{0}'s {1, plural, one {# empty mile} other {# empty miles}}",
                move.partnerName,
                Math.round(move.partnerMiles),
              ),
              cents: -move.emptyCostCents,
            },
          ],
          foot: t("Opens the console with the load and {0} picked.", move.partnerName),
        };
      case "Call":
        return {
          title: rt("Call <w>{0}</w>", { w: worker }, move.customerName),
          plain: t("Call {0}", move.customerName),
          meta: t(
            "{0} of the usual {1} in for {2}, {3}",
            move.tendered,
            Math.round(move.usual),
            move.toLabel,
            when,
          ),
          detail: t(
            "{0} of the usual {1, plural, one {# load} other {# loads}} in the next day are in; by now {2} usually are, so the rest may be going to another carrier.",
            move.tendered,
            Math.round(move.usual),
            Math.round(move.byNow),
          ),
          parts: [],
          foot: t("Opens the customer record."),
        };
    }
  };
}

function MoveAction({
  move,
  positioning,
  primary,
}: {
  move: FreightMove;
  positioning: FreightPositioning | undefined;
  primary: boolean;
}) {
  const t = useT();
  const { send, callOff } = useOutlookActions();
  const { allowed: canSend } = usePermission(Resource.DriverPositioning, Operation.Create);
  const { allowed: canCallOff } = usePermission(Resource.DriverPositioning, Operation.Cancel);
  const variant = primary ? "default" : "outline";

  switch (move.kind) {
    case "Send":
      if (positioning) {
        return canCallOff ? (
          <Button
            type="button"
            size="xs"
            variant="outline"
            className="border-success-border bg-success-subtle text-success-foreground hover:bg-success-subtle-hover"
            isLoading={callOff.isPending}
            onClick={() => callOff.mutateAsync(positioning.id)}
          >
            {t("On the way · Undo")}
          </Button>
        ) : (
          <Badge variant="success">{t("On the way")}</Badge>
        );
      }
      return canSend ? (
        <Button
          type="button"
          size="xs"
          variant={variant}
          isLoading={send.isPending}
          onClick={() => send.mutateAsync(move)}
        >
          {t("Send")}
        </Button>
      ) : null;
    case "Book":
      return (
        <Button
          size="xs"
          variant={variant}
          nativeButton={false}
          render={<Link to={consolePath(move.moveId, move.workerId)} />}
        >
          {t("See loads")}
        </Button>
      );
    case "Keep":
      return (
        <Button
          size="xs"
          variant={variant}
          nativeButton={false}
          render={<Link to={consolePath(move.moveId, move.partnerId)} />}
        >
          {t("Review")}
        </Button>
      );
    case "Call":
      return move.customerId ? (
        <Button
          size="xs"
          variant={variant}
          nativeButton={false}
          render={<Link to={recordPath("customer", move.customerId)} />}
        >
          {t("Open")}
        </Button>
      ) : null;
  }
}

function MoveCard({ move, text }: { move: FreightMove; text: MoveText }) {
  const t = useT();
  const net = move.kind === "Call" ? 0 : move.netCents;
  return (
    <div className="flex flex-col gap-2.25">
      <div>
        <p className="text-sm leading-snug font-medium">{text.plain}</p>
        <p className="mt-0.5 text-xs text-foreground-subtle">
          {t("For {0}", slotRange(windowOfSlot(move)))}
        </p>
      </div>
      <p className="text-xs leading-normal text-muted-foreground">{text.detail}</p>
      <dl className="flex flex-col gap-0.75 border-t border-border-subtle pt-2 text-xs">
        {move.kind === "Call" ? (
          <>
            <div className="flex justify-between gap-3 text-muted-foreground">
              <dt>{t("Tendered so far")}</dt>
              <dd className="text-foreground tabular-nums">
                {t("{0} of {1}", move.tendered, Math.round(move.usual))}
              </dd>
            </div>
            <div className="flex justify-between gap-3 text-muted-foreground">
              <dt>{t("Usually in by now")}</dt>
              <dd className="text-foreground tabular-nums">
                {formatNumber(Math.round(move.byNow))}
              </dd>
            </div>
            <div className="mt-0.5 flex justify-between gap-3 border-t border-border-subtle pt-1 font-medium text-foreground">
              <dt>{t("Still expected")}</dt>
              <dd className="text-warning-foreground tabular-nums">
                {t("{0, plural, one {# load} other {# loads}}", Math.round(move.loads))}
              </dd>
            </div>
          </>
        ) : (
          <>
            {text.parts.map((part) => (
              <div key={part.label} className="flex justify-between gap-3 text-muted-foreground">
                <dt>{part.label}</dt>
                <dd className="text-foreground tabular-nums">{signedCents(part.cents)}</dd>
              </div>
            ))}
            <div className="mt-0.5 flex justify-between gap-3 border-t border-border-subtle pt-1 font-medium text-foreground">
              <dt>{t("Net")}</dt>
              <dd className="text-success-foreground tabular-nums">{signedCents(net)}</dd>
            </div>
          </>
        )}
      </dl>
      <p className="text-2xs text-foreground-subtle">{text.foot}</p>
    </div>
  );
}

function MoveRow({
  move,
  rank,
  text,
  positioning,
  hot,
  onHot,
  onDismiss,
}: {
  move: FreightMove;
  rank: number;
  text: MoveText;
  positioning: FreightPositioning | undefined;
  hot: boolean;
  onHot: (key: string | null) => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const key = moveKey(move);
  return (
    <HoverCard>
      <HoverCardTrigger
        delay={CARD_DELAY_MS}
        closeDelay={CARD_CLOSE_MS}
        render={
          <li
            tabIndex={0}
            aria-label={text.plain}
            onPointerEnter={() => onHot(key)}
            onPointerLeave={() => onHot(null)}
            onFocus={() => onHot(key)}
            onBlur={(event) => {
              if (!event.currentTarget.contains(event.relatedTarget)) onHot(null);
            }}
          />
        }
        className={cn(
          "group/move grid animate-rise grid-cols-[1.125rem_minmax(0,1fr)_auto_auto] items-center gap-x-2.5 border-b border-border-subtle px-3.5 py-2 outline-none last:border-b-0",
          "focus-visible:shadow-[inset_2px_0_0_var(--color-ring)]",
          hot ? "bg-surface-hover" : "hover:bg-surface-hover",
        )}
      >
        <span className="grid size-4.5 place-items-center rounded-[5px] bg-sunken font-mono text-2xs font-medium text-muted-foreground">
          {rank}
        </span>
        <span className="min-w-0">
          <span className="block truncate text-sm leading-4.5">{text.title}</span>
          <span className="block truncate text-xs leading-4 text-foreground-subtle">
            {text.meta}
          </span>
        </span>
        <span
          className={cn(
            "text-right text-sm font-medium whitespace-nowrap tabular-nums",
            move.kind !== "Call" && "text-success-foreground",
          )}
        >
          {move.kind === "Call"
            ? t("{0, plural, one {# load} other {# loads}}", Math.round(move.loads))
            : signedCents(move.netCents)}
        </span>
        <span className="flex items-center gap-0.5">
          <MoveAction move={move} positioning={positioning} primary={rank === 1} />
          <Button
            type="button"
            size="icon-xs"
            variant="ghost"
            aria-label={t("Not now")}
            title={t("Not now")}
            onClick={onDismiss}
            className="text-foreground-subtle opacity-0 group-focus-within/move:opacity-100 group-hover/move:opacity-100"
          >
            <XCloseIcon className="size-3" />
          </Button>
        </span>
      </HoverCardTrigger>
      <HoverCardContent
        inverted={false}
        side="left"
        align="start"
        sideOffset={8}
        alignOffset={-6}
        className="w-76 px-3.5 py-3"
      >
        <MoveCard move={move} text={text} />
      </HoverCardContent>
    </HoverCard>
  );
}

/**
 * A driver already sent: the move that suggested them is gone once the forecast counts them at
 * the target, so the brief keeps them here, with a way to call it off, until they arrive.
 */
function SentRow({ positioning }: { positioning: FreightPositioning }) {
  const t = useT();
  const rt = useRichT();
  const { callOff } = useOutlookActions();
  const { allowed: canCallOff } = usePermission(Resource.DriverPositioning, Operation.Cancel);
  return (
    <li className="grid animate-rise grid-cols-[1.125rem_minmax(0,1fr)_auto_auto] items-center gap-x-2.5 border-b border-border-subtle px-3.5 py-2 last:border-b-0">
      <span className="grid size-4.5 place-items-center rounded-[5px] bg-success-subtle text-success-foreground">
        <CheckIcon className="size-3" />
      </span>
      <span className="min-w-0">
        <span className="block truncate text-sm leading-4.5">
          {rt(
            "Sent <w>{0}</w> to {1}",
            { w: (chunks) => <span className="font-medium">{chunks}</span> },
            positioning.workerName,
            positioning.toLabel,
          )}
        </span>
        <span className="block truncate text-xs leading-4 text-foreground-subtle">
          {positioning.fromLabel
            ? t(
                "From {0} · there by {1} · {2} mi",
                positioning.fromLabel,
                slotLabel(positioning.arriveAt),
                Math.round(positioning.miles),
              )
            : t("There by {0}", slotLabel(positioning.arriveAt))}
        </span>
      </span>
      <span className="text-right text-sm font-medium whitespace-nowrap text-success-foreground tabular-nums">
        {signedCents(positioning.valueCents)}
      </span>
      {canCallOff ? (
        <Button
          type="button"
          size="xs"
          variant="outline"
          className="border-success-border bg-success-subtle text-success-foreground hover:bg-success-subtle-hover"
          isLoading={callOff.isPending && callOff.variables === positioning.id}
          onClick={() => callOff.mutateAsync(positioning.id)}
        >
          {t("On the way · Undo")}
        </Button>
      ) : (
        <Badge variant="success">{t("On the way")}</Badge>
      )}
    </li>
  );
}

/**
 * What to do next, best first on one dollar scale: send an empty driver, book a load that ends
 * where freight is coming, keep a driver who would leave a short area, or call a customer behind
 * its usual pace. A row is two lines; resting on one opens why it is worth what it is, and draws
 * it on the board.
 */
export function OutlookMoves({
  outlook,
  moves,
  sent,
  unreached,
  dismissed,
  hotKey,
  onHot,
  onDismiss,
  onRestore,
  onOpenArea,
}: {
  outlook: FreightOutlook;
  moves: readonly FreightMove[];
  sent: readonly FreightPositioning[];
  unreached: readonly FreightArea[];
  dismissed: number;
  hotKey: string | null;
  onHot: (key: string | null) => void;
  onDismiss: (key: string) => void;
  onRestore: () => void;
  onOpenArea: (cell: number, slotStart: number) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const text = useMoveText(onOpenArea);
  const positionings = new Map(outlook.positionings.map((p) => [`${p.workerId}:${p.toCell}`, p]));
  const worth = moves.reduce((sum, move) => sum + (move.kind === "Call" ? 0 : move.netCents), 0);
  const first = outlook.slotStarts[0] ?? 0;

  return (
    <PanelSection
      id="outlook-moves"
      title={t("Next moves")}
      figure={
        moves.length > 0 ? (
          <span className="text-success-foreground">{signedCents(worth)}</span>
        ) : null
      }
    >
      {moves.length === 0 && sent.length === 0 ? (
        <p className="text-xs text-foreground-subtle">
          {t("Nothing is worth moving in this range.")}
        </p>
      ) : (
        <ScrollArea
          maskVariant="card"
          maskHeight={18}
          className="-mx-3.5 max-h-51.5"
          viewportClassName="overscroll-contain"
        >
          <ol aria-label={t("Next moves")}>
            {moves.map((move, index) => {
              const key = moveKey(move);
              return (
                <MoveRow
                  key={key}
                  move={move}
                  rank={index + 1}
                  text={text(move)}
                  positioning={
                    move.workerId ? positionings.get(`${move.workerId}:${move.toCell}`) : undefined
                  }
                  hot={hotKey === key}
                  onHot={onHot}
                  onDismiss={() => onDismiss(key)}
                />
              );
            })}
            {sent.map((positioning) => (
              <SentRow key={positioning.id} positioning={positioning} />
            ))}
          </ol>
        </ScrollArea>
      )}
      {unreached[0] ? (
        <p className="rounded-lg border border-dashed border-border px-3 py-2.25 text-xs text-foreground-subtle">
          {rt(
            "No free driver reaches <a>{0}</a>{1, plural, =0 {} one { or # other area} other { or # other areas}} in time. Open it to see who comes closest.",
            {
              a: (chunks) => (
                <button
                  type="button"
                  onClick={() => onOpenArea(unreached[0].cell, first)}
                  className="cursor-pointer rounded-sm ui-focus-ring text-foreground underline decoration-border-strong underline-offset-4 hover:decoration-brand"
                >
                  {chunks}
                </button>
              ),
            },
            unreached[0].label,
            unreached.length - 1,
          )}
        </p>
      ) : null}
      {dismissed > 0 ? (
        <Button type="button" size="xs" variant="ghost" className="self-start" onClick={onRestore}>
          {t("Show {0, plural, one {# hidden move} other {# hidden moves}}", dismissed)}
        </Button>
      ) : null}
    </PanelSection>
  );
}
