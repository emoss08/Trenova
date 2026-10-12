import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

export type ThemeChoice = "light" | "dark" | "system";

type Palette = "light" | "dark";

function MiniPage({ palette, className }: { palette: Palette; className?: string }) {
  const vars = {
    light: {
      canvas: "bg-(--theme-preview-light-canvas)",
      panel: "bg-(--theme-preview-light-panel)",
      line: "bg-(--theme-preview-light-line)",
      ink: "bg-(--theme-preview-light-ink)",
    },
    dark: {
      canvas: "bg-(--theme-preview-dark-canvas)",
      panel: "bg-(--theme-preview-dark-panel)",
      line: "bg-(--theme-preview-dark-line)",
      ink: "bg-(--theme-preview-dark-ink)",
    },
  }[palette];

  return (
    <span
      aria-hidden
      className={cn(
        "absolute inset-0 grid grid-cols-[28%_1fr] gap-1.75 p-2",
        vars.canvas,
        className,
      )}
    >
      <i className={cn("rounded-[5px]", vars.panel)} />
      <span className="flex flex-col gap-1.25 pt-0.5">
        <b className={cn("h-2 w-1/2 rounded-full", vars.ink)} />
        <b className={cn("h-1.5 rounded-full", vars.line)} />
        <b className={cn("h-1.5 w-[70%] rounded-full", vars.line)} />
      </span>
    </span>
  );
}

/**
 * Light, dark, or following the system, each drawn as a small page in that theme. The
 * system card is split on the diagonal so it reads as "either".
 */
export function ThemePicker({
  value,
  onChange,
}: {
  value: ThemeChoice;
  onChange: (theme: ThemeChoice) => void;
}) {
  const t = useT();
  const choices: { value: ThemeChoice; label: string }[] = [
    { value: "light", label: t("Light") },
    { value: "dark", label: t("Dark") },
    { value: "system", label: t("System") },
  ];

  return (
    <div
      role="radiogroup"
      aria-label={t("Theme")}
      className="grid grid-cols-1 gap-2.5 min-[620px]:grid-cols-3"
    >
      {choices.map((choice) => {
        const selected = value === choice.value;
        return (
          <button
            key={choice.value}
            type="button"
            role="radio"
            aria-checked={selected}
            onClick={() => onChange(choice.value)}
            className={cn(
              "group/theme ui-focus-ring flex flex-col gap-2 rounded-md text-left text-xs",
              selected ? "text-foreground font-medium" : "text-muted-foreground",
            )}
          >
            <span
              className={cn(
                "relative h-18 overflow-hidden rounded-md ring-1 transition-[translate,box-shadow] duration-200 group-hover/theme:-translate-y-0.5",
                selected ? "ring-brand ring-2" : "ring-border",
              )}
            >
              {choice.value === "system" ? (
                <>
                  <MiniPage palette="light" className="[clip-path:polygon(0_0,100%_0,0_100%)]" />
                  <MiniPage
                    palette="dark"
                    className="[clip-path:polygon(100%_0,100%_100%,0_100%)]"
                  />
                </>
              ) : (
                <MiniPage palette={choice.value} />
              )}
            </span>
            {choice.label}
          </button>
        );
      })}
    </div>
  );
}
