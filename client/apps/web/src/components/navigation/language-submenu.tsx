import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import {
  DEFAULT_LOCALE,
  LOCALE_FLAGS,
  LOCALE_NAMES,
  LOCALES,
  type Locale,
} from "@trenova/shared/i18n/generated/locales";
import { storeLocale } from "@trenova/shared/i18n/provider";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { useLocale, useT } from "@trenova/shared/i18n/use-t";
import {
  DropdownMenuCheckboxItem,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { UpdateMySettings, User } from "@trenova/shared/types/user";
import { Languages } from "lucide-react";
import { toast } from "sonner";

type SwitchArgs = {
  next: Locale;
  previous: Locale;
  user: Pick<User, "timezone" | "timeFormat"> | null;
  save: (values: UpdateMySettings) => Promise<unknown>;
  onSaved: () => void;
};

/**
 * switchLocale applies the language first and persists it second, so the menu closes on the
 * new language rather than after a round trip. If the save fails the interface goes back:
 * leaving it in a language the server rejected would mean the screen and the user's next
 * invoice silently disagree about which language they read.
 */
export async function switchLocale({
  next,
  previous,
  user,
  save,
  onSaved,
}: SwitchArgs): Promise<void> {
  if (next === previous) return;

  await setLocale(next);
  storeLocale(next);

  try {
    // settings is a whole-object PATCH, so the other two preferences ride along unchanged
    // rather than being reset to their defaults by omission.
    await save({
      locale: next,
      timezone: user?.timezone ?? "America/New_York",
      timeFormat: user?.timeFormat ?? "12-hour",
    });
  } catch {
    await setLocale(previous);
    storeLocale(previous);
    return;
  }

  onSaved();
}

/**
 * LanguageSubmenu switches the interface language from the user menu.
 *
 * The preference is stored server-side, because it also decides the language of the emails
 * and PDFs this user is sent - those render in a worker with no browser to ask. The catalog
 * swap is applied optimistically so the menu closes on the new language rather than after a
 * round trip, and the store update that follows keeps I18nProvider (which resolves from the
 * signed-in user) from resolving back to the previous one.
 */
export function LanguageSubmenu() {
  const t = useT();
  const active = useLocale();
  const user = useAuthStore((s) => s.user);

  const { mutateAsync: saveLocale } = useApiMutation<User, UpdateMySettings>({
    mutationFn: (values) => apiService.userService.updateMySettings(values),
    resourceName: "User Settings",
    onSuccess: (updatedUser) => {
      useAuthStore.getState().setUser(updatedUser);
    },
  });

  const choose = (next: Locale) =>
    switchLocale({
      next,
      previous: active,
      user,
      save: saveLocale,
      onSaved: () =>
        toast.success(t("Language updated"), {
          description: t("Your emails and documents will use it too."),
        }),
    });

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <Languages className="mr-2 size-4" />
        <span>{t("Language")}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent sideOffset={5}>
          {LOCALES.map((locale) => (
            <DropdownMenuCheckboxItem
              key={locale}
              checked={locale === (active ?? DEFAULT_LOCALE)}
              onCheckedChange={() => void choose(locale)}
              className="cursor-pointer"
            >
              <span aria-hidden="true" className="mr-2">
                {LOCALE_FLAGS[locale]}
              </span>
              {LOCALE_NAMES[locale]}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
