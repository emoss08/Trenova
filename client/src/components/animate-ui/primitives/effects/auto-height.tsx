"use client";

import {
  m,
  type HTMLMotionProps,
  type LegacyAnimationControls,
  type TargetAndTransition,
  type Transition,
} from "motion/react";
import * as React from "react";

import {
  Slot,
  type WithAsChild,
} from "@/components/animate-ui/primitives/animate/slot";

type AutoHeightProps = WithAsChild<
  {
    children: React.ReactNode;
    deps?: React.DependencyList;
    animate?: TargetAndTransition | LegacyAnimationControls;
    transition?: Transition;
  } & Omit<HTMLMotionProps<"div">, "animate">
>;

function AutoHeight({
  children,
  deps = [],
  transition = {
    type: "spring",
    stiffness: 300,
    damping: 30,
    bounce: 0,
    restDelta: 0.01,
  },
  style,
  animate,
  asChild = false,
  ...props
}: AutoHeightProps) {
  const Comp = asChild ? Slot : m.div;
  const layoutDependency = React.useMemo(() => JSON.stringify(deps), [deps]);

  return (
    <Comp
      style={{ overflow: "hidden", ...style }}
      layout="size"
      layoutDependency={layoutDependency}
      animate={animate}
      transition={transition}
      {...props}
    >
      <div>{children}</div>
    </Comp>
  );
}

export { AutoHeight, type AutoHeightProps };
