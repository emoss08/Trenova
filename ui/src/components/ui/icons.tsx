/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/**
 * Credits: Acorn1010 - https://gist.github.com/acorn1010/9f4621d3dfc33052ffd84f6c2a06d4d6.
 *
 * Permission was granted by the author to use this code. Please ask for permission before using this code.
 */

import { type IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import { type CSSProperties } from "react";
import { cn } from "../../lib/utils";

export type IconProp = IconDefinition;

type IconSize = "xs" | "sm" | "1x" | "lg" | "2x" | "3x" | "4x";
const SIZE_CLASSES = {
  xs: "text-[.75em]",
  sm: "text-[.875em]",
  "1x": "text-[1em]",
  lg: "text-lg",
  "2x": "text-[2em]",
  "3x": "text-[3em]",
  "4x": "text-[4em]",
} satisfies { [key in IconSize]: string };
const FIXED_WIDTH_CLASSES = {
  xs: "w-[.8em] h-[.8em]",
  sm: "w-[1em] h-[1em]",
  "1x": "w-[1.25em] h-[1.25em]",
  lg: "w-[1.5em] h-[1.5em]",
  "2x": "w-[2.5em] h-[2.5em]",
  "3x": "w-[3.75em] h-[3.75em]",
  "4x": "w-[5em] h-[5em]",
} satisfies { [key in IconSize]: string };

type IconProps = {
  className?: string;
  color?: string;
  icon: IconProp;
  style?: CSSProperties;
  fixedWidth?: boolean;
  spin?: boolean;
  title?: string;
  size?: IconSize;
  onClick?: (event: React.MouseEvent<SVGSVGElement>) => void;
};

/**
 * Displays a FontAwesome icon. We use this wrapper exclusively instead of using <FontAwesomeIcon />
 * because FontAwesomeIcon adds 20 KB of gzip size--wtf?!
 */
export function Icon(props: IconProps) {
  const { className, color, icon, fixedWidth, spin, size, style, onClick } =
    props;
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [width, height, _, _2, d] = icon.icon;
  const paths = Array.isArray(d) ? d : [d];
  return (
    <svg
      className={cn(
        "box-content inline-block h-[1em]",
        fixedWidth && FIXED_WIDTH_CLASSES[size || "1x"],
        spin && "animate-spin",
        SIZE_CLASSES[size || "1x"],
        className,
      )}
      onClick={onClick}
      style={style}
      role="img"
      xmlns="http://www.w3.org/2000/svg"
      data-prefix={icon.prefix}
      data-icon={icon.iconName}
      viewBox={`0 0 ${width} ${height}`}
    >
      {paths.length > 1 ? (
        <g>
          {paths.map((pathData, i) => (
            <path
              key={i}
              className={cn(i === 0 && "opacity-40")}
              fill={color || "currentColor"}
              d={pathData}
              style={style}
            />
          ))}
        </g>
      ) : (
        paths.map((pathData, i) => (
          <path
            key={i}
            fill={color || "currentColor"}
            d={pathData}
            style={style}
          />
        ))
      )}
    </svg>
  );
}
