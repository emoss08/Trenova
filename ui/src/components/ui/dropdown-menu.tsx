import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import * as React from "react";

import { cn } from "@/lib/utils";
import { CheckIcon, ChevronRightIcon, CircleIcon } from "@radix-ui/react-icons";
const statusColors = {
  danger:
    "text-red-600 focus:text-red-600 focus:bg-red-600/10 hover:text-red-600 hover:bg-red-600/10 dark:text-red-600 dark:focus:text-red-600 dark:focus:bg-red-600/20 dark:hover:bg-red-600/20 dark:hover:text-red-400",
  warning:
    "text-yellow-500 focus:text-yellow-600 focus:bg-yellow-600/10 hover:text-yellow-600 hover:bg-yellow-600/10 dark:text-yellow-600 dark:focus:text-yellow-600 dark:focus:bg-yellow-600/20 dark:hover:bg-yellow-600/20 dark:hover:text-yellow-400",
  info: "text-blue-600 focus:text-blue-600 focus:bg-blue-600/10 hover:text-blue-600 hover:bg-blue-600/20 dark:text-blue-600 dark:focus:text-blue-600 dark:focus:bg-blue-600/20 dark:hover:bg-blue-600/20 dark:hover:text-blue-400",
  success:
    "text-green-500 focus:text-green-600 focus:bg-green-600/10 hover:text-green-600 hover:bg-green-600/10 dark:text-green-600 dark:focus:text-green-600 dark:focus:bg-green-600/20 dark:hover:bg-green-600/20 dark:hover:text-green-400",
};
const DropdownMenu = DropdownMenuPrimitive.Root;

const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;

const DropdownMenuGroup = DropdownMenuPrimitive.Group;

const DropdownMenuPortal = DropdownMenuPrimitive.Portal;

const DropdownMenuSub = DropdownMenuPrimitive.Sub;

const DropdownMenuRadioGroup = DropdownMenuPrimitive.RadioGroup;

const DropdownMenuSubTrigger = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.SubTrigger>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.SubTrigger> & {
    inset?: boolean;
    color?: "danger" | "warning" | "info" | "success";
    startContent?: React.ReactNode;
    description?: string;
    titleClassProps?: string;
    descriptionClassProps?: string;
  }
>(
  (
    {
      className,
      inset,
      color,
      startContent,
      description,
      titleClassProps,
      descriptionClassProps,
      children,
      ...props
    },
    ref,
  ) => (
    <DropdownMenuPrimitive.SubTrigger
      ref={ref}
      className={cn(
        "flex cursor-default gap-2 select-none items-center rounded-sm px-2 py-1 text-sm outline-hidden focus:bg-muted-foreground/10 data-[state=open]:bg-muted-foreground/10 [&_svg]:pointer-events-none [&_svg]:size-3 [&_svg]:shrink-0",
        inset && "pl-8",
        color && statusColors[color],
        className,
      )}
      {...props}
    >
      {startContent && (
        <span className="mr-2 flex items-center">{startContent}</span>
      )}
      <span className="flex flex-col">
        <span className={cn("text-sm", titleClassProps)}>{children}</span>
        {description && (
          <span
            className={cn(
              "text-muted-foreground text-2xs",
              descriptionClassProps,
            )}
          >
            {description}
          </span>
        )}
      </span>
      <ChevronRightIcon className="ml-auto size-4" />
    </DropdownMenuPrimitive.SubTrigger>
  ),
);
DropdownMenuSubTrigger.displayName =
  DropdownMenuPrimitive.SubTrigger.displayName;

const DropdownMenuSubContent = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.SubContent>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.SubContent>
>(({ className, ...props }, ref) => (
  <DropdownMenuPrimitive.SubContent
    ref={ref}
    className={cn(
      "z-50 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-lg data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2",
      className,
    )}
    {...props}
  />
));
DropdownMenuSubContent.displayName =
  DropdownMenuPrimitive.SubContent.displayName;

const DropdownMenuContent = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Content>
>(({ className, sideOffset = 4, ...props }, ref) => (
  <DropdownMenuPrimitive.Portal>
    <DropdownMenuPrimitive.Content
      ref={ref}
      sideOffset={sideOffset}
      className={cn(
        "z-50 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md",
        "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2",
        className,
      )}
      {...props}
    />
  </DropdownMenuPrimitive.Portal>
));
DropdownMenuContent.displayName = DropdownMenuPrimitive.Content.displayName;

