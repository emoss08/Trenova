import * as TabsPrimitive from "@radix-ui/react-tabs";
import * as React from "react";

import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";

const Tabs = TabsPrimitive.Root;

const TabsList = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({ className, ...props }, ref) => (
  <ScrollArea className="w-full whitespace-nowrap">
    <TabsPrimitive.List
      ref={ref}
      className={cn(
        "flex h-10 mt-5 mb-1.5 items-center justify-between bg-transparent border-b border-border overflow-hidden",
        className,
      )}
      {...props}
    />

    <ScrollBar orientation="horizontal" />
  </ScrollArea>
));

TabsList.displayName = TabsPrimitive.List.displayName;

const TabsTrigger = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger> & {
    isError?: boolean;
    errorCount?: number;
  }
>(({ className, isError, errorCount, children, ...props }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(
      "relative inline-flex flex-1 items-center justify-center whitespace-nowrap px-3 py-1.5 text-sm text-foreground font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
      "data-[state=active]:border-b-2 data-[state=active]:border-blue-600 data-[state=active]:-mb-1.5 data-[state=active]:z-10",
      "data-[state=inactive]:border-b-2 data-[state=inactive]:border-transparent data-[state=inactive]:text-muted-foreground data-[state=inactive]:-mb-1.5 data-[state=active]:z-10",
      isError ? "data-[state=inactive]:border-red-500" : "border-transparent",
      className,
    )}
    {...props}
  >
    {children}
    {isError && (
      <span className="relative ml-2 rounded-full bg-red-500 px-2 text-xs font-medium text-white">
        {errorCount}
      </span>
    )}
  </TabsPrimitive.Trigger>
));
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName;

const TabsContent = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Content
    ref={ref}
    className={cn(
      "mt-2 ring-offset-background focus-visible:outline-none",
      className,
    )}
    {...props}
  />
));
TabsContent.displayName = TabsPrimitive.Content.displayName;

export { Tabs, TabsContent, TabsList, TabsTrigger };
