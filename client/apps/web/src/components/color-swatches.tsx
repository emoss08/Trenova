import { CheckIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/** The colours a record can be tagged with in one click; any other is typed as a hex code. */
export const COLOR_SWATCHES: readonly string[] = [
  "#E2E2E2",
  "#ff75c3",
  "#ffa647",
  "#ffe83f",
  "#9fff5b",
  "#70e2ff",
  "#cd93ff",
  "#09203f",
  "#ff7575",
  "#b284be",
  "#ff7f50",
  "#deb887",
  "#5f9ea0",
  "#ffd700",
  "#6a5acd",
  "#ff4500",
  "#2e8b57",
  "#4682b4",
  "#d2691e",
  "#6495ed",
  "#dc143c",
  "#008b8b",
  "#b8860b",
  "#006400",
  "#8b008b",
  "#556b2f",
  "#ff8c00",
  "#9932cc",
  "#8b0000",
  "#e9967a",
  "#20b2aa",
  "#ff6347",
  "#ffa07a",
  "#da70d6",
  "#f0e68c",
  "#40e0d0",
  "#ee82ee",
  "#696969",
  "#ffdab9",
  "#87cefa",
];

function sameColor(a: string | null | undefined, b: string): boolean {
  return !!a && a.toLowerCase() === b.toLowerCase();
}

/** The swatches to pick from; the one in use is ticked. */
export function ColorSwatchGrid({
  value,
  onPick,
}: {
  value?: string | null;
  onPick: (color: string) => void;
}) {
  const t = useT();

  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(20px,1fr))] gap-1.5">
      {COLOR_SWATCHES.map((color) => {
        const chosen = sameColor(value, color);
        return (
          <button
            key={color}
            type="button"
            aria-label={t("Color {0}", color)}
            aria-pressed={chosen}
            style={{ background: color }}
            className="ui-focus-ring flex size-6 cursor-pointer items-center justify-center rounded-md active:scale-105"
            onClick={() => onPick(color)}
          >
            {chosen ? (
              <CheckIcon aria-hidden className={cn("size-3.5 text-white mix-blend-difference")} />
            ) : null}
          </button>
        );
      })}
    </div>
  );
}
