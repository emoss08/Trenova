import { useAutoResizeTextarea } from "@trenova/shared/hooks/use-auto-resize-textarea";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowDownIcon, CheckIcon, TextIcon } from "lucide-react";
import * as React from "react";
import { useState } from "react";
import TextareaAutosizeComponent from "react-textarea-autosize";

export type TextareaProps = React.ComponentProps<typeof TextareaAutosizeComponent> & {
  isInvalid?: boolean;
};

function Textarea({ className, isInvalid, ...props }: TextareaProps) {
  return (
    <TextareaAutosizeComponent
      data-slot="textarea"
      className={cn(
        "ui-field flex w-full px-2 py-0.5 text-base",
        "ui-focus-ring placeholder:text-muted-foreground",
        "disabled:cursor-not-allowed disabled:opacity-60 md:text-xs",
        isInvalid &&
          "border-danger bg-danger/10 placeholder:text-danger-foreground [--ring:var(--ring-danger)]",
        className,
      )}
      {...props}
    />
  );
}

const ITEMS = [
  {
    text: "Summary",
    icon: <TextIcon />,
    colors: {
      icon: "text-warning-foreground",
      border: "border-warning",
      bg: "bg-warning/10",
    },
  },
  {
    text: "Fix Spelling and Grammar",
    icon: <CheckIcon />,
    colors: {
      icon: "text-success-foreground",
      border: "border-success",
      bg: "bg-success/10",
    },
  },
  {
    text: "Make shorter",
    icon: <ArrowDownIcon />,
    colors: {
      icon: "text-accent-violet-on-subtle",
      border: "border-accent-violet",
      bg: "bg-accent-violet/10",
    },
  },
];

function AITextarea({ className, isInvalid, onChange, ...props }: TextareaProps) {
  const [inputValue, setInputValue] = useState("");
  const [selectedItem, setSelectedItem] = useState<string | null>("Make shorter");
  const { textareaRef, adjustHeight } = useAutoResizeTextarea({
    minHeight: 70,
    maxHeight: 200,
  });

  const { id, ...rest } = props;

  const toggleItem = (itemText: string) => {
    setSelectedItem((prev) => (prev === itemText ? null : itemText));
  };

  const currentItem = selectedItem ? ITEMS.find((item) => item.text === selectedItem) : null;

  const handleSubmit = () => {
    setInputValue("");
    setSelectedItem(null);
    adjustHeight(true);
  };

  return (
    <>
      <div className="relative mx-auto w-full max-w-xl">
        <div
          className={cn(
            "relative rounded-md border border-muted-foreground/20 bg-muted",
            "ui-container-focus-ring",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            isInvalid &&
              "border-danger bg-danger/10 placeholder:text-danger-foreground [--ring:var(--ring-danger)]",
            className,
          )}
        >
          <div className="flex flex-col">
            <div className="max-h-[200px] overflow-y-auto">
              <Textarea
                ref={textareaRef}
                id={id}
                className={cn(
                  "w-full max-w-xl rounded-md border-none pt-3 pr-10 pb-3 placeholder:text-foreground-subtle",
                  "resize-none bg-transparent text-wrap text-foreground",
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
                <div className="absolute bottom-3 left-3 z-10">
                  <button
                    type="button"
                    onClick={handleSubmit}
                    className={cn(
                      "inline-flex items-center gap-1.5",
                      "rounded-md border px-2 py-0.5 text-xs font-medium",
                      "animate-fadeIn transition-colors duration-200 hover:bg-surface-hover",
                      currentItem.colors.bg,
                      currentItem.colors.border,
                    )}
                  >
                    {currentItem.icon}
                    <span className={currentItem.colors.icon}>{selectedItem}</span>
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
      <div className="mx-auto mt-2 flex max-w-xl flex-wrap justify-start gap-1.5 px-1">
        {ITEMS.filter((item) => item.text !== selectedItem).map(({ text, icon }) => (
          <button
            type="button"
            key={text}
            className={cn(
              "rounded-md p-1 text-2xs font-medium",
              "cursor-pointer border transition-all duration-200",
              "shrink-0",
            )}
            onClick={() => toggleItem(text)}
          >
            <div className="flex items-center gap-1.5">
              {icon}
              <span className="whitespace-nowrap text-foreground-muted">{text}</span>
            </div>
          </button>
        ))}
      </div>
    </>
  );
}
export { AITextarea, Textarea };
