import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { MicIcon, MicOffIcon, SquareIcon } from "lucide-react";
import { useReducedMotion } from "motion/react";
import { useEffect, useRef } from "react";
import type { DictationAvailability, DictationIssue, DictationPhase } from "./use-dictation";

/** Below this RMS the room is treated as silent and the meter holds still. */
const NOISE_FLOOR = 0.015;

/** The RMS a raised speaking voice reaches, where the meter tops out. */
const FULL_SCALE = 0.18;

/** How much each bar answers the level, so the three read as a voice rather than a block. */
const BAR_WEIGHTS = [0.6, 1, 0.75] as const;

/** The short line the composer shows under the box, and the fuller one on the mic. */
export function dictationIssueText(
  issue: DictationIssue,
  t: TranslateFn,
): { short: string; detail: string } {
  switch (issue) {
    case "denied":
      return {
        short: t("Microphone blocked. Allow it in the address bar."),
        detail: t(
          "Your browser is blocking the microphone for this site. Allow it from the icon in the address bar, then try again.",
        ),
      };
    case "blocked":
      return {
        short: t("The microphone is turned off for this site."),
        detail: t(
          "This site's security settings turn the microphone off, so dictation cannot run here. Ask an administrator.",
        ),
      };
    case "service":
      return {
        short: t("Speech recognition is turned off in this browser."),
        detail: t(
          "This browser's speech service is turned off. On a Mac, check that Siri and Dictation are on in System Settings.",
        ),
      };
    case "no-microphone":
      return {
        short: t("No microphone found."),
        detail: t("No microphone was found. Connect one and try again."),
      };
    case "microphone-busy":
      return {
        short: t("The microphone is in use by another app."),
        detail: t("Another app is using the microphone. Close it and try again."),
      };
    case "network":
      return {
        short: t("This browser has no speech service. Try Chrome or Edge."),
        detail: t(
          "This browser cannot reach a speech recognition service. Brave, Arc and some other Chromium browsers leave it out; Chrome, Edge and Safari include it.",
        ),
      };
    case "no-speech":
      return {
        short: t("Didn't catch that. Try again."),
        detail: t("Nothing was heard. Try again a little closer to the microphone."),
      };
    case "language":
      return {
        short: t("Dictation does not support this language."),
        detail: t("Speech recognition does not support your browser's language."),
      };
    case "interrupted":
      return {
        short: t("Dictation stopped unexpectedly."),
        detail: t("Something else took the microphone. Try again."),
      };
    default:
      return {
        short: t("Dictation could not start. Try again."),
        detail: t("Dictation could not start. Try again."),
      };
  }
}

/** Why the mic is disabled, for the tooltip on a control that cannot be used here. */
export function dictationUnavailableText(
  availability: Exclude<DictationAvailability, "available">,
  t: TranslateFn,
): string {
  switch (availability) {
    case "insecure":
      return t("Dictation needs a secure (HTTPS) connection.");
    case "blocked":
      return t("The microphone is turned off for this site, so dictation cannot run here.");
    default:
      return t("Dictation is not available in this browser. Try Chrome, Edge or Safari.");
  }
}

/**
 * The microphone's level as three bars, drawn straight to the DOM from an
 * analyser so the composer does not re-render sixty times a second. It moves
 * only while someone is speaking: under the noise floor every bar rests at
 * its minimum, so a quiet room is a still control.
 */
