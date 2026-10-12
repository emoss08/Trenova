import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState } from "react";
import { DeskIcon } from "../desk-icons";

const actClass =
  "dk-act relative min-w-7 gap-1.5 px-1.75 text-dsk-subtle has-[>svg]:px-1.75 hover:bg-dsk-hover hover:text-dsk-fg";
const actOnClass = "text-dsk-blue hover:text-dsk-blue [&_svg_path]:fill-current";

/** How long Copy holds its tick. */
const COPIED_MS = 1400;

function speechAvailable(): boolean {
  return typeof window !== "undefined" && "speechSynthesis" in window;
}

/**
 * The small actions under a reply, shown on hover: pin it as a chapter of the
 * conversation, copy it, or have it read aloud. A pinned reply says which
 * chapter it is; the row stays in view while one of them is on. A surface
 * without chapters leaves `onTogglePin` out and offers Copy and Read aloud.
 */
export function DeskMessageActions({
  text,
  chapter,
  onTogglePin,
}: {
  text: string;
  /** Which chapter this reply is, from 1, or 0 when it is not pinned. */
  chapter: number;
  onTogglePin?: () => void;
}) {
  const t = useT();
  const [copied, setCopied] = useState(false);
  const [reading, setReading] = useState(false);
  const pinned = chapter > 0;

  useEffect(() => {
    if (!copied) {
      return;
    }
    const timer = window.setTimeout(() => setCopied(false), COPIED_MS);
    return () => window.clearTimeout(timer);
  }, [copied]);

  useEffect(
    () => () => {
      if (reading && speechAvailable()) {
        window.speechSynthesis.cancel();
      }
    },
    [reading],
  );

  const copy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
  };

  const read = () => {
    if (!speechAvailable()) {
      return;
    }
    if (reading) {
      window.speechSynthesis.cancel();
      setReading(false);
      return;
    }
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.onend = () => setReading(false);
    utterance.onerror = () => setReading(false);
    window.speechSynthesis.cancel();
    window.speechSynthesis.speak(utterance);
    setReading(true);
  };

  return (
    <div className={cn("dk-acts", (pinned || reading || copied) && "dk-stay")}>
      {onTogglePin && (
        <Button
          variant="quiet"
          size="sm"
          className={cn(actClass, pinned && actOnClass)}
          data-tip={pinned ? t("Unpin chapter") : t("Pin as chapter")}
          aria-label={pinned ? t("Unpin chapter") : t("Pin as chapter")}
          aria-pressed={pinned}
          onClick={onTogglePin}
        >
          <DeskIcon name="bookmark" size={14} />
          {pinned && <span className="text-xs font-medium">{t("Chapter {0}", chapter)}</span>}
        </Button>
      )}
      <Button
        variant="quiet"
        size="sm"
        className={cn(actClass, copied && "text-dsk-success-fg hover:text-dsk-success-fg")}
        data-tip={copied ? t("Copied") : t("Copy")}
        aria-label={t("Copy reply")}
        onClick={() => void copy()}
      >
        {copied ? (
          <DeskIcon name="check" size={14} stroke={2.2} />
        ) : (
          <DeskIcon name="copy" size={14} />
        )}
      </Button>
      {speechAvailable() && (
        <Button
          variant="quiet"
          size="sm"
          className={cn(actClass, reading && actOnClass)}
          data-tip={reading ? t("Stop reading") : t("Read aloud")}
          aria-label={reading ? t("Stop reading") : t("Read aloud")}
          aria-pressed={reading}
          onClick={read}
        >
          {reading ? (
            <span className="dk-eq" aria-hidden>
              {[0, 1, 2, 3].map((bar) => (
                <i key={bar} style={{ animationDelay: `${bar * 120}ms` }} />
              ))}
            </span>
          ) : (
            <DeskIcon name="speaker" size={14} />
          )}
        </Button>
      )}
    </div>
  );
}
