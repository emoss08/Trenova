/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */
import { Button } from "@/components/ui/button";
import { InternalLink } from "@/components/ui/link";
import { useCookieStore } from "@/stores/AuthStore";

export function CookieConsent() {
  const [isCookieConsentGiven, setIsCookieConsentGiven] = useCookieStore(
    (state) => [state.isCookieConsentGiven, state.setIsCookieConsentGiven],
  );
  const [, setEssentialCookies] = useCookieStore((state) => [
    state.essentialCookies,
    state.setEssentialCookies,
  ]);

  const [, setFunctionalCookies] = useCookieStore((state) => [
    state.functionalCookies,
    state.setFunctionalCookies,
  ]);

  const [, setPerformanceCookies] = useCookieStore((state) => [
    state.performanceCookies,
    state.setPerformanceCookies,
  ]);

  const handleCookieConsent = () => {
    setIsCookieConsentGiven(true);
    setEssentialCookies(true);
    setFunctionalCookies(true);
    setPerformanceCookies(true);
  };

  const setDefaultSettings = () => {
    setIsCookieConsentGiven(true);
    setEssentialCookies(true);
  };

  // TODO: Add a way to set cookies based on user preferences

  return isCookieConsentGiven ? null : (
    <section className="fixed bottom-16 left-12 mx-auto max-w-md rounded-2xl border border-border bg-background p-4">
      <h2 className="font-semibold text-foreground">🍪 We use cookies!</h2>
      <span className="absolute right-5 top-5 flex size-3">
        <span className="absolute inline-flex size-full animate-ping rounded-full bg-orange-400 opacity-100"></span>
        <span className="relative inline-flex size-3 rounded-full bg-orange-600 ring-1 ring-background"></span>
      </span>
      <p className="mt-4 text-xs text-foreground">
        Hi, this website uses essential cookies to ensure its proper operation
        and tracking cookies to understand how you interact with it. The latter
        will be set only after consent.{" "}
        <InternalLink to="#">Let me choose</InternalLink>.
      </p>
      <p className="mt-3 text-xs text-muted-foreground">
        Closing this modal default settings will be saved.
      </p>

      <div className="mt-4 grid shrink-0 grid-cols-2 gap-4">
        <Button onClick={handleCookieConsent} size="sm" variant="gooeyRight">
          Accept all
        </Button>
        <Button size="sm" variant="outline">
          Preferences
        </Button>
        <Button size="sm" variant="outline">
          Reject all
        </Button>
        <Button onClick={setDefaultSettings} size="sm" variant="outline">
          Close
        </Button>
      </div>
    </section>
  );
}
