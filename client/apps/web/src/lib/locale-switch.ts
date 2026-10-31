import { storeLocale } from "@trenova/shared/i18n/provider";
import { setLocale } from "@trenova/shared/i18n/runtime";
import type { Locale } from "@trenova/shared/i18n/generated/locales";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { UpdateMySettings, User } from "@trenova/shared/types/user";

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
 * pending state around it and why the language picker prefetches on hover.
 *
 * The signed-in user's stored locale moves with it, and that is not bookkeeping: I18nProvider
 * resolves from that stored preference and re-runs whenever the active locale changes, so a
 * runtime swap that leaves the user record behind is immediately resolved back and undone.
 * Before this, the language only stuck once the server round trip returned and updated the
 * store - which is the delay this whole path was supposed to remove.
 */
export async function applyLocale(next: Locale): Promise<void> {
  await setLocale(next);
  storeLocale(next);

  const { user, setUser } = useAuthStore.getState();
  if (user) setUser({ ...user, locale: next });
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
