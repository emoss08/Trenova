import { cn } from "@trenova/shared/lib/utils";
import { Sparkles } from "lucide-react";
import { m } from "motion/react";
import React from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@trenova/shared/components/ui/tooltip";

interface RecommendedBadgeProps {
  text?: string;
  size?: "sm" | "md" | "lg";
  variant?: "default" | "premium" | "success" | "warning";
  className?: string;
  tooltip?: React.ReactNode;
}

const variantClasses = {
  default: "from-accent-violet via-accent-rose to-warning",
  premium: "from-warning via-warning to-warning",
  success: "from-success via-success to-accent-teal",
  warning: "from-warning via-danger to-accent-rose",
};

const sparklePositions = [
  { top: "10%", left: "15%", delay: 0 },
  { top: "20%", right: "20%", delay: 0.5 },
  { bottom: "15%", left: "25%", delay: 1 },
  { bottom: "25%", right: "15%", delay: 1.5 },
  { top: "50%", left: "5%", delay: 2 },
  { top: "40%", right: "8%", delay: 2.5 },
];

export function RecommendedBadge({
  text = "Recommended",
  variant = "default",
  className,
  tooltip,
}: RecommendedBadgeProps) {
  const badgeContent = (
    <div className="relative inline-block">
      {/* Animated sparkles */}
      {sparklePositions.map((position, index) => (
        <m.div
          key={index}
          className="pointer-events-none absolute"
          style={position}
          initial={{ opacity: 0, scale: 0 }}
          animate={{
            opacity: [0, 1, 0],
            scale: [0, 1, 0],
            rotate: [0, 180, 360],
          }}
          transition={{
            duration: 2,
            delay: position.delay,
            repeat: Number.POSITIVE_INFINITY,
            repeatDelay: 3,
          }}
        >
          <Sparkles className="size-3 text-warning-foreground" />
        </m.div>
      ))}

      {/* Main badge */}
      <m.div
        className={cn(
          "relative overflow-hidden rounded px-2 text-xs font-semibold text-white select-none",
          className,
        )}
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ type: "spring", stiffness: 300, damping: 20 }}
      >
        {/* Gradient background */}
        <div
          className={cn("absolute inset-0 bg-gradient-to-r opacity-90", variantClasses[variant])}
        />

        {/* Shimmer effect */}
        <m.div
          className="absolute inset-0 -skew-x-12 bg-gradient-to-r from-transparent via-white/30 to-transparent"
          initial={{ x: "-100%" }}
          animate={{ x: "200%" }}
          transition={{
            duration: 2,
            repeat: Number.POSITIVE_INFINITY,
            repeatDelay: 3,
            ease: "easeInOut",
          }}
        />

        {/* Pulsing glow */}
        <m.div
          className={cn(
            "absolute inset-0 bg-gradient-to-r opacity-50 blur-sm",
            variantClasses[variant],
          )}
          animate={{
            opacity: [0.3, 0.7, 0.3],
            scale: [1, 1.1, 1],
          }}
          transition={{
            duration: 2,
            repeat: Number.POSITIVE_INFINITY,
            ease: "easeInOut",
          }}
        />

        {/* Text content */}
        <span className="relative z-10 flex items-center gap-1">
          <Sparkles className="size-3" />
          {text}
        </span>

        {/* Inner highlight */}
        <div className="absolute inset-0 rounded bg-gradient-to-t from-transparent to-white/20" />
      </m.div>
    </div>
  );

  if (!tooltip) {
    return badgeContent;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger render={badgeContent} />
        <TooltipContent>{tooltip}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
