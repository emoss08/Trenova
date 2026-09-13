import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import {
  DEFAULT_LOCALE,
  LOCALE_NAMES,
  LOCALES,
  type Locale,
} from "@trenova/shared/i18n/generated/locales";
import { LocaleFlag } from "@trenova/shared/i18n/locale-flag";
import { storeLocale } from "@trenova/shared/i18n/provider";
import { loadCatalog, setLocale } from "@trenova/shared/i18n/runtime";
import { useLocale, useT } from "@trenova/shared/i18n/use-t";
import {
  DropdownMenuCheckboxItem,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { UpdateMySettings, User } from "@trenova/shared/types/user";
import { Languages } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

type PersistArgs = {
  next: Locale;
  previous: Locale;
  user: Pick<User, "timezone" | "timeFormat"> | null;
  save: (values: UpdateMySettings) => Promise<unknown>;
  onSaved: () => void;
};

/**
 * applyLocale swaps the catalog and is the only part the user waits on. The catalog is a
 * lazily-imported chunk of a few hundred kilobytes, so on a cold cache this is a real
 * network round trip rather than an instant toggle - which is why the caller shows a
 * pending state around it and why the menu prefetches on hover.
 */
export async function applyLocale(next: Locale): Promise<void> {
  await setLocale(next);
  storeLocale(next);
}

/**
 * persistLocale saves the preference after the interface has already changed. It is not
 * awaited by the click handler: the language is visibly applied first, and this settles
 * behind it. The save still matters - the stored locale also decides the language of the
 * emails and PDFs this user is sent, which render in a worker with no browser to ask - so a
 * failure puts the interface back rather than leaving the screen and their next invoice
 * disagreeing about which language they read.
 */
export async function persistLocale({
  next,
  previous,
  user,
  save,
  onSaved,
}: PersistArgs): Promise<void> {
  try {
    // settings is a whole-object PATCH, so the other two preferences ride along unchanged
    // rather than being reset to their defaults by omission.
    await save({
      locale: next,
      timezone: user?.timezone ?? "America/New_York",
      timeFormat: user?.timeFormat ?? "12-hour",
    });
  } catch {
    await applyLocale(previous);
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

  const [pending, setPending] = useState<Locale | null>(null);

  const choose = useCallback(
    async (next: Locale) => {
      if (next === active || pending !== null) return;

      const previous = active;
      setPending(next);
      try {
        await applyLocale(next);
      } finally {
        setPending(null);
      }

      // Deliberately not awaited: the language is already on screen, and making the menu
      // wait on the round trip is the stall this pending state exists to avoid.
      void persistLocale({
        next,
        previous,
        user,
        save: saveLocale,
        onSaved: () =>
          toast.success(t("Language updated"), {
            description: t("Your emails and documents will use it too."),
          }),
      });
    },
    [active, pending, user, saveLocale, t],
  );

  // Warming the catalog on hover turns the click into a cache hit in the common case, so
  // the spinner below is the exception rather than the rule.
  const prefetch = useCallback((locale: Locale) => {
    void loadCatalog(locale).catch(() => {
      // A failed warm is not worth reporting: the click retries it and reports for real.
    });
  }, []);

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <Languages className="mr-2 size-4" />
        <span>{t("Language")}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent sideOffset={5}>
          {LOCALES.map((locale) => {
            const isPending = pending === locale;

            return (
              <DropdownMenuCheckboxItem
                key={locale}
                checked={locale === (active ?? DEFAULT_LOCALE)}
                onCheckedChange={() => void choose(locale)}
                onPointerEnter={() => prefetch(locale)}
                onFocus={() => prefetch(locale)}
                disabled={pending !== null && !isPending}
                className="cursor-pointer"
              >
                <span className="mr-2 inline-flex size-3.5 items-center justify-center">
                  {isPending ? <Spinner className="size-3.5" /> : <LocaleFlag locale={locale} />}
                </span>
                {LOCALE_NAMES[locale]}
              </DropdownMenuCheckboxItem>
            );
          })}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
