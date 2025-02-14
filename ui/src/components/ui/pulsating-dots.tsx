import { cn } from "@/lib/utils";
import { motion } from "framer-motion";

type SizeOptions = 0.5 | 1 | 2 | 3 | 4 | 5;

type ColorOptions =
  | "foreground"
  | "primary"
  | "background"
  | "red"
  | "blue"
  | "green"
  | "yellow";

type DotsProps = {
  size?: SizeOptions;
  color?: ColorOptions;
};

const sizes: Record<SizeOptions, string> = {
  0.5: "size-0.5",
  1: "size-1",
  2: "size-2",
  3: "size-3",
  4: "size-4",
  5: "size-5",
};

const colors: Record<ColorOptions, string> = {
  foreground: "bg-foreground",
  primary: "bg-primary",
  background: "bg-background",
  red: "bg-red-500",
  blue: "bg-blue-500",
  green: "bg-green-500",
  yellow: "bg-yellow-500",
};

export function PulsatingDots({ size = 1, color = "red" }: DotsProps) {
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

export function FadeLoaderDots({ size = 1, color = "red" }: DotsProps) {
  const circleVariants = {
    hidden: { opacity: 0 },
    visible: { opacity: 1 },
  };

  const sizeClass = sizes[size];
  const colorClass = colors[color];

  return (
    <div className="flex items-center justify-center space-x-2">
      {[...Array(3)].map((_, index) => (
        <motion.div
          key={index}
          className={cn("h-4 w-4 rounded-full", sizeClass, colorClass)}
          variants={circleVariants}
          initial="hidden"
          animate="visible"
          transition={{
            duration: 0.9,
            delay: index * 0.2,
            repeat: Infinity,
            repeatType: "reverse",
          }}
        ></motion.div>
      ))}
    </div>
  );
}
