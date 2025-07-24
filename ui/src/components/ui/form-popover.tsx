/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { faFingerprint, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { AnimatePresence, motion } from "motion/react";
import { ReactNode, RefObject, useEffect, useRef } from "react";
import { Button } from "./button";
import { Icon } from "./icons";

type PopoverFormProps = {
  open: boolean;
  setOpen: (open: boolean) => void;
  openChild?: ReactNode;
  width?: string;
  height?: string;
  showCloseButton?: boolean;
  title: string;
  position?: "left" | "right";
};

export function PopoverForm({
  open,
  setOpen,
  openChild,
  width = "364px",
  height = "192px",
  title = "Feedback",
  showCloseButton = false,
  position = "right",
}: PopoverFormProps) {
  const ref = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  useClickOutside(containerRef, () => setOpen(false));

  return (
    <div
      key={title}
      className="flex h-full w-full items-center justify-center"
      ref={containerRef}
    >
      <motion.button
        layoutId={`${title}-wrapper`}
        onClick={() => setOpen(true)}
        style={{ borderRadius: 100 }}
        className="flex size-10 items-center justify-center text-sm text-white font-medium outline-none cursor-pointer bg-gradient-to-r from-blue-500 to-purple-500 hover:from-sky-500 hover:to-indigo-500 shadow-lg transition-colors duration-300"
      >
        <motion.span layoutId={`${title}-title`}>
          <Icon icon={faFingerprint} className="size-4" />
        </motion.span>
      </motion.button>
      <AnimatePresence>
        {open && (
          <motion.div
            layoutId={`${title}-wrapper`}
            className="fixed outline-none"
            ref={ref}
            style={{
              borderRadius: 10,
              width,
              height,
              [position]: "50px",
              bottom: "30px",
            }}
          >
            {showCloseButton && (
              <div className="absolute right-2 top-2 z-20">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setOpen(false)}
                  className="rounded-sm px-1.5 transition-[border-color,box-shadow] duration-100 ease-in-out focus:border focus:border-blue-600 focus:outline-hidden focus:ring-4 focus:ring-blue-600/20 disabled:pointer-events-none [&_svg]:size-4 "
                >
                  <Icon icon={faXmark} className="size-4" />
                  <span className="sr-only">Close</span>
                </Button>
              </div>
            )}

            <AnimatePresence mode="popLayout">
              <motion.div
                exit={{ y: 8, opacity: 0, filter: "blur(4px)" }}
                transition={{ type: "spring", duration: 0.4, bounce: 0 }}
                key="open-child"
                style={{ borderRadius: 10 }}
                className="z-20 h-full border bg-background"
              >
                {openChild}
              </motion.div>
            </AnimatePresence>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

const useClickOutside = (
  ref: RefObject<HTMLElement | null>,
  handleOnClickOutside: (event: MouseEvent | TouchEvent) => void,
) => {
  useEffect(() => {
    const listener = (event: MouseEvent | TouchEvent) => {
      if (!ref.current || ref.current.contains(event.target as Node)) {
        return;
      }
      handleOnClickOutside(event);
    };
    document.addEventListener("mousedown", listener);
    document.addEventListener("touchstart", listener);
    return () => {
      document.removeEventListener("mousedown", listener);
      document.removeEventListener("touchstart", listener);
    };
  }, [ref, handleOnClickOutside]);
};