const DropdownMenuItem = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.Item>,
  Omit<
    React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Item>,
    "title" | "children"
  > & {
    title: string;
    inset?: boolean;
    color?: "danger" | "warning" | "info" | "success";
    startContent?: React.ReactNode;
    endContent?: React.ReactNode;
    description?: string;
    titleClassProps?: string;
    descriptionClassProps?: string;
  }
>(
  (
    {
      className,
      inset,
      color,
      disabled,
      title,
      startContent,
      description,
      titleClassProps,
      descriptionClassProps,
      endContent,
      ...props
    },
    ref,
  ) => (
    <DropdownMenuPrimitive.Item
      ref={ref}
      disabled={disabled}
      className={cn(
        "group relative flex cursor-pointer select-none items-center gap-2 focus:bg-muted-foreground/10 hover:bg-muted-foreground/10 rounded-sm px-2 py-1 text-sm outline-hidden focus:text-accent-foreground transition-colors data-disabled:opacity-50 data-disabled:pointer-events-none [&>svg]:size-3 [&>svg]:shrink-0",
        inset && "pl-8",
        color && statusColors[color],

        className,
      )}
      {...props}
    >
      {startContent && (
        <span className="mr-2 flex items-center">{startContent}</span>
      )}
      <span className="flex flex-col">
        <span className={cn("text-sm", titleClassProps)}>{title}</span>
        {description && (
          <span
            className={cn(
              "text-muted-foreground text-2xs",
              descriptionClassProps,
            )}
          >
            {description}
          </span>
        )}
      </span>
      {endContent && (
        <span className="ml-auto flex items-center">{endContent}</span>
      )}
    </DropdownMenuPrimitive.Item>
  ),
);
DropdownMenuItem.displayName = DropdownMenuPrimitive.Item.displayName;

const DropdownMenuCheckboxItem = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.CheckboxItem>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.CheckboxItem>
>(({ className, children, checked, ...props }, ref) => (
  <DropdownMenuPrimitive.CheckboxItem
    ref={ref}
    className={cn(
      "relative flex cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-xs outline-hidden transition-colors focus:bg-muted-foreground/10 focus:text-accent-foreground data-disabled:cursor-not-allowed data-disabled:opacity-50 data-disabled:pointer-events-none",
      className,
    )}
    checked={checked}
    {...props}
  >
    <span className="absolute left-2 flex size-3.5 items-center justify-center">
      <DropdownMenuPrimitive.ItemIndicator>
        <CheckIcon className="size-3" />
      </DropdownMenuPrimitive.ItemIndicator>
    </span>
    {children}
  </DropdownMenuPrimitive.CheckboxItem>
));
DropdownMenuCheckboxItem.displayName =
  DropdownMenuPrimitive.CheckboxItem.displayName;

const DropdownMenuRadioItem = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.RadioItem>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.RadioItem>
>(({ className, children, ...props }, ref) => (
  <DropdownMenuPrimitive.RadioItem
    ref={ref}
    className={cn(
      "relative flex cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-hidden transition-colors focus:bg-muted focus:text-accent-foreground data-disabled:pointer-events-none data-disabled:opacity-50",
      className,
    )}
    {...props}
  >
    <span className="absolute left-2 flex size-3.5 items-center justify-center">
      <DropdownMenuPrimitive.ItemIndicator>
        <CircleIcon className="size-2 fill-current" />
      </DropdownMenuPrimitive.ItemIndicator>
    </span>
    {children}
  </DropdownMenuPrimitive.RadioItem>
));
DropdownMenuRadioItem.displayName = DropdownMenuPrimitive.RadioItem.displayName;

const DropdownMenuLabel = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.Label>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Label> & {
    inset?: boolean;
  }
>(({ className, inset, ...props }, ref) => (
  <DropdownMenuPrimitive.Label
    ref={ref}
    className={cn("p-1 text-xs font-semibold", inset && "pl-8", className)}
    {...props}
  />
));
DropdownMenuLabel.displayName = DropdownMenuPrimitive.Label.displayName;

const DropdownMenuSeparator = React.forwardRef<
  React.ComponentRef<typeof DropdownMenuPrimitive.Separator>,
  React.ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Separator>
>(({ className, ...props }, ref) => (
  <DropdownMenuPrimitive.Separator
    ref={ref}
    className={cn("-mx-1 my-1 h-px bg-muted", className)}
    {...props}
  />
));
DropdownMenuSeparator.displayName = DropdownMenuPrimitive.Separator.displayName;

const DropdownMenuShortcut = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLSpanElement>) => {
  return (
    <span
      className={cn("ml-auto text-xs tracking-widest opacity-60", className)}
      {...props}
    />
  );
};
DropdownMenuShortcut.displayName = "DropdownMenuShortcut";

export {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger
};

