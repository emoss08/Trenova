import { DialogContent } from "@trenova/shared/components/ui/dialog";
import { useRef, type ComponentProps } from "react";

/**
 * An integration dialog's frame. It opens with focus on the dialog itself, so a screen reader
 * reads its title and description first and Tab moves to the first control. Left to the
 * default, focus lands on the vendor's help link in the description, and Enter would start
 * leaving Trenova.
 */
export function IntegrationDialogContent(props: ComponentProps<typeof DialogContent>) {
  const popupRef = useRef<HTMLDivElement>(null);

  return <DialogContent {...props} ref={popupRef} initialFocus={popupRef} />;
}
