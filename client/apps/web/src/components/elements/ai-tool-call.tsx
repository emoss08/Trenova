"use client";

import { useT } from "@trenova/shared/i18n/use-t";
import * as React from "react";

import * as CollapsiblePrimitive from "@radix-ui/react-collapsible";
import {
  AlertTriangle,
  Check,
  ChevronDown,
  Clock,
  Loader2,
  ShieldQuestion,
  Wrench,
  X,
} from "lucide-react";

import { cn } from "@trenova/shared/lib/utils";

type ToolCallState = "pending" | "running" | "completed" | "error" | "awaiting-approval" | "denied";

interface AiToolCallContextValue {
  name: string;
  state: ToolCallState;
  isOpen: boolean;
}

const AiToolCallContext = React.createContext<AiToolCallContextValue | null>(null);

function useToolCallContext() {
  const context = React.useContext(AiToolCallContext);
  if (!context) {
    throw new Error("AiToolCall components must be used within <AiToolCall>");
  }
  return context;
}

interface AiToolCallProps {
  name: string;
  state: ToolCallState;
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  children?: React.ReactNode;
  className?: string;
}

function AiToolCall({
  name,
  state,
  defaultOpen = false,
  open: controlledOpen,
  onOpenChange,
  children,
  className,
}: AiToolCallProps) {
  const [uncontrolledOpen, setUncontrolledOpen] = React.useState(defaultOpen);

  const isControlled = controlledOpen !== undefined;
  const isOpen = isControlled ? controlledOpen : uncontrolledOpen;

  const handleOpenChange = React.useCallback(
    (open: boolean) => {
      if (!isControlled) {
        setUncontrolledOpen(open);
      }
      onOpenChange?.(open);
    },
    [isControlled, onOpenChange],
  );

  React.useEffect(() => {
    if (state === "completed" || state === "error") {
      handleOpenChange(true);
    }
  }, [state, handleOpenChange]);

  const contextValue = React.useMemo(() => ({ name, state, isOpen }), [name, state, isOpen]);

  return (
    <AiToolCallContext.Provider value={contextValue}>
      <CollapsiblePrimitive.Root
        data-slot="ai-tool-call"
        open={isOpen}
        onOpenChange={handleOpenChange}
        className={cn(
          "border-border bg-card text-card-foreground overflow-hidden rounded-lg border",
          className,
        )}
      >
        {children}
      </CollapsiblePrimitive.Root>
    </AiToolCallContext.Provider>
  );
}

interface AiToolCallHeaderProps {
  children?: React.ReactNode;
  className?: string;
}

function AiToolCallHeader({ children, className }: AiToolCallHeaderProps) {
  const t = useT();

  const { name, state, isOpen } = useToolCallContext();

  const stateConfig = React.useMemo(() => {
    const configs: Record<
      ToolCallState,
      { icon: React.ReactNode; label: string; className: string }
    > = {
      pending: {
        icon: <Clock className="size-3.5" />,
        label: t("Pending"),
        className: "bg-muted text-muted-foreground",
      },
      running: {
        icon: <Loader2 className="size-3.5 animate-spin" />,
        label: t("Running"),
        className: "bg-info-subtle text-info-foreground dark:bg-info-subtle dark:text-info-foreground",
      },
      completed: {
        icon: <Check className="size-3.5" />,
        label: t("Completed"),
        className: "bg-success-subtle text-success-foreground dark:bg-success-subtle dark:text-success-foreground",
      },
      error: {
        icon: <X className="size-3.5" />,
        label: t("Error"),
        className: "bg-danger-subtle text-danger-foreground dark:bg-danger-subtle dark:text-danger-foreground",
      },
      "awaiting-approval": {
        icon: <ShieldQuestion className="size-3.5" />,
        label: t("Awaiting Approval"),
        className: "bg-warning-subtle text-warning-foreground dark:bg-warning-subtle dark:text-warning-foreground",
      },
      denied: {
        icon: <AlertTriangle className="size-3.5" />,
        label: t("Denied"),
        className: "bg-warning-subtle text-warning-foreground dark:bg-warning-subtle dark:text-warning-foreground",
      },
    };
    return configs[state];
  }, [state, t]);

  return (
    <CollapsiblePrimitive.Trigger
      data-slot="ai-tool-call-header"
      className={cn(
"ui-focus-ring hover:bg-muted/50 flex w-full items-center gap-3 px-4 py-3 text-sm font-medium transition-colors",
        className,
      )}
    >
      <div className="bg-muted flex size-8 shrink-0 items-center justify-center rounded-md">
        <Wrench className="text-muted-foreground size-4" />
      </div>
      <div className="flex flex-1 items-center gap-2 text-left">
        <span className="font-mono text-sm">{name}</span>
        <span
          className={cn(
            "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
            stateConfig.className,
          )}
        >
          {stateConfig.icon}
          {t(stateConfig.label)}
        </span>
      </div>
      {children}
      <ChevronDown
        className={cn(
          "text-muted-foreground size-4 shrink-0 transition-transform duration-200",
          isOpen && "rotate-180",
        )}
      />
    </CollapsiblePrimitive.Trigger>
  );
}

