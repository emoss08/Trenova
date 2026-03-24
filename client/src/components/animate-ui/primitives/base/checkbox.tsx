"use client";

import { Checkbox as CheckboxPrimitive } from "@base-ui/react/checkbox";
import { m, type HTMLMotionProps, type SVGMotionProps } from "motion/react";
import * as React from "react";

import { useControlledState } from "@/hooks/use-controlled-state";
import { getStrictContext } from "@/lib/get-strict-context";

type CheckboxContextType = {
  isChecked: boolean;
  setIsChecked: CheckboxProps["onCheckedChange"];
  isIndeterminate: boolean | undefined;
};

const [CheckboxProvider, useCheckbox] =
  getStrictContext<CheckboxContextType>("CheckboxContext");

type CheckboxProps = Omit<
  React.ComponentProps<typeof CheckboxPrimitive.Root>,
  "render"
> &
  HTMLMotionProps<"button">;

function Checkbox({
  name,
  checked,
  defaultChecked,
  onCheckedChange,
  indeterminate,
  value,
  nativeButton,
  parent,
  disabled,
  readOnly,
  required,
  inputRef,
  id,
  ...props
}: CheckboxProps) {
  const [isChecked, setIsChecked] = useControlledState({
    value: checked,
    defaultValue: defaultChecked,
    onChange: onCheckedChange,
  });

  return (
    <CheckboxProvider
      value={{ isChecked, setIsChecked, isIndeterminate: indeterminate }}
    >
      <CheckboxPrimitive.Root
        name={name}
        defaultChecked={defaultChecked}
        checked={checked}
        onCheckedChange={setIsChecked}
        indeterminate={indeterminate}
        value={value}
        nativeButton={nativeButton}
        parent={parent}
        disabled={disabled}
        readOnly={readOnly}
        required={required}
        inputRef={inputRef}
        id={id}
        render={
          <m.button
            data-slot="checkbox"
            whileTap={{ scale: 0.95 }}
            whileHover={{ scale: 1.05 }}
            {...props}
          />
        }
      />
    </CheckboxProvider>
  );
}

type CheckboxIndicatorProps = SVGMotionProps<SVGSVGElement>;

function CheckboxIndicator(props: CheckboxIndicatorProps) {
  const { isChecked, isIndeterminate } = useCheckbox();

  return (
    <CheckboxPrimitive.Indicator
      keepMounted
      render={
        <m.svg
          data-slot="checkbox-indicator"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth="3.5"
          stroke="currentColor"
          initial="unchecked"
          animate={isChecked ? "checked" : "unchecked"}
          {...props}
        >
          {isIndeterminate ? (
            <m.line
              x1="5"
              y1="12"
              x2="19"
              y2="12"
              strokeLinecap="round"
              initial={{ pathLength: 0, opacity: 0 }}
              animate={{
                pathLength: 1,
                opacity: 1,
                transition: { duration: 0.2 },
              }}
            />
          ) : (
            <m.path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M4.5 12.75l6 6 9-13.5"
              variants={{
                checked: {
                  pathLength: 1,
                  opacity: 1,
                  transition: {
                    duration: 0.2,
                    delay: 0.2,
                  },
                },
                unchecked: {
                  pathLength: 0,
                  opacity: 0,
                  transition: {
                    duration: 0.2,
                  },
                },
              }}
            />
          )}
        </m.svg>
      }
    ></CheckboxPrimitive.Indicator>
  );
}

export {
  Checkbox,
  CheckboxIndicator,
  useCheckbox,
  type CheckboxContextType,
  type CheckboxIndicatorProps,
  type CheckboxProps,
};
