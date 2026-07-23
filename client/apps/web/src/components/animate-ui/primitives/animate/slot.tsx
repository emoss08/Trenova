import { cn } from "@/lib/utils";
import { isMotionComponent, m, type HTMLElements, type HTMLMotionProps } from "motion/react";
import * as React from "react";

type AnyProps = Record<string, unknown>;

type DOMMotionProps = Omit<HTMLMotionProps<keyof HTMLElements>, "ref"> & {
  ref?: React.Ref<HTMLElement>;
};

type WithAsChild<Base extends object> =
  | (Base & { asChild: true; children: React.ReactElement })
  | (Base & { asChild?: false | undefined });

type SlotProps = {
  children?: any;
} & DOMMotionProps;

const motionIntrinsic = {
  div: m.div,
  span: m.span,
  button: m.button,
  li: m.li,
  ul: m.ul,
  p: m.p,
} as const;

function mergeRefs<T>(...refs: (React.Ref<T> | undefined)[]): React.RefCallback<T> {
  return (node) => {
    refs.forEach((ref) => {
      if (!ref) return;
      if (typeof ref === "function") {
        ref(node);
      } else {
        (ref as React.RefObject<T | null>).current = node;
      }
    });
  };
}

function mergeProps(childProps: AnyProps, slotProps: DOMMotionProps): AnyProps {
  const merged: AnyProps = { ...childProps, ...slotProps };

  if (childProps.className || slotProps.className) {
    merged.className = cn(childProps.className as string, slotProps.className as string);
  }

  if (childProps.style || slotProps.style) {
    merged.style = {
      ...(childProps.style as React.CSSProperties),
      ...(slotProps.style as React.CSSProperties),
    };
  }

  return merged;
}

function Slot({ children, ref, ...props }: SlotProps) {
  if (!React.isValidElement(children)) return null;

  const isAlreadyMotion =
    typeof children.type === "object" && children.type !== null && isMotionComponent(children.type);

  const { ref: childRef, ...childProps } = children.props as AnyProps;

  const mergedProps = mergeProps(childProps, props);

  if (isAlreadyMotion) {
    const MotionBase = children.type as React.ElementType;
    return (
      <MotionBase
        {...mergedProps}
        ref={mergeRefs(childRef as React.Ref<HTMLElement>, ref) as React.Ref<any>}
      />
    );
  }

  if (typeof children.type === "string") {
    const tag = children.type as keyof typeof motionIntrinsic;
    const MotionBase = motionIntrinsic[tag] ?? m.div;
    return (
      <MotionBase
        {...mergedProps}
        ref={mergeRefs(childRef as React.Ref<HTMLElement>, ref) as React.Ref<any>}
      />
    );
  }

  return (
    <m.div {...(props as any)} ref={ref as React.Ref<HTMLDivElement>}>
      {React.cloneElement(children, childProps)}
    </m.div>
  );
}

export { Slot, type AnyProps, type DOMMotionProps, type SlotProps, type WithAsChild };
