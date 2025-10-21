import { cn } from "@/lib/utils";
import type { HTMLMotionProps, Variants } from "motion/react";
import { motion, useAnimation, useReducedMotion } from "motion/react";
import { forwardRef, useCallback, useImperativeHandle, useRef } from "react";

export interface CircleCheckBigIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

interface CircleCheckBigIconProps extends HTMLMotionProps<"div"> {
  size?: number;
  startAnimation?: boolean;
}

const CircleCheckBigIcon = forwardRef<
  CircleCheckBigIconHandle,
  CircleCheckBigIconProps
>(
  (
    {
      onMouseEnter,
      onMouseLeave,
      className,
      size = 28,
      startAnimation = false,
      ...props
    },
    ref,
  ) => {
    const controls = useAnimation();
    const tickControls = useAnimation();
    const reduced = useReducedMotion();
    const isControlled = useRef(false);

    useImperativeHandle(ref, () => {
      isControlled.current = true;
      return {
        startAnimation: () => {
          if (reduced) {
            controls.start("normal");
            tickControls.start("normal");
          } else {
            controls.start("animate");
            tickControls.start("animate");
          }
        },
        stopAnimation: () => {
          controls.start("normal");
          tickControls.start("normal");
        },
      };
    });

    const handleEnter = useCallback(
      (e?: React.MouseEvent<HTMLDivElement>) => {
        if (reduced) return;
        if (!isControlled.current) {
          if (startAnimation) {
            controls.start("animate");
            tickControls.start("animate");
          }
          tickControls.start("animate");
        } else {
          onMouseEnter?.(e as any);
        }
      },
      [controls, tickControls, reduced, onMouseEnter, startAnimation],
    );

    const handleLeave = useCallback(
      (e?: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlled.current) {
          if (startAnimation) {
            controls.start("normal");
            tickControls.start("normal");
          }
          tickControls.start("normal");
        } else {
          onMouseLeave?.(e as any);
        }
      },
      [controls, tickControls, onMouseLeave, startAnimation],
    );

    const svgVariants: Variants = {
      normal: { scale: 1 },
      animate: {
        scale: [1, 1.05, 0.98, 1],
        transition: {
          duration: 1,
          ease: [0.42, 0, 0.58, 1],
        },
      },
    };

    const circleVariants: Variants = {
      normal: { pathLength: 1, opacity: 1 },
      animate: { pathLength: 1, opacity: 1 },
    };

    const tickVariants: Variants = {
      normal: { pathLength: 1, opacity: 1 },
      animate: {
        pathLength: [0, 1],
        opacity: 1,
        transition: {
          duration: 0.8,
          ease: [0.42, 0, 0.58, 1],
        },
      },
    };

    return (
      <motion.div
        className={cn("inline-flex items-center justify-center", className)}
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
        {...props}
      >
        <motion.svg
          xmlns="http://www.w3.org/2000/svg"
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          animate={startAnimation ? controls : "normal"}
          initial="normal"
          variants={svgVariants}
        >
          <motion.path
            d="M21.801 10A10 10 0 1 1 17 3.335"
            variants={circleVariants}
            initial="normal"
          />
          <motion.path
            d="m9 11 3 3L22 4"
            animate={startAnimation ? tickControls : "normal"}
            initial="normal"
            variants={tickVariants}
          />
        </motion.svg>
      </motion.div>
    );
  },
);
CircleCheckBigIcon.displayName = "CircleCheckBigIcon";

export interface XIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

interface XIconProps extends HTMLMotionProps<"div"> {
  size?: number;
}

const XIcon = forwardRef<XIconHandle, XIconProps>(
  ({ onMouseEnter, onMouseLeave, className, size = 24, ...props }, ref) => {
    const svgControls = useAnimation();
    const path1Controls = useAnimation();
    const path2Controls = useAnimation();
    const reduced = useReducedMotion();
    const isControlled = useRef(false);

    useImperativeHandle(ref, () => {
      isControlled.current = true;
      return {
        startAnimation: () => {
          if (reduced) {
            svgControls.start("normal");
            path1Controls.start("normal");
            path2Controls.start("normal");
          } else {
            svgControls.start("animate");
            path1Controls.start("animate");
            path2Controls.start("animate");
          }
        },
        stopAnimation: () => {
          svgControls.start("normal");
          path1Controls.start("normal");
          path2Controls.start("normal");
        },
      };
    });

    const handleEnter = useCallback(
      (e?: React.MouseEvent<HTMLDivElement>) => {
        if (reduced) return;
        if (!isControlled.current) {
          svgControls.start("animate");
          path1Controls.start("animate");
          path2Controls.start("animate");
        } else {
          onMouseEnter?.(e as any);
        }
      },
      [svgControls, path1Controls, path2Controls, reduced, onMouseEnter],
    );

    const handleLeave = useCallback(
      (e: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlled.current) {
          svgControls.start("normal");
          path1Controls.start("normal");
          path2Controls.start("normal");
        } else {
          onMouseLeave?.(e);
        }
      },
      [svgControls, path1Controls, path2Controls, onMouseLeave],
    );

    const svgVariants: Variants = {
      normal: { rotate: 0, scale: 1, transition: { duration: 0.3 } },
      animate: {
        rotate: [0, 15, -15, 0],
        scale: [1, 1.1, 1],
        transition: { duration: 0.6 },
      },
    };

    const pathVariants: Variants = {
      normal: { pathLength: 1, opacity: 1 },
      animate: {
        pathLength: [0, 1],
        opacity: [0, 1],
        transition: { duration: 0.6, ease: "easeInOut" },
      },
    };

    return (
      <motion.div
        className={cn("inline-flex items-center justify-center", className)}
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
        {...props}
      >
        <motion.svg
          xmlns="http://www.w3.org/2000/svg"
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          variants={svgVariants}
          initial="normal"
          animate={svgControls}
        >
          <motion.path
            d="M18 6 6 18"
            variants={pathVariants}
            initial="normal"
            animate={path1Controls}
          />
          <motion.path
            d="m6 6 12 12"
            variants={pathVariants}
            initial="normal"
            animate={path2Controls}
            transition={{ delay: 0.2 }}
          />
        </motion.svg>
      </motion.div>
    );
  },
);