function DictationMeter({ stream }: { stream: MediaStream }) {
  const barsRef = useRef<(HTMLSpanElement | null)[]>([]);

  useEffect(() => {
    if (typeof AudioContext === "undefined") {
      return;
    }
    const context = new AudioContext();
    const source = context.createMediaStreamSource(stream);
    const analyser = context.createAnalyser();
    analyser.fftSize = 512;
    analyser.smoothingTimeConstant = 0.6;
    source.connect(analyser);
    void context.resume().catch(() => undefined);

    const samples = new Float32Array(analyser.fftSize);
    let frame = 0;
    let shown = -1;
    const draw = () => {
      analyser.getFloatTimeDomainData(samples);
      let sum = 0;
      for (const sample of samples) {
        sum += sample * sample;
      }
      const rms = Math.sqrt(sum / samples.length);
      const level =
        rms < NOISE_FLOOR ? 0 : Math.min(1, (rms - NOISE_FLOOR) / (FULL_SCALE - NOISE_FLOOR));
      const rounded = Math.round(level * 20) / 20;
      if (rounded !== shown) {
        shown = rounded;
        barsRef.current.forEach((bar, index) => {
          if (bar) {
            bar.style.transform = `scaleY(${0.25 + 0.75 * rounded * BAR_WEIGHTS[index]})`;
          }
        });
      }
      frame = window.requestAnimationFrame(draw);
    };
    frame = window.requestAnimationFrame(draw);

    return () => {
      window.cancelAnimationFrame(frame);
      source.disconnect();
      void context.close().catch(() => undefined);
    };
  }, [stream]);

  return (
    <span aria-hidden className="flex h-3 items-center gap-0.5">
      {BAR_WEIGHTS.map((weight, index) => (
        <span
          key={weight}
          ref={(element) => {
            barsRef.current[index] = element;
          }}
          style={{ transform: "scaleY(0.25)" }}
          className="h-3 w-0.5 origin-center rounded-full bg-current transition-transform duration-75"
        />
      ))}
    </span>
  );
}

export type DictationControlProps = {
  availability: DictationAvailability;
  phase: DictationPhase;
  issue: DictationIssue | null;
  stream: MediaStream | null;
  disabled?: boolean;
  onToggle: () => void;
};

/**
 * The microphone in the composer's control row.
 *
 * At rest it is a plain icon button. Clicked, it answers at once: while the
 * browser asks for the microphone it is held pressed; once listening it
 * becomes a filled pill in the danger tone with a stop square and the live
 * level, so there is no doubt the room is being heard. Where dictation cannot
 * run it stays in the row, disabled, and its tooltip says why — a button that
 * vanishes on one browser is a bug report waiting to be filed.
 */
export function DictationControl({
  availability,
  phase,
  issue,
  stream,
  disabled = false,
  onToggle,
}: DictationControlProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  if (availability !== "available") {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              size="icon-sm"
              variant="ghost"
              disabled
              focusableWhenDisabled
              className="text-foreground-subtle cursor-not-allowed rounded-full hover:bg-transparent hover:text-foreground-subtle"
              aria-label={t("Dictate")}
            />
          }
        >
          <MicOffIcon className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{dictationUnavailableText(availability, t)}</TooltipContent>
      </Tooltip>
    );
  }

  const listening = phase === "listening";
  const starting = phase === "starting";
  const label = phase === "idle" ? t("Dictate") : t("Stop dictating");

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            size={listening ? "sm" : "icon-sm"}
            variant="ghost"
            disabled={disabled}
            aria-pressed={phase !== "idle"}
            aria-busy={starting || undefined}
            aria-label={label}
            onClick={onToggle}
            className={cn(
              "rounded-full transition-[background-color,color,width,padding]",
              listening
                ? "bg-danger-subtle text-danger-subtle-foreground hover:bg-danger-subtle hover:text-danger-subtle-foreground gap-1.5 px-2"
                : starting
                  ? "bg-surface-active text-foreground"
                  : issue !== null && issue !== "no-speech"
                    ? "text-danger-foreground hover:text-danger-foreground"
                    : "text-foreground-muted hover:text-foreground",
            )}
          />
        }
      >
        {listening ? (
          <>
            <SquareIcon className="size-2.5 fill-current" />
            {stream && !reduceMotion ? (
              <DictationMeter stream={stream} />
            ) : (
              <span className="text-xs">{t("Listening")}</span>
            )}
          </>
        ) : issue !== null && issue !== "no-speech" ? (
          <MicOffIcon className="size-4" />
        ) : (
          <MicIcon className="size-4" />
        )}
      </TooltipTrigger>
      <TooltipContent>
        {issue !== null && phase === "idle" ? dictationIssueText(issue, t).detail : label}
      </TooltipContent>
    </Tooltip>
  );
}
