import { useTheme } from "../ui/theme-provider";

export function RainbowTopBar() {
  const { isRainbowAnimationActive } = useTheme();

  return (
    <div
      className={`bg-rainbow-gradient-light bg-200% h-1 ${
        isRainbowAnimationActive ? "animate-rainbow-flow" : ""
      }`}
    />
  );
}
