import { useMemo, type CSSProperties } from "react";

const PIECES = 64;

/** A repeatable 0–1 value from the burst's seed, so a re-render draws the same burst. */
function noise(seed: number, piece: number, channel: number): number {
  const value = Math.sin(seed * 97 + piece * 13.37 + channel * 7.1) * 10000;
  return value - Math.floor(value);
}

/**
 * Confetti in Trenova's colours, raining from the top of the conversation and
 * passing behind the composer, for the moment a change is approved. Each
 * piece falls for three to four and a half seconds; the burst is gone in five.
 */
export function DeskConfetti({ seed }: { seed: number }) {
  const pieces = useMemo(
    () =>
      Array.from({ length: PIECES }, (_, piece) => {
        const n = (channel: number) => noise(seed, piece, channel);
        return {
          x: (n(1) - 0.5) * 900,
          sway: (n(2) - 0.5) * 80,
          rotation: (n(4) - 0.5) * 900,
          duration: 2800 + n(5) * 1600,
          delay: n(6) * 700,
          width: 5 + n(7) * 5,
          height: n(8) > 0.5 ? 10 + n(9) * 6 : 5 + n(9) * 3,
          round: n(10) > 0.8,
        };
      }),
    [seed],
  );

  return (
    <div className="dk-confetti" aria-hidden>
      {pieces.map((piece, index) => (
        <i
          // oxlint-disable-next-line react/no-array-index-key -- a burst's pieces never reorder
          key={index}
          style={
            {
              "--dk-x": `${piece.x}px`,
              "--dk-sway": `${piece.sway}px`,
              "--dk-rot": `${piece.rotation}deg`,
              width: piece.width,
              height: piece.height,
              borderRadius: piece.round ? "50%" : 2,
              animationDuration: `${piece.duration}ms`,
              animationDelay: `${piece.delay}ms`,
            } as CSSProperties
          }
        />
      ))}
    </div>
  );
}
