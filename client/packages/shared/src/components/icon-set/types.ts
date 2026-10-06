import type { FC, SVGProps } from "react";

export interface IconProps extends SVGProps<SVGSVGElement> {
  size?: number;
  color?: string;
}

export type IconComponent = FC<IconProps>;
