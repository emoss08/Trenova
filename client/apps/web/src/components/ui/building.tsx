"use client";

import type { Variants } from "motion/react";
import { m, useAnimation } from "motion/react";
import type { HTMLAttributes } from "react";
import { forwardRef, useCallback, useImperativeHandle, useRef } from "react";

import { cn } from "@/lib/utils";

export interface BuildingIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

type BuildingIconProps = HTMLAttributes<HTMLDivElement>;

const LINE_VARIANTS: Variants = {
  normal: {
    pathLength: 1,
    opacity: 1,
  },
  animate: {
    opacity: [0, 1],
    pathLength: [0, 1],
  },
};

const BuildingIcon = forwardRef<BuildingIconHandle, BuildingIconProps>(
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
          <path d="M6 21V5a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v16" />
          <path d="M6 10H4a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-2" />
          <m.path
            d="M10 8h4"
            variants={LINE_VARIANTS}
            transition={{
              duration: 0.2,
              opacity: { duration: 0.1 },
            }}
            animate={controls}
          />
          <m.path
            d="M10 12h4"
            variants={LINE_VARIANTS}
            transition={{
              duration: 0.2,
              delay: 0.15,
              opacity: { duration: 0.1, delay: 0.15 },
            }}
            animate={controls}
          />
          <m.path
            d="M14 21v-3a2 2 0 0 0-4 0v3"
            variants={LINE_VARIANTS}
            transition={{
              duration: 0.3,
              delay: 0.3,
              opacity: { duration: 0.1, delay: 0.3 },
            }}
            animate={controls}
          />
        </svg>
      </div>
    );
  },
);

BuildingIcon.displayName = "BuildingIcon";

export { BuildingIcon };
