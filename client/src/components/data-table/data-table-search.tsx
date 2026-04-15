"use no memo";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { HelpCircleIcon, SearchIcon } from "lucide-react";
import { useEffect, useRef } from "react";

type DataTableSearchProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
};

const SEARCH_SYNTAX_EXAMPLES = [
  {
    syntax: "word",
    description: "Match word",
    example: "invoice",
  },
  {
    syntax: '"phrase"',
    description: "Exact phrase match",
    example: '"pending approval"',
  },
  {
    syntax: "word1 OR word2",
    description: "Match either word",
    example: "active OR pending",
  },
  {
    syntax: "-word",
    description: "Exclude word",
    example: "invoice -draft",
  },
  {
    syntax: "word1 word2",
    description: "Match both words (AND)",
    example: "customer order",
  },
];

export default function DataTableSearch({
  value,
  onChange,
  placeholder,
}: DataTableSearchProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (inputRef.current && inputRef.current.value !== value) {
      inputRef.current.value = value;
    }
  }, [value]);

  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const handleChange = (nextValue: string) => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    debounceTimerRef.current = setTimeout(() => {
      onChange(nextValue);
    }, 300);
  };

  return (
    <Input
      ref={inputRef}
      type="text"
      defaultValue={value}
      onChange={(e) => handleChange(e.target.value)}
      placeholder={placeholder ?? "Search..."}
      className="h-7 w-48 text-sm"
      leftElement={
        <SearchIcon className="size-3.5 shrink-0 text-muted-foreground" />
      }
      rightElement={<SearchSyntaxHelper />}
    />
  );
}

function SearchSyntaxHelper() {
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className="size-6 cursor-help text-muted-foreground hover:text-foreground"
          >
            <HelpCircleIcon className="size-3.5" />
          </Button>
        }
      />
      <PopoverContent className="w-80" align="end">
        <div className="space-y-3">
          <div>
            <h4 className="font-medium">Search Syntax</h4>
            <p className="text-sm text-muted-foreground">
              Use these patterns to refine your search results.
            </p>
          </div>
          <div className="space-y-2">
            {SEARCH_SYNTAX_EXAMPLES.map((item) => (
              <div
                key={item.syntax}
                className="grid grid-cols-[100px_1fr] gap-2 text-sm"
              >
                <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                  {item.syntax}
                </code>
                <div>
                  <p className="text-foreground">{item.description}</p>
                  <p className="text-xs text-muted-foreground">
                    e.g., <code className="font-mono">{item.example}</code>
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
