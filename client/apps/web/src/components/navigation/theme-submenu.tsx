import { useTheme } from "@trenova/shared/components/theme-provider";
import {
  Monitor01Icon,
  Moon01Icon,
  SunIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import {
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";

const THEMES = ["light", "dark", "system"] as const;
type ThemeChoice = (typeof THEMES)[number];

const THEME_ICONS: Record<ThemeChoice, IconComponent> = {
  light: SunIcon,
  dark: Moon01Icon,
  system: Monitor01Icon,
};

function isThemeChoice(value: unknown): value is ThemeChoice {
  return THEMES.some((theme) => theme === value);
}

/**
 * Light, dark or the system's, from the user menu. The trigger wears the icon of the theme
 * in use, and the menu stays open on a choice so the change is seen before it is left.
 */
export function ThemeSubmenu() {
  const t = useT();
  const { theme, setTheme } = useTheme();
  const labels: Record<ThemeChoice, string> = {
    light: t("Light"),
    dark: t("Dark"),
    system: t("System"),
  };
  const TriggerIcon = isThemeChoice(theme) ? THEME_ICONS[theme] : Monitor01Icon;

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <TriggerIcon className="mr-2 size-4" />
        <span>{t("Theme")}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent sideOffset={5} className="min-w-40">
          <DropdownMenuRadioGroup
            value={theme}
            onValueChange={(next) => {
              if (isThemeChoice(next)) setTheme(next);
            }}
          >
            {THEMES.map((choice) => {
              const Icon = THEME_ICONS[choice];
              return (
                <DropdownMenuRadioItem key={choice} value={choice} className="cursor-pointer">
                  <Icon className="mr-2 size-4" />
                  {labels[choice]}
                </DropdownMenuRadioItem>
              );
            })}
          </DropdownMenuRadioGroup>
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
