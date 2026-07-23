"use client";

import { m, useAnimation } from "motion/react";
import type { HTMLAttributes } from "react";
import { forwardRef, useCallback, useImperativeHandle, useRef } from "react";

import { cn } from "@/lib/utils";

export interface TruckIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

interface TruckIconProps extends HTMLAttributes<HTMLDivElement> {}

const TruckIcon = forwardRef<TruckIconHandle, TruckIconProps>(
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
          <m.g
            variants={{
              normal: { x: 0 },
              animate: {
                x: [0, -1, 1, -1, 0.5, -0.5, 0],
              },
            }}
            transition={{
              duration: 0.5,
              ease: "easeInOut",
            }}
            animate={controls}
          >
            <path d="M14 18V6a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2v11a1 1 0 0 0 1 1h2" />
            <path d="M15 18H9" />
            <path d="M19 18h2a1 1 0 0 0 1-1v-3.65a1 1 0 0 0-.22-.624l-3.48-4.35A1 1 0 0 0 17.52 8H14" />
            <g
              style={{
                transformBox: "fill-box",
                transformOrigin: "center",
              }}
            >
              <circle cx="7" cy="18" r="2" />
            </g>
            <g
              style={{
                transformBox: "fill-box",
                transformOrigin: "center",
              }}
            >
              <circle cx="17" cy="18" r="2" />
            </g>
          </m.g>
        </svg>
      </div>
    );
  },
);

TruckIcon.displayName = "TruckIcon";

export { TruckIcon };
