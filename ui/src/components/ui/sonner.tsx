import {
  faCircleCheck,
  faCircleInfo,
  faCircleXmark,
  faTriangleExclamation,
} from "@fortawesome/pro-solid-svg-icons";
import { useTheme } from "next-themes";
import { Toaster as Sonner } from "sonner";
import { Icon } from "./icons";

type ToasterProps = React.ComponentProps<typeof Sonner>;

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme();

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      icons={{
        info: <Icon icon={faCircleInfo} className="size-4 text-blue-500" />,
        warning: (
          <Icon
            icon={faTriangleExclamation}
            className="size-4 text-yellow-500"
          />
        ),
        error: <Icon icon={faCircleXmark} className="size-4 text-red-500" />,
        success: (
          <Icon icon={faCircleCheck} className="size-4 text-green-500" />
        ),
      }}
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-muted-foreground text-xs",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };

