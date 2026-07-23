"use client";

import type { Transition, Variants } from "motion/react";
import { m, useAnimation } from "motion/react";
import type { HTMLAttributes } from "react";
import { forwardRef, useCallback, useImperativeHandle, useRef } from "react";

import { cn } from "@/lib/utils";

export interface ContainerIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

type ContainerIconProps = HTMLAttributes<HTMLDivElement>;

const DEFAULT_TRANSITION: Transition = {
  duration: 0.5,
  opacity: { duration: 0.2 },
};

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

const ContainerIcon = forwardRef<ContainerIconHandle, ContainerIconProps>(
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
          <path d="M22 7.7c0-.6-.4-1.2-.8-1.5l-6.3-3.9a1.72 1.72 0 0 0-1.7 0l-10.3 6c-.5.2-.9.8-.9 1.4v6.6c0 .5.4 1.2.8 1.5l6.3 3.9a1.72 1.72 0 0 0 1.7 0l10.3-6c.5-.3.9-1 .9-1.5Z" />
          <path d="M10 21.9V14L2.1 9.1" />
          <path d="m10 14 11.9-6.9" />
          <m.path
            d="M14 19.8v-8.1"
            variants={LINE_VARIANTS}
            transition={DEFAULT_TRANSITION}
            animate={controls}
          />
          <m.path
            d="M18 17.5V9.4"
            variants={LINE_VARIANTS}
            transition={{ ...DEFAULT_TRANSITION, delay: 0.1 }}
            animate={controls}
          />
        </svg>
      </div>
    );
  },
);

ContainerIcon.displayName = "ContainerIcon";

export { ContainerIcon };
