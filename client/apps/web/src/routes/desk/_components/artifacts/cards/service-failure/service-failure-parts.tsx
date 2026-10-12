import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ClockEvent, ClockGap, EdiCode } from "./service-failure-card";

/**
 * The failure's clock read left to right along one track: each stretch as
 * wide as its share of the time, in the colour of its window, named above the
 * track, and each moment where two stretches meet, its time and what it was
 * beneath. A stretch is never drawn narrower than its name.
 */
export function MissClock({ clock }: { clock: readonly (ClockEvent | ClockGap)[] }) {
  const t = useT();
  return (
    <ol className="dk-svf-clock" aria-label={t("What happened")}>
      {clock.map((entry) =>
        entry.kind === "event" ? (
          <li key={entry.key} className={cn("dk-svf-ev", entry.state && `dk-${entry.state}`)}>
            <i aria-hidden />
            <span className="dk-svf-at">
              <b>{entry.time}</b>
              <span className="dk-svf-what">{entry.label}</span>
              {entry.day && <span className="dk-svf-day">{entry.day}</span>}
            </span>
          </li>
        ) : (
          <li
            key={entry.key}
            className={cn("dk-svf-gap", `dk-${entry.window}`)}
            style={{ flexGrow: Number(entry.share.toFixed(3)) }}
          >
            <span className="dk-svf-span">{entry.text}</span>
            <i aria-hidden />
          </li>
        ),
      )}
    </ol>
  );
}

/** The codes the failure's 214 sends, a tile each with the code large and where it comes from. */
export function EdiCodes({ codes }: { codes: readonly EdiCode[] }) {
  return (
    <dl className="dk-svf-edi">
      {codes.map((code) => (
        <div key={code.key} className={cn(!code.code && "dk-none")}>
          <dt>{code.label}</dt>
          <dd>
            <code>{code.code ?? "—"}</code>
            <span>{code.note}</span>
          </dd>
        </div>
      ))}
    </dl>
  );
}
