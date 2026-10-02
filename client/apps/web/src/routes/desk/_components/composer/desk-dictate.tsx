import type { ComposerDictation } from "@/components/assistant/use-composer-dictation";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useState } from "react";
import { DeskIcon } from "../desk-icons";

function elapsedLabel(seconds: number): string {
  const whole = Math.floor(seconds);
  return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, "0")}`;
}

/** How long the microphone has been open, counted while it listens. */
function useElapsed(running: boolean): number {
  const [seconds, setSeconds] = useState(0);
  useEffect(() => {
    if (!running) {
      return;
    }
    const started = performance.now();
    const timer = window.setInterval(
      () => setSeconds((performance.now() - started) / 1000),
      100,
    );
    return () => {
      window.clearInterval(timer);
      setSeconds(0);
    };
  }, [running]);
  return seconds;
}

/**
 * The microphone beside send. While it listens it becomes a red pill with a
 * moving wave and the time so far; the words appear in the box as they are
 * heard, and a press stops it. Where the page cannot listen, the button says
 * why instead of doing nothing.
 */
export function DeskDictate({
  dictation,
  disabled = false,
}: {
  dictation: ComposerDictation;
  disabled?: boolean;
}) {
  const t = useT();
  const listening = dictation.phase !== "idle";
  const seconds = useElapsed(listening);

  if (!listening) {
    const unavailable = !dictation.supported;
    return (
      <button
        type="button"
        className="dk-ib"
        title={unavailable ? t("Dictation isn't available in this browser") : t("Dictate")}
        aria-label={t("Dictate")}
        disabled={disabled || unavailable}
        onClick={dictation.toggleFromDraft}
      >
        <DeskIcon name="mic" size={15} />
      </button>
    );
  }

  return (
    <button
      type="button"
      className="dk-dict"
      title={t("Stop dictating")}
      aria-label={t("Stop dictating")}
      onClick={dictation.toggleFromDraft}
    >
      <span className="dk-wave">
        {[0, 1, 2, 3, 4].map((bar) => (
          <i key={bar} style={{ animationDelay: `${bar * 110}ms` }} />
        ))}
      </span>
      <span className="dk-mono">{elapsedLabel(seconds)}</span>
      <span className="dk-dict-x" />
    </button>
  );
}
