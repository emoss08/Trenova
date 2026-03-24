import { Toaster as Sonner, type ToasterProps } from "sonner";
import { useTheme } from "../theme-provider";
import {
  ToastErrorIcon,
  ToastInfoIcon,
  ToastSuccessIcon,
  ToastWarningIcon,
} from "./toast-icons";

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme();

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
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
