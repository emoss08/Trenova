"use client";

import * as React from "react";

type AutoHeightOptions = {
  includeParentBox?: boolean;
  includeSelfBox?: boolean;
};

const DEFAULT_DEPS: React.DependencyList = [];

export function useAutoHeight<T extends HTMLElement = HTMLDivElement>(
  deps: React.DependencyList = DEFAULT_DEPS,
  options: AutoHeightOptions = {
    includeParentBox: true,
    includeSelfBox: false,
  },
) {
  const ref = React.useRef<T | null>(null);
  const roRef = React.useRef<ResizeObserver | null>(null);
  const [height, setHeight] = React.useState(0);
  const includeParentBox = options.includeParentBox ?? true;
  const includeSelfBox = options.includeSelfBox ?? false;

  const measure = React.useCallback(() => {
    const el = ref.current;
    if (!el) return 0;

    const base = el.getBoundingClientRect().height || 0;

    let extra = 0;

    if (includeParentBox && el.parentElement) {
      const cs = getComputedStyle(el.parentElement);
      const paddingY =
        (parseFloat(cs.paddingTop || "0") || 0) +
        (parseFloat(cs.paddingBottom || "0") || 0);
      const borderY =
        (parseFloat(cs.borderTopWidth || "0") || 0) +
        (parseFloat(cs.borderBottomWidth || "0") || 0);
      const isBorderBox = cs.boxSizing === "border-box";
      if (isBorderBox) {
        extra += paddingY + borderY;
      }
    }

    if (includeSelfBox) {
      const cs = getComputedStyle(el);
      const paddingY =
        (parseFloat(cs.paddingTop || "0") || 0) +
        (parseFloat(cs.paddingBottom || "0") || 0);
      const borderY =
        (parseFloat(cs.borderTopWidth || "0") || 0) +
        (parseFloat(cs.borderBottomWidth || "0") || 0);
      const isBorderBox = cs.boxSizing === "border-box";
      if (isBorderBox) {
        extra += paddingY + borderY;
      }
    }

    const dpr =
      typeof window !== "undefined" ? window.devicePixelRatio || 1 : 1;
    const total = Math.ceil((base + extra) * dpr) / dpr;

    return total;
  }, [includeParentBox, includeSelfBox]);

  React.useEffect(() => {
    const el = ref.current;
    if (!el) return;

    setHeight(measure());

    if (roRef.current) {
      roRef.current.disconnect();
      roRef.current = null;
    }

    const ro = new ResizeObserver(() => {
      const next = measure();
      requestAnimationFrame(() => setHeight(next));
    });

    ro.observe(el);
    if (includeParentBox && el.parentElement) {
      ro.observe(el.parentElement);
    }

    roRef.current = ro;

    return () => {
      ro.disconnect();
      roRef.current = null;
    };
  }, [measure, includeParentBox, deps]);

  React.useLayoutEffect(() => {
    if (height === 0) {
      const next = measure();
      if (next !== 0) setHeight(next);
    }
  }, [height, measure]);

  return { ref, height } as const;
}
