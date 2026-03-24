import { useAutoResizeTextarea } from "@/hooks/use-auto-resize-textarea";
import { cn } from "@/lib/utils";
import { ArrowDownIcon, CheckIcon, TextIcon } from "lucide-react";
import * as React from "react";
import { useState } from "react";
import TextareaAutosizeComponent from "react-textarea-autosize";

export type TextareaProps = React.ComponentProps<
  typeof TextareaAutosizeComponent
> & {
  isInvalid?: boolean;
};

function Textarea({ className, isInvalid, ...props }: TextareaProps) {
  return (
    <TextareaAutosizeComponent
      data-slot="textarea"
      className={cn(
        "flex w-full rounded-md border border-input bg-muted px-2 py-1 text-base",
        "shadow-xs placeholder:text-muted-foreground focus-visible:outline-hidden",
        "focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-xs",
        "focus-visible:border-brand focus-visible:ring-4 focus-visible:ring-brand/20 focus-visible:outline-hidden",
        "transition-[border-color,box-shadow] duration-200 ease-in-out",
        isInvalid &&
          "border-destructive bg-destructive/20 ring-0 ring-destructive placeholder:text-destructive focus:outline-hidden focus-visible:border-destructive focus-visible:ring-4 focus-visible:ring-destructive/20",
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
      icon: "text-orange-600",
      border: "border-orange-500",
      bg: "bg-orange-500/10",
    },
  },
  {
    text: "Fix Spelling and Grammar",
    icon: <CheckIcon />,
    colors: {
      icon: "text-emerald-600",
      border: "border-emerald-500",
      bg: "bg-emerald-500/10",
    },
  },
  {
    text: "Make shorter",
    icon: <ArrowDownIcon />,
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
      <div className="relative mx-auto w-full max-w-xl">
        <div
          className={cn(
            "relative rounded-md border border-muted-foreground/20 bg-muted",
            "focus-within:border-foreground focus-within:ring-4 focus-within:ring-foreground/20 focus-within:outline-hidden",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            isInvalid &&
              "border-destructive bg-destructive/20 ring-0 ring-destructive placeholder:text-destructive focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
            className,
          )}
        >
          <div className="flex flex-col">
            <div className="max-h-[200px] overflow-y-auto">
              <Textarea
                ref={textareaRef}
                id={id}
                className={cn(
                  "w-full max-w-xl rounded-md border-none pt-3 pr-10 pb-3 placeholder:text-black/70 focus:ring-3 dark:placeholder:text-white/70",
                  "resize-none bg-transparent text-wrap text-black focus-visible:ring-0 focus-visible:ring-offset-0 dark:text-white",
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
                      "rounded-md border px-2 py-0.5 text-xs font-medium shadow-xs",
                      "animate-fadeIn transition-colors duration-200 hover:bg-black/5 dark:hover:bg-white/5",
                      currentItem.colors.bg,
                      currentItem.colors.border,
                    )}
                  >
                    {currentItem.icon}
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
      <div className="mx-auto mt-2 flex max-w-xl flex-wrap justify-start gap-1.5 px-1">
        {ITEMS.filter((item) => item.text !== selectedItem).map(
          ({ text, icon }) => (
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
                <span className="whitespace-nowrap text-black/70 dark:text-white/70">
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
export { AITextarea, Textarea };
