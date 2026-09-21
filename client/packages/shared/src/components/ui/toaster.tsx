import type { CSSProperties } from "react";
import { Toaster as Sonner, type ToasterProps } from "sonner";
import { useTheme } from "../theme-provider";
import { ToastErrorIcon, ToastInfoIcon, ToastSuccessIcon, ToastWarningIcon } from "./toast-icons";

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme } = useTheme();

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      style={
        {
          "--normal-bg": "var(--raised)",
          "--normal-text": "var(--foreground)",
          "--normal-border": "var(--border)",
          "--border-radius": "var(--radius-surface)",
        } as CSSProperties
      }
      toastOptions={{
        classNames: {
          toast: "!font-sans !shadow-overlay !text-sm",
          title: "!font-medium",
          description: "!text-foreground-muted",
          actionButton: "!rounded-md !bg-ink !text-ink-foreground !font-medium",
          cancelButton: "!rounded-md !bg-surface-active !text-foreground !font-medium",
        },
      }}
      icons={{
        success: <ToastSuccessIcon />,
        error: <ToastErrorIcon />,
        warning: <ToastWarningIcon />,
        info: <ToastInfoIcon />,
      }}
      {...props}
    />
  );
};

export { Toaster };
