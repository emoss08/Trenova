import { cn } from "@trenova/shared/lib/utils";
import { CHIP_FILTERS, type ChipFilterId } from "./saved-views";
import { useCommandCenterUrl } from "./url-state";

const CHIP_TONE: Record<ChipFilterId, { on: string; off: string }> = {
  "at-risk": {
    on: "bg-destructive/12 text-destructive border-destructive/30",
    off: "bg-muted text-muted-foreground border-transparent hover:text-foreground",
  },
  reefer: {
    on: "bg-brand/12 text-brand border-brand/30",
    off: "bg-muted text-muted-foreground border-transparent hover:text-foreground",
  },
  today: {
    on: "bg-foreground/8 text-foreground border-foreground/20",
    off: "bg-muted text-muted-foreground border-transparent hover:text-foreground",
  },
};

export function FilterChipRow() {
  const [{ chips }, setUrl] = useCommandCenterUrl();

  const toggle = (chip: ChipFilterId) => {
    const next = chips.includes(chip)
      ? chips.filter((c) => c !== chip)
      : [...chips, chip];
    void setUrl({ chips: next.length === 0 ? null : next, page: 1, expanded: null });
  };

  return (
    <div className="flex flex-wrap items-center gap-1">
      {CHIP_FILTERS.map((chip) => {
        const isOn = chips.includes(chip.id);
        const tone = CHIP_TONE[chip.id];
        return (
          <button
            key={chip.id}
            type="button"
            onClick={() => toggle(chip.id)}
            aria-pressed={isOn}
            className={cn(
              "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-semibold tracking-wide uppercase transition-colors",
              isOn ? tone.on : tone.off,
            )}
          >
            <span>{chip.label}</span>
            {isOn && <span aria-hidden>×</span>}
          </button>
        );
      })}
    </div>
  );
}
