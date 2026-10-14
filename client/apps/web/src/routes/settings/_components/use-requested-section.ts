import {
  isSettingsSection,
  SETTINGS_SECTION_PARAM,
  type SettingsSection,
} from "@/lib/settings-path";
import { useEffect } from "react";
import { useSearchParams } from "react-router";

/**
 * Lands on the section the address names (`/settings?section=notifications`), which is how
 * the notification panel and an email's "Change what reaches you" link arrive, then takes the
 * parameter back out so a reload or a later visit starts at the top.
 */
export function useRequestedSection(jump: (section: SettingsSection) => void) {
  const [searchParams, setSearchParams] = useSearchParams();
  const requested = searchParams.get(SETTINGS_SECTION_PARAM);

  useEffect(() => {
    if (requested === null) return;
    const frame = window.requestAnimationFrame(() => {
      if (isSettingsSection(requested)) {
        jump(requested);
      }
      setSearchParams(
        (current) => {
          const next = new URLSearchParams(current);
          next.delete(SETTINGS_SECTION_PARAM);
          return next;
        },
        { replace: true },
      );
    });
    return () => window.cancelAnimationFrame(frame);
  }, [jump, requested, setSearchParams]);
}
