import { cn } from "@/lib/utils";
import { motion } from "framer-motion";
import { Sparkles } from "lucide-react";

interface RecommendedBadgeProps {
  text?: string;
  size?: "sm" | "md" | "lg";
  variant?: "default" | "premium" | "success" | "warning";
  className?: string;
}

const variantClasses = {
  default: "from-purple-500 via-pink-500 to-orange-400",
  premium: "from-yellow-400 via-yellow-500 to-amber-500",
  success: "from-emerald-400 via-green-500 to-teal-500",
  warning: "from-orange-400 via-red-500 to-pink-500",
};

const sparklePositions = [
  { top: "10%", left: "15%", delay: 0 },
  { top: "20%", right: "20%", delay: 0.5 },
  { bottom: "15%", left: "25%", delay: 1 },
  { bottom: "25%", right: "15%", delay: 1.5 },
  { top: "50%", left: "5%", delay: 2 },
  { top: "40%", right: "8%", delay: 2.5 },
];

export default function RecommendedBadge({
  text = "Recommended",
  variant = "default",
  className,
}: RecommendedBadgeProps) {
  return (
    <div className="relative inline-block">
      {/* Animated sparkles */}
      {sparklePositions.map((position, index) => (
        <motion.div
          key={index}
          className="absolute pointer-events-none"
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
          <Sparkles className="size-3 text-yellow-300" />
        </motion.div>
      ))}

      {/* Main badge */}
      <motion.div
        className={cn(
          "relative overflow-hidden rounded font-semibold px-2 text-xs text-white cursor-pointer select-none",
          className,
        )}
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ type: "spring", stiffness: 300, damping: 20 }}
      >
        {/* Gradient background */}
        <div
          className={cn(
            "absolute inset-0 bg-gradient-to-r opacity-90",
            variantClasses[variant],
          )}
        />

        {/* Shimmer effect */}
        <motion.div
          className="absolute inset-0 bg-gradient-to-r from-transparent via-white/30 to-transparent -skew-x-12"
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
        <motion.div
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
      </motion.div>
    </div>
  );
}