XIcon.displayName = "XIcon";

export interface ChevronDownIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

interface ChevronDownIconProps extends HTMLMotionProps<"div"> {
  size?: number;
}

const ChevronDownIcon = forwardRef<ChevronDownIconHandle, ChevronDownIconProps>(
  ({ onMouseEnter, onMouseLeave, className, size = 28, ...props }, ref) => {
    const controls = useAnimation();
    const reduced = useReducedMotion();
    const isControlled = useRef(false);

    useImperativeHandle(ref, () => {
      isControlled.current = true;
      return {
        startAnimation: () =>
          reduced ? controls.start("normal") : controls.start("animate"),
        stopAnimation: () => controls.start("normal"),
      };
    });

    const handleEnter = useCallback(
      (e?: React.MouseEvent<HTMLDivElement>) => {
        if (reduced) return;
        if (!isControlled.current) controls.start("animate");
        else onMouseEnter?.(e as any);
      },
      [controls, reduced, onMouseEnter],
    );

    const handleLeave = useCallback(
      (e?: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlled.current) controls.start("normal");
        else onMouseLeave?.(e as any);
      },
      [controls, onMouseLeave],
    );

    const leadingArrow: Variants = {
      normal: { y: 0, opacity: 1 },
      animate: {
        y: [0, 4, 0],
        opacity: [1, 0.6, 1],
        transition: {
          duration: 0.8,
          repeat: 0,
        },
      },
    };

    const trailingArrow: Variants = {
      normal: { y: 0, opacity: 0.5 },
      animate: {
        y: [0, 6, 0],
        opacity: [0.5, 0.2, 0.5],
        transition: {
          duration: 0.8,
          repeat: 0,
          delay: 0.2,
        },
      },
    };

    return (
      <motion.div
        className={cn("inline-flex items-center justify-center", className)}
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
        {...props}
      >
        <motion.svg
          xmlns="http://www.w3.org/2000/svg"
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          animate={controls}
          initial="normal"
        >
          <motion.path
            d="m6 9 6 6 6-6"
            variants={trailingArrow}
            stroke="currentColor"
          />
          <motion.path
            d="m6 9 6 6 6-6"
            variants={leadingArrow}
            stroke="currentColor"
          />
        </motion.svg>
      </motion.div>
    );
  },
);

ChevronDownIcon.displayName = "ChevronDownIcon";

export interface ChevronUpIconHandle {
  startAnimation: () => void;
  stopAnimation: () => void;
}

interface ChevronUpIconProps extends HTMLMotionProps<"div"> {
  size?: number;
}

const ChevronUpIcon = forwardRef<ChevronUpIconHandle, ChevronUpIconProps>(
  ({ onMouseEnter, onMouseLeave, className, size = 28, ...props }, ref) => {
    const controls = useAnimation();
    const reduced = useReducedMotion();
    const isControlled = useRef(false);

    useImperativeHandle(ref, () => {
      isControlled.current = true;
      return {
        startAnimation: () =>
          reduced ? controls.start("normal") : controls.start("animate"),
        stopAnimation: () => controls.start("normal"),
      };
    });

    const handleEnter = useCallback(
      (e?: React.MouseEvent<HTMLDivElement>) => {
        if (reduced) return;
        if (!isControlled.current) controls.start("animate");
        else onMouseEnter?.(e as any);
      },
      [controls, reduced, onMouseEnter],
    );

    const handleLeave = useCallback(
      (e?: React.MouseEvent<HTMLDivElement>) => {
        if (!isControlled.current) controls.start("normal");
        else onMouseLeave?.(e as any);
      },
      [controls, onMouseLeave],
    );

    const leadingArrow: Variants = {
      normal: { y: 0, opacity: 1 },
      animate: {
        y: [0, -4, 0],
        opacity: [1, 0.6, 1],
        transition: {
          duration: 0.8,
          repeat: 0,
        },
      },
    };

    const trailingArrow: Variants = {
      normal: { y: 0, opacity: 0.5 },
      animate: {
        y: [0, -6, 0],
        opacity: [0.5, 0.2, 0.5],
        transition: {
          duration: 0.8,
          repeat: 0,
          delay: 0.2,
        },
      },
    };

    return (
      <motion.div
        className={cn("inline-flex items-center justify-center", className)}
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
        {...props}
      >
        <motion.svg
          xmlns="http://www.w3.org/2000/svg"
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          animate={controls}
          initial="normal"
        >
          <motion.path
            d="m18 15-6-6-6 6"
            variants={trailingArrow}
            stroke="currentColor"
          />
          <motion.path
            d="m18 15-6-6-6 6"
            variants={leadingArrow}
            stroke="currentColor"
          />
        </motion.svg>
      </motion.div>
    );
  },
);

ChevronUpIcon.displayName = "ChevronUpIcon";

export { ChevronDownIcon, ChevronUpIcon, CircleCheckBigIcon, XIcon };
