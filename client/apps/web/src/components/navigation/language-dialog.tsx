import { useApiMutation } from "@/hooks/use-api-mutation";
import { filterLanguageOptions, languageOptions } from "@/lib/language-options";
import { applyLocale, persistLocale } from "@/lib/locale-switch";
import { apiService } from "@/services/api";
import { CheckIcon, SearchLgIcon } from "@trenova/shared/components/icons";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import type { Locale } from "@trenova/shared/i18n/generated/locales";
import { loadCatalog } from "@trenova/shared/i18n/runtime";
import { useLocale, useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { UpdateMySettings, User } from "@trenova/shared/types/user";
import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";

/**
 * Warming the catalog on hover or focus turns the choice into a cache hit in the common
 * case, so the spinner is the exception rather than the rule. A failed warm is not worth
 * reporting: the choice retries it and reports for real.
 */
function prefetch(locale: Locale) {
  void loadCatalog(locale).catch(() => {});
}

/**
 * Picks the interface language: every language Trenova speaks, named in itself and in the
 * language on screen now, with a search for when the list is long or unreadable.
 *
 * The catalog swap is applied before the dialog closes, so it closes on the new language;
 * the stored preference settles behind it (see persistLocale), because it also decides the
 * language of the emails and PDFs this user is sent.
 */
export function LanguageDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const active = useLocale();
  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();
  const searchRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [pending, setPending] = useState<Locale | null>(null);

  const { mutateAsync: saveLocale } = useApiMutation<User, UpdateMySettings>({
    mutationFn: (values) => apiService.userService.updateMySettings(values),
    resourceName: "User Settings",
    onSuccess: (updatedUser) => {
      useAuthStore.getState().setUser(updatedUser);
    },
  });

  const options = filterLanguageOptions(languageOptions(active), query);

  const close = () => {
    onOpenChange(false);
    setQuery("");
  };

  const choose = async (next: Locale) => {
    if (pending !== null) return;
    if (next === active) {
      close();
      return;
    }

    const previous = active;
    setPending(next);
    try {
      await applyLocale(next);
    } finally {
      setPending(null);
    }
    close();

    void persistLocale({
      next,
      previous,
      user,
      save: saveLocale,
      onSaved: () => {
        // What the server renders (the report catalog, messages it composed) follows the
        // stored preference, so it is fetched again once the server holds the new one.
        void queryClient.invalidateQueries();
      },
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (next) onOpenChange(true);
        else close();
      }}
    >
      <DialogContent size="lg" initialFocus={searchRef}>
        <DialogHeader>
          <DialogTitle>{t("Choose your language")}</DialogTitle>
        </DialogHeader>
        <Input
          ref={searchRef}
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            const first = options[0];
            if (event.key === "Enter" && first) {
              event.preventDefault();
              void choose(first.locale);
            }
          }}
          placeholder={t("Search languages")}
          aria-label={t("Search languages")}
          leftElement={<SearchLgIcon className="text-foreground-subtle size-3.5" />}
        />
        {options.length === 0 ? (
          <p className="text-muted-foreground py-8 text-center text-sm">
            {t("No languages match “{0}”.", query.trim())}
          </p>
        ) : (
          <ul
            aria-label={t("Languages")}
            className="-mx-1 grid max-h-[60vh] grid-cols-1 gap-1 overflow-y-auto px-1 sm:grid-cols-2 md:grid-cols-3"
          >
            {options.map((option) => {
              const selected = option.locale === active;
              const isPending = pending === option.locale;
              return (
                <li key={option.locale}>
                  <button
                    type="button"
                    aria-pressed={selected}
                    disabled={pending !== null}
                    onClick={() => void choose(option.locale)}
                    onPointerEnter={() => prefetch(option.locale)}
                    onFocus={() => prefetch(option.locale)}
                    className={cn(
                      "ui-focus-ring flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left transition-colors outline-none disabled:cursor-default",
                      selected ? "bg-accent text-accent-foreground" : "hover:bg-surface-hover",
                    )}
                  >
                    <span className="grid min-w-0 flex-1">
                      <span lang={option.locale} className="truncate text-sm font-medium">
                        {option.nativeName}
                      </span>
                      <span className="text-muted-foreground truncate text-xs">
                        {option.displayName}
                      </span>
                    </span>
                    {isPending ? (
                      <Spinner className="size-4 shrink-0" />
                    ) : selected ? (
                      <CheckIcon className="text-brand size-4 shrink-0" />
                    ) : null}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </DialogContent>
    </Dialog>
  );
}
