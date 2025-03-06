import { RefObject, useEffect, useRef, useState } from "react";

export function useResponsiveDimensions(
  ref: RefObject<HTMLDivElement | null>,
  open: boolean,
) {
  const [dimensions, setDimensions] = useState({
    contentHeight: 0,
    viewportHeight: 0,
  });
  const isMountedRef = useRef(false);

  useEffect(() => {
    if (!open) return;

    const updateDimensions = () => {
      if (ref.current) {
        const contentHeight = ref.current.getBoundingClientRect().height;
        const viewportHeight = window.innerHeight;
        if (contentHeight > 0 && viewportHeight > 0) {
          setDimensions({ contentHeight, viewportHeight });
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
      resizeObserver.disconnect();
      window.removeEventListener("resize", handleResize);
      isMountedRef.current = false;
    };
  }, [open, ref]);

  return dimensions;
}
