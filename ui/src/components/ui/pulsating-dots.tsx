import { cn } from "@/lib/utils";
import { motion } from "framer-motion";

type PulsatingDotProps = {
  size?: number;
  color?: string;
};

const sizes: Record<number, string> = {
  1: "size-1",
  2: "size-2",
  3: "size-3",
  4: "size-4",
  5: "size-5",
};

const colors: Record<string, string> = {
  foreground: "bg-foreground",
  primary: "bg-primary",
  background: "bg-background",
  red: "bg-red-500",
  blue: "bg-blue-500",
  green: "bg-green-500",
  yellow: "bg-yellow-500",
};

export default function PulsatingDots({
  size = 1,
  color = "red",
}: PulsatingDotProps) {
  const sizeClass = sizes[size];
  const colorClass = colors[color];

  return (
    <div className="flex items-center justify-center">
      <div className="flex space-x-1">
        <motion.div
          className={cn("rounded-full", sizeClass, colorClass)}
          animate={{
            scale: [1, 1.5, 1],
            opacity: [0.5, 1, 0.5],
          }}
          transition={{
            duration: 1,
            ease: "easeInOut",
            repeat: Infinity,
          }}
        />
        <motion.div
          className={cn("rounded-full", sizeClass, colorClass)}
          animate={{
            scale: [1, 1.5, 1],
            opacity: [0.5, 1, 0.5],
          }}
          transition={{
            duration: 1,
            ease: "easeInOut",
            repeat: Infinity,
            delay: 0.3,
          }}
        />
        <motion.div
          className={cn("rounded-full", sizeClass, colorClass)}
          animate={{
            scale: [1, 1.5, 1],
            opacity: [0.5, 1, 0.5],
          }}
          transition={{
            duration: 1,
            ease: "easeInOut",
            repeat: Infinity,
            delay: 0.6,
          }}
        />
      </div>
    </div>
  );
}
