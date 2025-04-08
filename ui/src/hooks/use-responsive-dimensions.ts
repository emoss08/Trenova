import { RefObject, useEffect, useMemo, useRef, useState } from "react";

export function useResponsiveDimensions(
  ref: RefObject<HTMLDivElement | null>,
  open: boolean,
) {
  const [dimensions, setDimensions] = useState({
    contentHeight: 0,
    viewportHeight: 0,
  });
  const isMountedRef = useRef(false);

  // Add debounce timer ref
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce function to prevent too many updates
  const debouncedSetDimensions = (newDimensions: typeof dimensions) => {
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
          return newDimensions;
        }
        return prev;
      });
    }, 100);
  };

  useEffect(() => {
    if (!open) return;

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
  }, [open, ref]);

  // Memoize the returned dimensions object to prevent unnecessary re-renders
  return useMemo(
    () => dimensions,
    [dimensions.contentHeight, dimensions.viewportHeight],
  );
}
