import { cn } from "@/lib/utils";
import { ReactNode, createContext, useContext, useState } from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

interface TabsProps {
  defaultValue: string;
  value: string;
  className?: string;
  onValueChange: (value: string) => void;
  children: ReactNode;
}

interface TabsListProps {
  children: ReactNode;
}

interface TabsTriggerProps {
  value: string;
  children: ReactNode;
}

interface TabsContentProps {
  value: string;
  children: ReactNode;
}

const TabsContext = createContext<{
  activeTab: string;
  setActiveTab: (value: string) => void;
} | null>(null);

export const Tabs = ({
  defaultValue,
  value,
  className,
  onValueChange,
  children,
}: TabsProps) => {
  const [activeTab, setActiveTab] = useState(defaultValue);

  const handleValueChange = (value: string) => {
    setActiveTab(value);
    onValueChange(value);
  };

  return (
    <TabsContext.Provider
      value={{ activeTab: value || activeTab, setActiveTab: handleValueChange }}
    >
      <div className={cn("flex", className)}>{children}</div>
    </TabsContext.Provider>
  );
};

export const TabsList = ({ children }: TabsListProps) => {
  return (
    <div className="border-border border-r border-dashed">
      <ul>{children}</ul>
    </div>
  );
};

export const TabsTrigger = ({ value, children }: TabsTriggerProps) => {
  const context = useContext(TabsContext);
  if (!context) throw new Error("TabsTrigger must be used within a Tabs");

  const humanReadableValue = value.replace(/([A-Z])/g, " $1").trim();
  const capitalizedValue =
    humanReadableValue.charAt(0).toUpperCase() + humanReadableValue.slice(1);

  const { activeTab, setActiveTab } = context;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <li
            className={cn(
              "cursor-pointer p-2 rounded-md m-2",

              activeTab === value
                ? "bg-foreground text-background"
                : "hover:bg-muted hover:text-primary",
            )}
            onClick={() => setActiveTab(value)}
          >
            {children}
          </li>
        </TooltipTrigger>
        <TooltipContent side="left" sideOffset={10}>
          <p>{capitalizedValue}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export const TabsContent = ({ value, children }: TabsContentProps) => {
  const context = useContext(TabsContext);
  if (!context) throw new Error("TabsContent must be used within a Tabs");

  const { activeTab } = context;

  return (
    <div
      className={cn(
        "transition-opacity duration-500 mt-2 ml-4 ring-offset-background focus-visible:outline-none",
        activeTab === value ? "opacity-100" : "h-0 overflow-hidden opacity-0",
      )}
    >
      {activeTab === value && children}
    </div>
  );
};
