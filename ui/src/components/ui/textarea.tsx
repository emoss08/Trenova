import { useAutoResizeTextarea } from "@/hooks/use-auto-resize-textarea";
import { cn } from "@/lib/utils";
import {
  faArrowDown,
  faCheck,
  faText,
} from "@fortawesome/pro-regular-svg-icons";
import * as React from "react";

import { useState } from "react";
import { Icon } from "./icons";

export type TextareaProps = React.ComponentProps<"textarea"> & {
  isInvalid?: boolean;
};

function Textarea({ className, isInvalid, rows = 3, ...props }: TextareaProps) {
  return (
    <textarea
      data-slot="textarea"
      rows={rows}
      className={cn(
        "flex w-full rounded-md border border-muted-foreground/20 bg-muted px-2 py-1 text-base",
        "shadow-xs placeholder:text-muted-foreground focus-visible:outline-hidden",
        "focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-xs",
        "focus-visible:border-blue-600 focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-blue-600/20",
        "transition-[border-color,box-shadow] duration-200 ease-in-out",
        isInvalid &&
          "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
        className,
      )}
      {...props}
    />
  );
}

export { Textarea };

function AutoResizeTextarea({
  className,
  isInvalid,
  onChange,
  ...props
}: TextareaProps) {
  const { textareaRef, adjustHeight } = useAutoResizeTextarea({
    minHeight: 70,
    maxHeight: 200,
  });

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    adjustHeight();
    onChange?.(e);
  };

  return (
    <Textarea
      ref={textareaRef}
      className={cn("resize-none min-h-[70px] max-h-[200px]", className)}
      isInvalid={isInvalid}
      onChange={handleChange}
      {...props}
    />
  );
}

export { AutoResizeTextarea };

const ITEMS = [
  {
    text: "Summary",
    icon: faText,
    colors: {
      icon: "text-orange-600",
      border: "border-orange-500",
      bg: "bg-orange-500/10",
    },
  },
  {
    text: "Fix Spelling and Grammar",
    icon: faCheck,
    colors: {
      icon: "text-emerald-600",
      border: "border-emerald-500",
      bg: "bg-emerald-500/10",
    },
  },
  {
    text: "Make shorter",
    icon: faArrowDown,
    colors: {
      icon: "text-purple-600",
      border: "border-purple-500",
      bg: "bg-purple-500/10",
    },
  },
];

function AITextarea({
  className,
  isInvalid,
  onChange,
  ...props
}: TextareaProps) {
  const [inputValue, setInputValue] = useState("");
  const [selectedItem, setSelectedItem] = useState<string | null>(
    "Make shorter",
  );
  const { textareaRef, adjustHeight } = useAutoResizeTextarea({
    minHeight: 70,
    maxHeight: 200,
  });

  const { id, ...rest } = props;

  const toggleItem = (itemText: string) => {
    setSelectedItem((prev) => (prev === itemText ? null : itemText));
  };

  const currentItem = selectedItem
    ? ITEMS.find((item) => item.text === selectedItem)
    : null;

  const handleSubmit = () => {
    setInputValue("");
    setSelectedItem(null);
    adjustHeight(true);
  };

  return (
    <>
      <div className="relative max-w-xl w-full mx-auto">
        <div
          className={cn(
            "relative border border-muted-foreground/20 rounded-md bg-muted",
            "focus-within:border-blue-600 focus-within:outline-hidden focus-within:ring-4 focus-within:ring-blue-600/20",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            isInvalid &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
            className,
          )}
        >
          <div className="flex flex-col">
            <div className="overflow-y-auto max-h-[200px]">
              <Textarea
                ref={textareaRef}
                id={id}
                className={cn(
                  "max-w-xl w-full rounded-md pr-10 pt-3 pb-3 placeholder:text-black/70 dark:placeholder:text-white/70 border-none focus:ring-3",
                  "text-black dark:text-white resize-none text-wrap bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0",
                  "min-h-[70px]",
                  "max-h-[200px]",
                )}
                value={inputValue}
                onChange={(e) => {
                  setInputValue(e.target.value);
                  adjustHeight();
                  onChange?.(e);
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    handleSubmit();
                  }
                }}
                {...rest}
              />
            </div>
            <div className="h-12 bg-transparent">
              {currentItem && (
                <div className="absolute left-3 bottom-3 z-10">
                  <button
                    type="button"
                    onClick={handleSubmit}
                    className={cn(
                      "inline-flex items-center gap-1.5",
                      "border shadow-xs rounded-md px-2 py-0.5 text-xs font-medium",
                      "animate-fadeIn hover:bg-black/5 dark:hover:bg-white/5 transition-colors duration-200",
                      currentItem.colors.bg,
                      currentItem.colors.border,
                    )}
                  >
                    <Icon
                      icon={currentItem.icon}
                      className={`size-3.5 ${currentItem.colors.icon}`}
                    />
                    <span className={currentItem.colors.icon}>
                      {selectedItem}
                    </span>
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
      <div className="flex flex-wrap gap-1.5 mt-2 max-w-xl mx-auto justify-start px-1">
        {ITEMS.filter((item) => item.text !== selectedItem).map(
          ({ text, icon, colors }) => (
            <button
              type="button"
              key={text}
              className={cn(
                "p-1 text-2xs font-medium rounded-md",
                "border transition-all duration-200 cursor-pointer",
                "shrink-0",
              )}
              onClick={() => toggleItem(text)}
            >
              <div className="flex items-center gap-1.5">
                <Icon icon={icon} className={cn("size-3", colors.icon)} />
                <span className="text-black/70 dark:text-white/70 whitespace-nowrap">
                  {text}
                </span>
              </div>
            </button>
          ),
        )}
      </div>
    </>
  );
}

export { AITextarea };