interface AiToolCallContentProps {
  children?: React.ReactNode;
  className?: string;
}

function AiToolCallContent({ children, className }: AiToolCallContentProps) {
  return (
    <CollapsiblePrimitive.Content
      data-slot="ai-tool-call-content"
      className={cn(
        "border-border data-[state=closed]:animate-collapsible-up data-[state=open]:animate-collapsible-down border-t",
        className,
      )}
    >
      <div className="space-y-4 p-4">{children}</div>
    </CollapsiblePrimitive.Content>
  );
}

interface AiToolCallInputProps {
  input: Record<string, unknown>;
  className?: string;
}

function AiToolCallInput({ input, className }: AiToolCallInputProps) {
  const t = useT();

  const formattedJson = React.useMemo(() => JSON.stringify(input, null, 2), [input]);

  return (
    <div data-slot="ai-tool-call-input" className={cn("space-y-1.5", className)}>
      <span className="text-muted-foreground text-xs font-medium tracking-wider uppercase">
        {t("Input")}
      </span>
      <pre className="bg-muted/50 text-foreground overflow-x-auto rounded-md p-3 font-mono text-xs">
        {formattedJson}
      </pre>
    </div>
  );
}

interface AiToolCallOutputProps {
  children?: React.ReactNode;
  className?: string;
}

function AiToolCallOutput({ children, className }: AiToolCallOutputProps) {
  const t = useT();

  return (
    <div data-slot="ai-tool-call-output" className={cn("space-y-1.5", className)}>
      <span className="text-muted-foreground text-xs font-medium tracking-wider uppercase">
        {t("Output")}
      </span>
      <div className="bg-muted/50 overflow-x-auto rounded-md p-3 text-sm">{children}</div>
    </div>
  );
}

interface AiToolCallErrorProps {
  error: string;
  className?: string;
}

function AiToolCallError({ error, className }: AiToolCallErrorProps) {
  const t = useT();

  return (
    <div data-slot="ai-tool-call-error" className={cn("space-y-1.5", className)}>
      <span className="text-xs font-medium tracking-wider text-danger-foreground uppercase dark:text-danger-foreground">
        {t("Error")}
      </span>
      <div className="rounded-md border border-danger-border bg-danger-subtle p-3 text-sm text-danger-foreground dark:border-danger-border dark:bg-danger-subtle/30 dark:text-danger-foreground">
        {error}
      </div>
    </div>
  );
}

export {
  AiToolCall,
  AiToolCallHeader,
  AiToolCallContent,
  AiToolCallInput,
  AiToolCallOutput,
  AiToolCallError,
};
export type { AiToolCallProps, ToolCallState };
