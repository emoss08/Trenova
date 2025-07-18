import * as SelectPrimitive from "@radix-ui/react-select";
import * as React from "react";

import { cn } from "@/lib/utils";
import type { SelectOption, SelectValue } from "@/types/fields";
import {
  CheckIcon,
  ChevronDownIcon,
  ChevronUpIcon,
} from "@radix-ui/react-icons";
import { Icon } from "./icons";

function Select({
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Root>) {
  return <SelectPrimitive.Root data-slot="select" {...props} />;
}

function SelectGroup({
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Group>) {
  return <SelectPrimitive.Group data-slot="select-group" {...props} />;
}

function SelectValue({ color, icon, ...props }: SelectValue) {
  const renderIcon = () => {
    if (typeof icon === "object" && icon !== null && "icon" in icon) {
      return <Icon icon={icon} />;
    }
    return icon;
  };

  return (
    <div className="flex h-5 items-center text-xs font-normal text-foreground gap-x-1.5 flex-1 min-w-0 overflow-hidden [&_svg]:size-3 [&_svg]:shrink-0">
      {color ? (
        <div
          className="size-2 rounded-full flex-shrink-0"
          style={{
            backgroundColor: color,
          }}
        />
      ) : icon ? (
        renderIcon()
      ) : null}
      <div className="truncate min-w-0 flex-1 text-left">
        <SelectPrimitive.Value data-slot="select-value" {...props} />
      </div>
    </div>
  );
}

function SelectTrigger({
  className,
  children,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Trigger>) {
  return (
    <SelectPrimitive.Trigger
      data-slot="select-trigger"
      className={cn(
        "group bg-muted flex h-7 w-full items-center justify-between whitespace-nowrap rounded-md border border-muted-foreground/20",
        "px-1.5 py-2 text-xs ring-offset-background placeholder:text-muted-foreground outline-hidden",
        "data-[state=open]:border-blue-600 data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-blue-600/20",
        "focus-visible:border-blue-600 focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-blue-600/20",
        "transition-[border-color,box-shadow] duration-200 ease-in-out",
        "disabled:opacity-50 [&>span]:line-clamp-1 cursor-pointer disabled:cursor-not-allowed",
        className,
      )}
      {...props}
    >
      {children}
      <SelectPrimitive.Icon asChild>
        <ChevronDownIcon className="group-data-[state=open]:rotate-180 transition-transform duration-200 ease-in-out size-3 opacity-50 flex-shrink-0 ml-1" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
  );
}
function SelectContent({
  className,
  children,
  position = "popper",
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Content>) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Content
        data-slot="select-content"
        className={cn(
          "bg-popover max-w-[var(--radix-select-trigger-width)] text-popover-foreground",
          "data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95",
          "data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95",
          "data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2",
          "data-[side=top]:slide-in-from-bottom-2 relative z-50 max-h-96 min-w-[8rem] overflow-hidden rounded-md border shadow-md",
          position === "popper" &&
            "data-[side=bottom]:translate-y-1 data-[side=left]:-translate-x-1 data-[side=right]:translate-x-1 data-[side=top]:-translate-y-1",
          className,
        )}
        position={position}
        {...props}
      >
        <SelectScrollUpButton />
        <SelectPrimitive.Viewport
          className={cn(
            "p-1",
            position === "popper" &&
              "h-[var(--radix-select-trigger-height)] w-full min-w-[var(--radix-select-trigger-width)] scroll-my-1",
          )}
        >
          {children}
        </SelectPrimitive.Viewport>
        <SelectScrollDownButton />
      </SelectPrimitive.Content>
    </SelectPrimitive.Portal>
  );
}

function SelectLabel({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Label>) {
  return (
    <SelectPrimitive.Label
      data-slot="select-label"
      className={cn("px-2 py-1.5 text-sm font-medium", className)}
      {...props}
    />
  );
}

function SelectItem({
  className,
  children,
  icon,
  color,
  description,
  ...props
}: Omit<SelectOption, "label">) {
  const renderIcon = () => {
    if (typeof icon === "object" && icon !== null && "icon" in icon) {
      return (
        <Icon
          icon={icon}
          className="size-3 data-[state=checked]:text-foreground data-[state=checked]:bg-muted-foreground/10"
        />
      );
    }
    return icon;
  };

  return (
    <SelectPrimitive.Item
      data-slot="select-item"
      className={cn(
        "focus:bg-muted text-left focus:text-accent-foreground cursor-pointer hover:bg-muted dark:hover:bg-primary/10 data-[state=checked]:bg-muted",
        "data-[state=checked]:dark:bg-primary/10 [&_svg:not([class*='text-'])]:text-muted-foreground relative flex w-full items-center gap-2 rounded-sm",
        "py-1.5 pr-8 pl-2 text-2xs outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
        "[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 *:[span]:last:flex *:[span]:last:items-center *:[span]:last:gap-2",
        className,
      )}
      {...props}
    >
      {icon ? (
        renderIcon()
      ) : color ? (
        <span
          className="block size-2 rounded-full"
          style={{ backgroundColor: color }}
        />
      ) : null}
      <span className="absolute right-2 flex size-3.5 items-center justify-center">
        <SelectPrimitive.ItemIndicator>
          <CheckIcon className="size-4" />
        </SelectPrimitive.ItemIndicator>
      </span>
      <div className="flex flex-col w-full">
        <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
        {description && (
          <SelectItemDescription>{description}</SelectItemDescription>
        )}
      </div>
    </SelectPrimitive.Item>
  );
}

function SelectItemDescription({
  className,
  ...props
}: React.ComponentProps<"p">) {
  return (
    <p
      data-slot="select-item-description"
      className={cn("text-muted-foreground text-2xs", className)}
      {...props}
    />
  );
}

function SelectSeparator({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Separator>) {
  return (
    <SelectPrimitive.Separator
      data-slot="select-separator"
      className={cn("bg-border pointer-events-none -mx-1 my-1 h-px", className)}
      {...props}
    />
  );
}

function SelectScrollUpButton({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.ScrollUpButton>) {
  return (
    <SelectPrimitive.ScrollUpButton
      data-slot="select-scroll-up-button"
      className={cn(
        "flex cursor-default items-center justify-center py-1",
        className,
      )}
      {...props}
    >
      <ChevronUpIcon className="size-4" />
    </SelectPrimitive.ScrollUpButton>
  );
}

function SelectScrollDownButton({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.ScrollDownButton>) {
  return (
    <SelectPrimitive.ScrollDownButton
      data-slot="select-scroll-down-button"
      className={cn(
        "flex cursor-default items-center justify-center py-1",
        className,
      )}
      {...props}
    >
      <ChevronDownIcon className="size-4" />
    </SelectPrimitive.ScrollDownButton>
  );
}

export {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectItemDescription,
  SelectLabel,
  SelectScrollDownButton,
  SelectScrollUpButton,
  SelectSeparator,
  SelectTrigger,
  SelectValue
};

