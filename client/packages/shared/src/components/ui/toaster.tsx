import { TOAST_SOUND_VOLUME, useToastSoundStore } from "@trenova/shared/components/ui/toast-sound";
import { useT } from "@trenova/shared/i18n/use-t";
import { Toaster as DingToaster, toast as ding, type Position } from "@yusufceng1z/ding-react";
import { useEffect, useRef, type RefObject } from "react";

export type ToastPosition = Position;

/**
 * ding names its region and its close button in English. Its markup never sets them again
 * after mount, so they are relabelled once each time a toast is drawn.
 */
function useTranslatedLabels(
  frame: RefObject<HTMLDivElement | null>,
  labels: { region: string; dismiss: string },
) {
  const { region, dismiss } = labels;
  useEffect(() => {
    const root = frame.current;
    if (!root) {
      return;
    }
    const apply = () => {
      root.querySelector(".tst-viewport")?.setAttribute("aria-label", region);
      for (const button of root.querySelectorAll(".tst-close")) {
        button.setAttribute("aria-label", dismiss);
      }
    };
    apply();
    const observer = new MutationObserver(apply);
    observer.observe(root, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, [frame, region, dismiss]);
}

/**
 * Where toasts appear: ding's, one at a time, morphing in place, with a cue per status
 * unless this person turned sounds off. Drawn dark in both themes, like the menus and
 * popovers that also float above the page; app.css maps ding's variables onto the tokens.
 * Mount one per app.
 */
export function Toaster({ position = "top-right" }: { position?: ToastPosition }) {
  const t = useT();
  const sound = useToastSoundStore((state) => state.on);
  const frame = useRef<HTMLDivElement>(null);

  useEffect(() => {
    ding.configure({ sound, volume: TOAST_SOUND_VOLUME });
  }, [sound]);

  useTranslatedLabels(frame, {
    region: t("Notifications"),
    dismiss: t("Dismiss notification"),
  });

  return (
    <div ref={frame} className="dark contents">
      <DingToaster position={position} theme="dark" />
    </div>
  );
}
