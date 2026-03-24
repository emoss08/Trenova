"use no memo";
import { useCallback, useEffect, useState } from "react";

const ANIMATION_DURATION = 200;

type UseAnimatedPanelOptions = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function useAnimatedPanel({
  open,
  onOpenChange,
}: UseAnimatedPanelOptions) {
  const [isVisible, setIsVisible] = useState(open);
  const [isAnimating, setIsAnimating] = useState(false);

  useEffect(() => {
    if (open) {
      setIsVisible(true);
      setIsAnimating(false);
    } else {
      setIsVisible(false);
    }
  }, [open]);

  const close = useCallback(() => {
    if (isAnimating) return;

    setIsAnimating(true);
    setIsVisible(false);

    setTimeout(() => {
      onOpenChange(false);
      setIsAnimating(false);
    }, ANIMATION_DURATION);
  }, [onOpenChange, isAnimating]);

  return {
    isVisible,
    isAnimating,
    close,
  };
}
