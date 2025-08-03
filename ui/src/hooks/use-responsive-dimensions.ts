/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { RefObject, useEffect, useMemo, useRef, useState } from "react";

export interface Dimensions {
  contentHeight: number | null;
  viewportHeight: number | null;
  isReady: boolean;
}

export function useResponsiveDimensions(
  ref: RefObject<HTMLDivElement | null>,
  open: boolean,
): Dimensions {
  const [dimensions, setDimensions] = useState<Dimensions>({
    contentHeight: null,
    viewportHeight: null,
    isReady: false,
  });
  const isMountedRef = useRef(false);

  // Add debounce timer ref
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce function to prevent too many updates
  const debouncedSetDimensions = (
    newDimensions: Omit<Dimensions, "isReady">,
  ) => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    debounceTimerRef.current = setTimeout(() => {
      setDimensions((prev) => {
        // Only update if dimensions actually changed
        if (
          prev.contentHeight !== newDimensions.contentHeight ||
          prev.viewportHeight !== newDimensions.viewportHeight
        ) {
          return { ...newDimensions, isReady: true };
        }
        return prev;
      });
    }, 100);
  };

  useEffect(() => {
    if (!open) return;

    // Initialize with window height on first render to avoid layout shift
    if (!dimensions.isReady) {
      const viewportHeight = window.innerHeight;
      // Use a reasonable initial contentHeight based on viewport
      const estimatedContentHeight = viewportHeight * 0.8;
      setDimensions({
        contentHeight: estimatedContentHeight,
        viewportHeight,
        isReady: false,
      });
    }

    const updateDimensions = () => {
      if (ref.current) {
        const contentHeight = ref.current.getBoundingClientRect().height;
        const viewportHeight = window.innerHeight;
        if (contentHeight > 0 && viewportHeight > 0) {
          debouncedSetDimensions({ contentHeight, viewportHeight });
        }
      }
    };

    const initialTimer = setTimeout(() => {
      isMountedRef.current = true;
      updateDimensions();
    }, 100);

    const resizeObserver = new ResizeObserver(() => {
      if (isMountedRef.current) {
        updateDimensions();
      }
    });

    if (ref.current) {
      resizeObserver.observe(ref.current);
    }

    const handleResize = () => {
      if (isMountedRef.current) updateDimensions();
    };
    window.addEventListener("resize", handleResize);

    return () => {
      clearTimeout(initialTimer);
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      resizeObserver.disconnect();
      window.removeEventListener("resize", handleResize);
      isMountedRef.current = false;
    };
  }, [open, ref, dimensions.isReady]);

  // Memoize the returned dimensions object to prevent unnecessary re-renders
  return useMemo(
    () => dimensions,
    [dimensions.contentHeight, dimensions.viewportHeight, dimensions.isReady],
  );
}
