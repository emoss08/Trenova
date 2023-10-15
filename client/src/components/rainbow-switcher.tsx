import { useTheme } from "./theme-provider";

export function RainbowToggle() {
  const { isRainbowAnimationActive, toggleRainbowAnimation } = useTheme();
  return (
    <button onClick={toggleRainbowAnimation}>
      {isRainbowAnimationActive
        ? "Turn off Rainbow Animation"
        : "Turn on Rainbow Animation"}
    </button>
  );
}
