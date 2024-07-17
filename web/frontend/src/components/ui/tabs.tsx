/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import * as TabsPrimitive from "@radix-ui/react-tabs";
import * as React from "react";

import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { Badge } from "./badge";

const Tabs = TabsPrimitive.Root;

const TabsList = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({ className, ...props }, ref) => (
  <ScrollArea className="w-full whitespace-nowrap">
    <TabsPrimitive.List
      ref={ref}
      className={cn(
        "flex h-10 mb-1.5 items-center justify-between bg-transparent border-b border-border overflow-hidden",
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
    isNotification?: boolean;
    notificationCount?: number;
  }
>(
  (
    {
      className,
      isError,
      errorCount,
      isNotification,
      notificationCount,
      children,
      ...props
    },
    ref,
  ) => (
    <TabsPrimitive.Trigger
      ref={ref}
      className={cn(
        "relative inline-flex flex-1 items-center justify-center whitespace-nowrap px-3 py-1.5 text-sm text-foreground font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
        "data-[state=active]:border-b-4 data-[state=active]:border-white data-[state=active]:-mb-2 data-[state=active]:z-10",
        "data-[state=inactive]:border-b-2 data-[state=inactive]:text-muted-foreground data-[state=inactive]:-mb-1.5 data-[state=active]:z-10",
        " data-[state=inactive]:hover:border-b-4 data-[state=inactive]:hover:border-white data-[state=inactive]:hover:-mb-2 data-[state=inactive]:hover:text-foreground",
        isError
          ? "data-[state=inactive]:border-red-500 data-[state=active]:border-red-500"
          : "border-transparent",
        isNotification
          ? "data-[state=active]:border-green-500"
          : "border-transparent",
        className,
      )}
      {...props}
    >
      {children}
      {isError && (
        <Badge className="ml-2 px-1.5 py-0" variant="inactive" withDot={false}>
          {errorCount}
        </Badge>
      )}
      {isNotification && (
        <Badge className="ml-2 px-1.5 py-0" variant="active" withDot={false}>
          {notificationCount}
        </Badge>
      )}
    </TabsPrimitive.Trigger>
  ),
);
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

