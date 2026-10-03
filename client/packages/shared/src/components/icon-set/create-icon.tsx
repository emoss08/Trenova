import type { ReactNode } from "react";
import type { IconComponent, IconProps } from "./types";

export function createIcon(displayName: string, children: ReactNode): IconComponent {
  function Icon({ size = 24, color = "currentColor", ...props }: IconProps) {
    return (
      <svg
        viewBox="0 0 24 24"
        width={size}
        height={size}
        stroke={color}
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
        {...props}
      >
        {children}
      </svg>
    );
  }
  Icon.displayName = displayName;
  return Icon;
}
