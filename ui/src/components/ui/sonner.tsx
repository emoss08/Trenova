import {
  faCircleCheck,
  faCircleExclamation,
  faCircleInfo,
  faTriangleExclamation,
} from "@fortawesome/pro-regular-svg-icons";
import { Toaster as Sonner, type ToasterProps } from "sonner";
import { useTheme } from "../theme-provider";
import { Icon } from "./icons";

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme();

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      richColors
      icons={{
        info: <Icon icon={faCircleInfo} className="size-4 text-blue-500" />,
        warning: (
          <Icon icon={faCircleExclamation} className="size-4 text-yellow-500" />
        ),
        error: (
          <Icon icon={faTriangleExclamation} className="size-4 text-red-500" />
        ),
        success: (
          <Icon icon={faCircleCheck} className="size-4 text-green-500" />
        ),
      }}
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground font-medium",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground font-medium",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };

