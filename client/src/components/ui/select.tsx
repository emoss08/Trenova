import * as React from "react";
import * as SelectPrimitive from "@radix-ui/react-select";
import { AlertTriangle, Check, ChevronDown, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { CaretSortIcon, CheckIcon } from "@radix-ui/react-icons";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Label } from "@radix-ui/react-dropdown-menu";

const Select = SelectPrimitive.Root;

const SelectGroup = SelectPrimitive.Group;

const SelectValue = SelectPrimitive.Value;

const SelectTrigger = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Trigger>
>(({ className, children, ...props }, ref) => (
  <SelectPrimitive.Trigger
    ref={ref}
    className={cn(
      "flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
      className,
    )}
    {...props}
  >
    {children}
    <SelectPrimitive.Icon asChild>
      <ChevronDown className="h-4 w-4 opacity-50" />
    </SelectPrimitive.Icon>
  </SelectPrimitive.Trigger>
));
SelectTrigger.displayName = SelectPrimitive.Trigger.displayName;

const SelectContent = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Content>
>(({ className, children, position = "popper", ...props }, ref) => (
  <SelectPrimitive.Portal>
    <SelectPrimitive.Content
      ref={ref}
      className={cn(
        "relative z-50 min-w-[8rem] overflow-hidden rounded-md border bg-popover text-popover-foreground shadow-md data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2",
        position === "popper" &&
          "data-[side=bottom]:translate-y-1 data-[side=left]:-translate-x-1 data-[side=right]:translate-x-1 data-[side=top]:-translate-y-1",
        className,
      )}
      position={position}
      {...props}
    >
      <SelectPrimitive.Viewport
        className={cn(
          "p-1",
          position === "popper" &&
            "h-[var(--radix-select-trigger-height)] w-full min-w-[var(--radix-select-trigger-width)]",
        )}
      >
        {children}
      </SelectPrimitive.Viewport>
    </SelectPrimitive.Content>
  </SelectPrimitive.Portal>
));
SelectContent.displayName = SelectPrimitive.Content.displayName;

const SelectLabel = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Label>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Label>
>(({ className, ...props }, ref) => (
  <SelectPrimitive.Label
    ref={ref}
    className={cn("py-1.5 pl-8 pr-2 text-sm font-semibold", className)}
    {...props}
  />
));
SelectLabel.displayName = SelectPrimitive.Label.displayName;

const SelectItem = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Item>
>(({ className, children, ...props }, ref) => (
  <SelectPrimitive.Item
    ref={ref}
    className={cn(
      "relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
      className,
    )}
    {...props}
  >
    <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
      <SelectPrimitive.ItemIndicator>
        <Check className="h-4 w-4" />
      </SelectPrimitive.ItemIndicator>
    </span>

    <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
  </SelectPrimitive.Item>
));
SelectItem.displayName = SelectPrimitive.Item.displayName;

const SelectSeparator = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Separator>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Separator>
>(({ className, ...props }, ref) => (
  <SelectPrimitive.Separator
    ref={ref}
    className={cn("-mx-1 my-1 h-px bg-muted", className)}
    {...props}
  />
));
SelectSeparator.displayName = SelectPrimitive.Separator.displayName;

export {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
};

export function SelectInput({
  data,
  placeholder,
  className,
  description,
  isLoading,
  isError,
  label,
  withAsterisk,
  searchText,
  limit = 10, // Default to 10
  ...props
}: {
  data: { label: string; value: string }[];
  placeholder?: string;
  description?: string;
  className?: string;
  isLoading?: boolean;
  isError?: boolean;
  label: string;
  withAsterisk?: boolean;
  searchText?: string;
  limit?: number;
}) {
  const [open, setOpen] = React.useState(false);
  const [value, setValue] = React.useState("");
  const [searchTerm, setSearchTerm] = React.useState("");
  const [displayedData, setDisplayedData] = React.useState(
    data.slice(0, limit),
  );

  const isComponentError = isError || !data;

  const filteredData = React.useMemo(() => {
    if (!searchTerm) return data.slice(0, limit);
    return data.filter((item) =>
      item.label.toLowerCase().includes(searchTerm.toLowerCase()),
    );
  }, [data, searchTerm, limit]);

  React.useEffect(() => {
    setDisplayedData(filteredData);
  }, [filteredData]);

  return (
    <>
      {label && (
        <Label
          className={cn("text-sm font-medium", withAsterisk && "required")}
        >
          {label}
        </Label>
      )}
      <div className="w-full relative">
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              role="combobox"
              aria-expanded={open}
              className={cn(
                "justify-between disabled:opacity-100 truncate w-full",
                className,
                isComponentError &&
                  "ring-2 ring-inset ring-red-500 text-red-500 focus:ring-red-500",
              )}
              disabled={isLoading || isComponentError}
              {...props}
            >
              <span className="truncate">
                {isLoading
                  ? "Fetching Data..."
                  : isComponentError
                  ? "Unable to load data."
                  : value
                  ? displayedData?.find((item) => item.value === value)?.label
                  : placeholder || "Select Item..."}
              </span>
              {isLoading ? (
                <Loader2 className="ml-2 h-4 w-4 shrink-0 animate-spin" />
              ) : isComponentError ? (
                <AlertTriangle
                  size={20}
                  className="ml-2 h-4 w-4 shrink-0 text-red-500"
                />
              ) : (
                <CaretSortIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
              )}
            </Button>
          </PopoverTrigger>
          <p className="text-xs text-foreground/70">{description}</p>
          <PopoverContent className="w-fit p-0">
            <Command>
              <CommandInput
                placeholder={searchText || "Search..."}
                className="h-9"
                value={searchTerm}
                onValueChange={(value) => setSearchTerm(value)}
              />
              {filteredData.length === 0 ? (
                <CommandEmpty>No item found.</CommandEmpty>
              ) : (
                <CommandGroup>
                  {filteredData.map((item) => (
                    <CommandItem
                      key={item.value}
                      onSelect={() => {
                        setValue(item.value === value ? "" : item.value);
                        setOpen(false);
                        setSearchTerm(""); // Clear search term after selecting
                      }}
                    >
                      {item.label}
                      <CheckIcon
                        className={cn(
                          "ml-auto h-4 w-4",
                          value === item.value ? "opacity-100" : "opacity-0",
                        )}
                      />
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
            </Command>
          </PopoverContent>
        </Popover>
      </div>
    </>
  );
}

export function ComboboxDemo({
  frameworks,
}: {
  frameworks: { label: string; value: string }[];
}) {
  const [value, setValue] = React.useState("");
  const [filteredOptions, setFilteredOptions] = React.useState(frameworks);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const query = event.target.value.toLowerCase();
    const matchedFrameworks = frameworks.filter((framework) =>
      framework.label.toLowerCase().includes(query),
    );
    setFilteredOptions(matchedFrameworks);
    setValue(event.target.value);
  };

  return (
    <div className="w-[200px] relative">
      <input
        type="text"
        value={value}
        onChange={handleChange}
        placeholder="Search framework..."
        className="h-9 px-2 w-full border border-gray-300 rounded"
      />

      {filteredOptions.length > 0 && (
        <div className="absolute z-10 w-full mt-2 border border-gray-300 bg-white shadow-md rounded">
          {filteredOptions.map((framework) => (
            <div
              key={framework.value}
              onClick={() => setValue(framework.label)}
              className="cursor-pointer px-2 py-1 hover:bg-gray-200"
            >
              {framework.label}
              {value === framework.label && (
                <CheckIcon className="ml-auto h-4 w-4 opacity-100" />
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
