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

import { Separator } from "../ui/separator";
import { ColorBlindSwitcher } from "./appearance/color-mode-switcher";
import { ThemeSwitcher } from "./appearance/theme-switcher";

function Preferences() {
  return (
    <>
      <div className="space-y-3">
        <div>
          <h1 className="text-foreground text-2xl font-semibold">
            Personalize Your Experience
          </h1>
          <p className="text-muted-foreground text-sm">
            Customize your settings for an optimal, accessible, and enjoyable
            user experience. We are committed to creating an environment that is
            inclusive and easy to navigate for everyone.
          </p>
        </div>
        <Separator />
      </div>
      <div className="mt-6 grid max-w-7xl grid-cols-1 gap-x-8 gap-y-10 px-4 md:grid-cols-12">
        <div className="md:col-span-4">
          <h2 className="text-foreground text-base font-semibold leading-7">
            Interface Theme
          </h2>
          <p className="text-muted-foreground mt-1 text-sm leading-6">
            Adjust the visual aspects of your interface to meet your individual
            needs, enhancing readability and overall accessibility.
          </p>
        </div>

        <div className="md:col-span-8">
          <ThemeSwitcher />
        </div>
      </div>
    </>
  );
}

function ColorBlindPreferences() {
  return (
    <div className="mt-6 grid max-w-7xl grid-cols-1 gap-x-8 gap-y-10 px-4 md:grid-cols-12">
      <div className="md:col-span-4">
        <h2 className="text-foreground text-base font-semibold leading-7">
          Color Accessibility Options
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          Optimize your visual experience with our color accessibility settings,
          tailored to accommodate different forms of color vision deficiency.
        </p>
      </div>
      <div className="md:col-span-8">
        <ColorBlindSwitcher />
      </div>
    </div>
  );
}

export default function UserPreferences() {
  return (
    <div className="mb-5">
      <Preferences />
      <ColorBlindPreferences />
    </div>
  );
}
