import { useTheme } from "next-themes";
import { Toaster as Sonner, ToasterProps } from "sonner";

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme();

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast: "!bg-popover !border !border-border",
          title: "!text-sm !text-primary",
          description: "!text-sm !text-muted-foreground",
          icon: "!text-primary",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
