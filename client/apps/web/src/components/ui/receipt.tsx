"use client";

import type { Variants } from "motion/react";
import { m, useAnimation } from "motion/react";
import type { HTMLAttributes } from "react";
import { forwardRef, useCallback, useImperativeHandle, useRef } from "react";

import { cn } from "@/lib/utils";

export interface ReceiptIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

type ReceiptIconProps = HTMLAttributes<HTMLDivElement>;

const S_VARIANTS: Variants = {
  normal: {
    pathLength: 1,
    opacity: 1,
  },
  animate: {
    opacity: [0, 1],
    pathLength: [0, 1],
  },
};

const STRIKE_VARIANTS: Variants = {
  normal: {
    pathLength: 1,
    opacity: 1,
  },
  animate: {
    opacity: [0, 1],
    pathLength: [0, 1],
  },
};

const ReceiptIcon = forwardRef<ReceiptIconHandle, ReceiptIconProps>(
  ({ onMouseEnter, onMouseLeave, className, ...props }, ref) => {
    const controls = useAnimation();
    const isControlledRef = useRef(false);

    useImperativeHandle(ref, () => {
      isControlledRef.current = true;

      return {
        startAnimation: () => controls.start("animate"),
        stopAnimation: () => controls.start("normal"),
      };
    });

    const handleMouseEnter = useCallback(
      (e: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlledRef.current) {
          void controls.start("animate");
        } else {
          onMouseEnter?.(e);
        }
      },
      [controls, onMouseEnter],
    );

    const handleMouseLeave = useCallback(
      (e: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlledRef.current) {
          void controls.start("normal");
        } else {
          onMouseLeave?.(e);
        }
      },
      [controls, onMouseLeave],
    );

    return (
      <div
        className={cn(className)}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        {...props}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="100%"
          height="100%"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1Z" />
          <m.path
            d="M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8"
            variants={S_VARIANTS}
            transition={{
              duration: 0.4,
              opacity: { duration: 0.1 },
            }}
            animate={controls}
          />
          <m.path
            d="M12 17.5v-11"
            variants={STRIKE_VARIANTS}
            transition={{
              duration: 0.3,
              delay: 0.35,
              opacity: { duration: 0.1, delay: 0.35 },
            }}
            animate={controls}
          />
        </svg>
      </div>
    );
  },
);

ReceiptIcon.displayName = "ReceiptIcon";

export { ReceiptIcon };
