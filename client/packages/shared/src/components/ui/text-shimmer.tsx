import { cn } from "@trenova/shared/lib/utils";
import React, { type CSSProperties, type ReactNode } from "react";

export type TextShimmerProps = Omit<React.HTMLAttributes<HTMLElement>, "children"> & {
  children: ReactNode;
  as?: React.ElementType;
  className?: string;
  /** Seconds for the band to cross the text once. */
  duration?: number;
  /** Off, the text renders plain in the same element, so a step can stop shimmering when it finishes. */
  active?: boolean;
};

/**
 * Text something is working on, with a lighter band crossing it on a loop.
 *
 * The band is a copy of the text in a soft-edged window (`ui-text-shimmer-sweep`
 * in app.css): the window slides across while the copy slides back, so only
 * transforms animate and the browser never repaints the text. The copy is hidden
 * from assistive technology, which reads the text once, and plain text is copied
 * by CSS, so the element's text content is the text once. The colours default to
 * subtle text with a full-strength band; a surface sets `--shimmer-base` and
 * `--shimmer-highlight` on the element to use its own.
 */
function TextShimmerComponent({
  children,
  as: Component = "p",
  className,
  duration = 2,
  active = true,
  style,
  ...props
}: TextShimmerProps) {
  if (!active) {
    return (
      <Component className={className} style={style} {...props}>
        {children}
      </Component>
    );
  }

  return (
    <Component
      className={cn("ui-text-shimmer", className)}
      style={{ "--shimmer-duration": `${duration}s`, ...style } as CSSProperties}
      {...props}
    >
      {children}
      <span aria-hidden="true" className="ui-text-shimmer-sweep">
        {/* Plain text is copied by CSS from data-text, so the element's text
            stays the text once; rich children are copied as elements. */}
        {typeof children === "string" ? <span data-text={children} /> : <span>{children}</span>}
      </span>
    </Component>
  );
}

export const TextShimmer = React.memo(TextShimmerComponent);
