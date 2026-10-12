import { useMotionPreferenceStore } from "@/stores/motion-preference-store";
import { MotionConfig } from "motion/react";
import { useEffect, type ReactNode } from "react";

/**
 * Applies the person's reduce-motion choice everywhere: CSS animations and transitions
 * through the root attribute, and every motion component through MotionConfig.
 */
export function MotionPreferenceProvider({ children }: { children: ReactNode }) {
  const reduceMotion = useMotionPreferenceStore((state) => state.reduceMotion);

  useEffect(() => {
    const root = document.documentElement;
    if (reduceMotion) {
      root.dataset.reduceMotion = "true";
    } else {
      delete root.dataset.reduceMotion;
    }
  }, [reduceMotion]);

  return <MotionConfig reducedMotion={reduceMotion ? "always" : "user"}>{children}</MotionConfig>;
}
