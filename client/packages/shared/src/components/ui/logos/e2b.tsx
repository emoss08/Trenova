import { cn } from "@trenova/shared/lib/utils";
import e2bIcon from "./e2b-icon.jpeg";

/**
 * E2B's icon as Brandfetch publishes it: its white mark on black, available only as an image,
 * so it is drawn as one rather than redrawn as a path.
 */
export function E2BLogo({ className }: { className?: string }) {
  return (
    <img
      src={e2bIcon}
      alt=""
      role="presentation"
      decoding="async"
      className={cn("rounded-badge size-4 object-contain", className)}
    />
  );
}
