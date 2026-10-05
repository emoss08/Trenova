import { useTypewriter } from "@/hooks/use-typewriter";
import { segmentsText, type NovaSegment } from "@/lib/onboarding-copy";
import { useT } from "@trenova/shared/i18n/use-t";
import { Fragment, type ReactNode } from "react";

function revealSegments(segments: readonly NovaSegment[], count: number): ReactNode[] {
  let left = count;
  const out: ReactNode[] = [];
  segments.forEach((segment, index) => {
    if (left <= 0) {
      return;
    }
    const part = segment.text.slice(0, left);
    left -= segment.text.length;
    out.push(segment.bold ? <b key={index}>{part}</b> : <Fragment key={index}>{part}</Fragment>);
  });
  return out;
}

/**
 * One thing Nova says: the guide's header row, a short "Typing" pause, then the line
 * typed out. A line that has been typed once renders whole and never types again.
 */
export function NovaMessage({
  segments,
  animate,
  onDone,
}: {
  segments: readonly NovaSegment[];
  animate: boolean;
  onDone?: () => void;
}) {
  const t = useT();
  const text = segmentsText(segments);
  const { revealed, thinking, typing } = useTypewriter(text, { animate, onDone });

  return (
    <div>
      <div className="nv-who">
        <span className="nv-mark" data-busy={thinking || typing} aria-hidden="true" />
        <b>{t("Nova")}</b>
        <span>{t("Setup guide")}</span>
      </div>
      {thinking ? (
        <div className="nv-thk" aria-hidden="true">
          {t("Typing")}
        </div>
      ) : (
        <p className="nv-prose" aria-hidden={typing || undefined}>
          {revealSegments(segments, revealed)}
          {typing ? <span className="nv-caret" /> : null}
        </p>
      )}
    </div>
  );
}
