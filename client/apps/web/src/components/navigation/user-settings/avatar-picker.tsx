import { Shuffle01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  AVATAR_PRESETS,
  AVATAR_STYLE_AUTO,
  avatarBackground,
  avatarMesh,
} from "@trenova/shared/lib/avatar-gradient";
import { cn } from "@trenova/shared/lib/utils";

const swatchClass =
  "ui-focus-ring relative size-8 rounded-full bg-cover ring-1 ring-foreground/10 ring-inset transition-transform duration-200 hover:scale-108";
const selectedClass = "outline-2 outline-offset-2 outline-brand";

/**
 * The gradient a person without a photo is drawn with: Auto, made from their username,
 * or one of the named presets. Shuffle picks a preset other than the current one.
 */
export function AvatarPicker({
  seed,
  value,
  onChange,
}: {
  seed: string;
  value: string;
  onChange: (style: string) => void;
}) {
  const t = useT();

  const shuffle = () => {
    const others = AVATAR_PRESETS.filter((preset) => preset.id !== value);
    const next = others[Math.floor(Math.random() * others.length)];
    if (next) onChange(next.id);
  };

  return (
    <div
      role="radiogroup"
      aria-label={t("Avatar style")}
      className="flex flex-wrap items-center gap-2"
    >
      <button
        type="button"
        role="radio"
        aria-checked={value === AVATAR_STYLE_AUTO}
        title={t("Auto, from your username")}
        onClick={() => onChange(AVATAR_STYLE_AUTO)}
        className={cn(swatchClass, value === AVATAR_STYLE_AUTO && selectedClass)}
        style={{ backgroundImage: avatarBackground(seed) }}
      >
        <span className="text-3xs text-foreground-on-solid absolute inset-0 grid place-items-center font-semibold">
          {t("Auto")}
        </span>
      </button>
      {AVATAR_PRESETS.map((preset) => (
        <button
          key={preset.id}
          type="button"
          role="radio"
          aria-checked={value === preset.id}
          aria-label={t(preset.label)}
          title={t(preset.label)}
          onClick={() => onChange(preset.id)}
          className={cn(swatchClass, value === preset.id && selectedClass)}
          style={{ backgroundImage: avatarMesh(preset.hues, preset.chroma) }}
        />
      ))}
      <button
        type="button"
        aria-label={t("Shuffle")}
        title={t("Shuffle")}
        onClick={shuffle}
        className="ui-focus-ring bg-card text-muted-foreground hover:text-foreground grid size-8 place-items-center rounded-full border transition-colors"
      >
        <Shuffle01Icon className="size-3.5" />
      </button>
    </div>
  );
}
