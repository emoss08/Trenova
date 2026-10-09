import { Button, type ButtonProps } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { deskPanelIconClass } from "../desk-chat/desk-button-styles";

/** The panel's icon commands: a 28px square in the Desk's muted ink that fills on hover. */
export function AssistantIconButton({
  className,
  ...props
}: Omit<ButtonProps, "variant" | "size">) {
  return (
    <Button
      variant="quiet"
      size="bare"
      className={cn(deskPanelIconClass, className)}
      {...props}
    />
  );
}
