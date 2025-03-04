import * as chrono from "chrono-node";
import { memo, useEffect, useImperativeHandle, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { generateDateTime, generateDateTimeString } from "@/lib/date";
import { cn } from "@/lib/utils";
import type { DateTimePickerProps, Suggestion } from "@/types/fields";
import { CalendarIcon, Cross2Icon } from "@radix-ui/react-icons";
import { DateTimePickerPopover } from "./datetime-picker-popover";

const defaultSuggestions = [
  "Today",
  "Tomorrow",
  "Sunday",
  "Next Monday",
  "Next Tuesday",
  "Next Sunday",
];

function generateSuggestions(
  inputValue: string,
  suggestion: Suggestion | null,
): Suggestion[] {
  if (!inputValue.length) {
    return defaultSuggestions
      .map((suggestion) => ({
        date: generateDateTime(suggestion),
        inputString: suggestion,
      }))
      .filter((s): s is Suggestion => s.date !== null);
  }

  const filteredDefaultSuggestions = defaultSuggestions.filter((suggestion) =>
    suggestion.toLowerCase().includes(inputValue.toLowerCase()),
  );
  if (filteredDefaultSuggestions.length) {
    return filteredDefaultSuggestions
      .map((suggestion) => ({
        date: generateDateTime(suggestion),
        inputString: suggestion,
      }))
      .filter((s): s is Suggestion => s.date !== null);
  }

  return [suggestion].filter((suggestion) => suggestion !== null);
}

export const DateTimePicker = memo(function DateTimePicker({
  dateTime,
  setDateTime,
  isInvalid,
  placeholder,
  clearable,
  ref,
  ...props
}: DateTimePickerProps) {
  const [suggestion, setSuggestion] = useState<Suggestion | null>(null);
  const [inputValue, setInputValue] = useState("");
  const [isOpen, setIsOpen] = useState(false);
  const [isClosing, setClosing] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);

  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useImperativeHandle(ref, () => inputRef.current!);

  const suggestions = generateSuggestions(inputValue, suggestion);

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const value = e.target.value;
    setInputValue(value);

    if (value.length > 0) {
      setIsOpen(true);
    } else {
      setIsOpen(false);
    }

    setSelectedIndex(0);

    const result = chrono.parseDate(value);
    if (result) {
      setSuggestion({ date: result, inputString: value });
    } else {
      setSuggestion(null);
    }
  }

  // 🔑 Only react to external changes in `date`:
  useEffect(() => {
    if (dateTime) {
      // If an outside update provides a date, sync the input to it.
      const formatted = generateDateTimeString(dateTime);
      setInputValue(formatted);
    } else {
      // If outside set `date` to undefined, clear the input
      setInputValue("");
    }
  }, [dateTime]);

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prevIndex) =>
        prevIndex < suggestions.length - 1 ? prevIndex + 1 : prevIndex,
      );
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prevIndex) => (prevIndex > 0 ? prevIndex - 1 : 0));
    } else if (e.key === "Enter" && isOpen && suggestions.length > 0) {
      e.preventDefault();
      const dateStr = generateDateTimeString(suggestions[selectedIndex].date);
      setInputValue(dateStr);
      setDateTime(suggestions[selectedIndex].date);
      closeDropdown();
    } else if (e.key === "Escape" || e.key === "Tab") {
      closeDropdown();
    }
  }

  function closeDropdown() {
    setClosing(true);
    setSelectedIndex(0);
    setTimeout(() => {
      setIsOpen(false);
      setClosing(false);
    }, 200);
  }

  function handleClear() {
    setInputValue("");
    setDateTime(undefined);
    closeDropdown();
    // inputRef.current?.focus();
  }

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        closeDropdown();
      }
    }

    document.addEventListener("mousedown", handleClickOutside);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  return (
    <div className="relative">
      <div className="relative">
        <Input
          placeholder={placeholder || "Tomorrow"}
          {...props}
          ref={inputRef}
          isInvalid={isInvalid}
          type="text"
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={() => setIsOpen(true)}
          onClick={() => setIsOpen(true)}
        />
        {clearable && inputValue && (
          <TooltipProvider>
            <Tooltip delayDuration={0}>
              <TooltipTrigger asChild>
                <Button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleClear();
                  }}
                  type="button"
                  size="icon"
                  className="absolute right-8 top-1/2 size-5 -translate-y-1/2 rounded-sm bg-transparent text-muted-foreground hover:bg-foreground/10"
                >
                  <span className="sr-only">Clear date</span>
                  <Cross2Icon className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Clear date</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
        <DateTimePickerPopover
          onOpen={() => setSuggestion(null)}
          dateTime={dateTime}
          setDateTime={setDateTime}
          setInputValue={setInputValue}
        >
          <Button
            type="button"
            size="icon"
            className="absolute right-2 top-1/2 size-5 -translate-y-1/2 rounded-sm bg-transparent text-muted-foreground hover:bg-foreground/10 [&>svg]:size-4"
          >
            <span className="sr-only">Open normal date time picker</span>
            <CalendarIcon className="size-4" />
          </Button>
        </DateTimePickerPopover>
      </div>

      {isOpen && suggestions.length > 0 && (
        <div
          ref={dropdownRef}
          role="dialog"
          className={cn(
            "absolute z-10 mt-2 w-full rounded-md border bg-popover p-0 shadow-md transition-all animate-in fade-in-0 zoom-in-95 slide-in-from-top-2",
            isClosing && "duration-300 animate-out fade-out-0 zoom-out-95",
          )}
          tabIndex={-1}
          aria-label="Suggestions"
        >
          <ul
            role="listbox"
            aria-label="Suggestions"
            className="max-h-56 overflow-auto p-1"
          >
            {suggestions.map((suggestion, index) => (
              <li
                key={suggestion.inputString}
                role="option"
                aria-selected={selectedIndex === index}
                className={cn(
                  "flex cursor-pointer items-center justify-between gap-1 rounded-sm px-3 py-1.5 text-xs",
                  index === selectedIndex && "bg-accent text-accent-foreground",
                )}
                onClick={() => {
                  const dateStr = generateDateTimeString(suggestion.date);
                  setInputValue(dateStr);
                  setDateTime(suggestion.date);
                  closeDropdown();
                  inputRef.current?.focus();
                }}
                onMouseEnter={() => setSelectedIndex(index)}
              >
                <span className="xs:w-auto w-[110px] truncate">
                  {suggestion.inputString}
                </span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {generateDateTimeString(suggestion.date)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
});
