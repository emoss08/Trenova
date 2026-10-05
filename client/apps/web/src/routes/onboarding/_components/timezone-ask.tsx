import { SelectField } from "@/components/fields/select-field";
import { timezoneGroupedChoices } from "@/lib/choices";
import { isTypingTarget, numberKeyIndex } from "@/lib/dom";
import { onboardingTimezoneTiles, onboardingTimezoneLabel } from "@/lib/onboarding-form";
import type { OnboardingFormValues } from "@/types/onboarding";
import { useLocale, useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useMemo, useRef, useState } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";

const PICK_DELAY_MS = 280;
const CLOCK_REFRESH_MS = 30_000;

function localTime(locale: string, zone: string, now: Date): string {
  try {
    return new Intl.DateTimeFormat(locale, {
      hour: "numeric",
      minute: "2-digit",
      timeZone: zone,
    }).format(now);
  } catch {
    return "";
  }
}

function useNow(): Date {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), CLOCK_REFRESH_MS);
    return () => window.clearInterval(id);
  }, []);
  return now;
}

/**
 * The timezone turn: the browser's zone first and the US zones after it, each showing
 * its local time now. A pick is held on screen for a beat before it is sent, so the
 * selection is seen; "Other timezone" opens the full grouped list for anywhere else.
 */
export function TimezoneAsk({
  detected,
  onSubmit,
}: {
  detected: string;
  onSubmit: () => Promise<boolean>;
}) {
  const t = useT();
  const locale = useLocale();
  const now = useNow();
  const { control, setValue, getFieldState } = useFormContext<OnboardingFormValues>();
  const formState = useFormState({ control, name: "organization.timezone" });
  const error = getFieldState("organization.timezone", formState).error?.message;
  const value = useWatch({ control, name: "organization.timezone" });
  const tiles = useMemo(() => onboardingTimezoneTiles(detected), [detected]);
  const listed = tiles.includes(value);
  const [showOther, setShowOther] = useState(() => value !== "" && !listed);
  const [picked, setPicked] = useState<string | null>(null);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (timer.current !== null) {
        window.clearTimeout(timer.current);
      }
    };
  }, []);

  const pick = (zone: string) => {
    if (picked !== null || !zone) {
      return;
    }
    setPicked(zone);
    setValue("organization.timezone", zone, { shouldDirty: true });
    timer.current = window.setTimeout(() => {
      timer.current = null;
      void onSubmit().then((advanced) => {
        if (!advanced) {
          setPicked(null);
        }
      });
    }, PICK_DELAY_MS);
  };
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const index = numberKeyIndex(event, tiles.length);
      if (index !== -1) {
        event.preventDefault();
        pick(tiles[index]);
        return;
      }
      if (
        event.key === "Enter" &&
        !isTypingTarget(event.target) &&
        !(event.target instanceof HTMLButtonElement)
      ) {
        event.preventDefault();
        pick(value);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const selected = picked ?? value;

  return (
    <div>
      {error ? (
        <p className="nv-err" role="alert" style={{ marginBottom: 10 }}>
          {error}
        </p>
      ) : null}
      <div className="nv-tzs" role="group" aria-label={t("Timezone")}>
        {tiles.map((zone, index) => (
          <button
            key={zone}
            type="button"
            className="nv-tile"
            aria-pressed={selected === zone}
            style={{ animationDelay: `${index * 40}ms` }}
            onClick={() => pick(zone)}
          >
            <b>{t(onboardingTimezoneLabel(zone))}</b>
            <span>{localTime(locale, zone, now)}</span>
          </button>
        ))}
      </div>
      {showOther ? (
        <div className="nv-fc" style={{ marginTop: 10 }}>
          <SelectField
            control={control}
            name="organization.timezone"
            label={t("Other timezone")}
            placeholder={t("Select timezone")}
            groups={timezoneGroupedChoices}
            onValueChange={(zone) => pick(zone)}
          />
        </div>
      ) : (
        <div className="nv-other">
          <button type="button" onClick={() => setShowOther(true)}>
            {t("Other timezone…")}
          </button>
        </div>
      )}
    </div>
  );
}
