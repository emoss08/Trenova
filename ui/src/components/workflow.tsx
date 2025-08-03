/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn, truncateText } from "@/lib/utils";
import { CaretSortIcon } from "@radix-ui/react-icons";
import { useState } from "react";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "./ui/sidebar";

interface Workflow {
  id: string;
  name: string;
  color: "purple" | "blue" | "green" | "yellow" | "red";
}

const workflows: Workflow[] = [
  {
    id: "operation-management",
    name: "Operation",
    color: "purple",
  },
  {
    id: "billing-management",
    name: "Billing",
    color: "blue",
  },
  {
    id: "accounting-management",
    name: "Accounting",
    color: "green",
  },
  {
    id: "human-resources-management",
    name: "Human Resources",
    color: "yellow",
  },
  {
    id: "fleet-management",
    name: "Fleet",
    color: "red",
  },
];

const colorStyles = {
  purple: {
    border: "border-purple-600 dark:border-purple-500",
    bg: "bg-purple-200 dark:bg-purple-600/20",
    hoverBg: "hover:bg-purple-100 dark:hover:bg-purple-600/30",
    text: "text-purple-600 dark:text-purple-200",
    title: "text-purple-700 dark:text-purple-400",
    icon: "text-purple-600 dark:text-purple-400",
    menuItem: "bg-purple-600",
  },
  blue: {
    border: "border-blue-600 dark:border-blue-500",
    bg: "bg-blue-200 dark:bg-blue-600/20",
    hoverBg: "hover:bg-blue-100 dark:hover:bg-blue-600/30",
    text: "text-blue-600 dark:text-blue-200",
    title: "text-blue-700 dark:text-blue-400",
    icon: "text-blue-600 dark:text-blue-400",
    menuItem: "bg-blue-600",
  },
  green: {
    border: "border-green-600 dark:border-green-500",
    bg: "bg-green-200 dark:bg-green-600/20",
    hoverBg: "hover:bg-green-100 dark:hover:bg-green-600/30",
    text: "text-green-600 dark:text-green-200",
    title: "text-green-700 dark:text-green-400",
    icon: "text-green-600 dark:text-green-400",
    menuItem: "bg-green-600",
  },
  yellow: {
    border: "border-yellow-600 dark:border-yellow-500",
    bg: "bg-yellow-200 dark:bg-yellow-600/20",
    hoverBg: "hover:bg-yellow-100 dark:hover:bg-yellow-600/30",
    text: "text-yellow-600 dark:text-yellow-200",
    title: "text-yellow-700 dark:text-yellow-400",
    icon: "text-yellow-600 dark:text-yellow-400",
    menuItem: "bg-yellow-600",
  },
  red: {
    border: "border-red-600 dark:border-red-500",
    bg: "bg-red-200 dark:bg-red-600/20",
    hoverBg: "hover:bg-red-100 dark:hover:bg-red-600/30",
    text: "text-red-600 dark:text-red-200",
    title: "text-red-700 dark:text-red-400",
    icon: "text-red-600 dark:text-red-400",
    menuItem: "bg-red-600",
  },
};

// TODO: Add a workflow placeholder

export function WorkflowPlaceholder() {
  const [selectedWorkflow, setSelectedWorkflow] = useState<Workflow>(
    workflows[4],
  );
  const currentColor = colorStyles[selectedWorkflow.color];

  return (
    <SidebarMenu className="">
      <SidebarMenuItem className="flex flex-col gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className={cn(
                "group col-span-full flex h-14 select-none items-center gap-x-4 rounded-lg border border-dashed p-1 px-4 transition-colors hover:cursor-pointer",
                currentColor.border,
                currentColor.bg,
                currentColor.hoverBg,
              )}
            >
              <div className="flex flex-1 flex-col">
                <p className={cn("text-sm", currentColor.text)}>Workflow</p>
                <h2
                  className={cn(
                    "truncate font-bold text-lg",
                    currentColor.title,
                  )}
                >
                  {truncateText(`${selectedWorkflow.name} Management`, 20)}
                </h2>
              </div>
              <div className="ml-auto flex items-center justify-center">
                <CaretSortIcon className={cn("size-5", currentColor.icon)} />
              </div>
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="start"
            side="bottom"
            sideOffset={10}
            className="w-[var(--radix-dropdown-menu-trigger-width)] truncate"
          >
            {workflows.map((workflow) => (
              <DropdownMenuCheckboxItem
                key={workflow.id}
                checked={selectedWorkflow.id === workflow.id}
                onCheckedChange={() => setSelectedWorkflow(workflow)}
              >
                <div className="flex items-center gap-2">
                  <div
                    className={cn(
                      "size-2 rounded-full",
                      colorStyles[workflow.color].menuItem,
                    )}
                  />
                  <span>{workflow.name}</span>
                </div>
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
