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
import { Label } from "@/components/common/fields/label";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/components/ui/theme-provider";
import { type ThemeOptions } from "@/types";
import { useState } from "react";

export function ThemeSwitcher() {
  const { theme, setTheme, setIsRainbowAnimationActive } = useTheme();
  const [currentTheme, setCurrentTheme] = useState(theme);

  const switchTheme = (selectedTheme: ThemeOptions) => {
    // If the selected theme is the same as the current one, just return
    if (currentTheme === selectedTheme) {
      return;
    }

    // Now, set the current theme to the selected theme
    setCurrentTheme(selectedTheme);

    // Then, make necessary changes like showing toast and so on
    setTheme(selectedTheme);
  };
  return (
    <div className="space-y-1">
      <Label>Theme</Label>
      <p className="text-[0.8rem] text-muted-foreground">
        Select the theme you'd like to use.
      </p>
      <div className="grid max-w-xl grid-cols-3 gap-8 pt-2">
        <div className="space-y-2">
          <label className="border-primary text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
            <Button
              type="button"
              onClick={() => switchTheme("light")}
              className="sr-only aspect-square size-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-white p-2">
                <div className="space-y-2 rounded-md bg-black/10 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-black/50"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-black/50"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-black/10 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-black/50"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-black/50"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-black/10 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-black/50"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-black/50"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Light
            </span>
          </label>
        </div>
        <div className="space-y-2">
          <label className="border-primary text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
            <button
              type="button"
              onClick={() => switchTheme("dark")}
              className="sr-only aspect-square size-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            ></button>
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-zinc-950 p-2">
                <div className="space-y-2 rounded-md bg-zinc-800 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-zinc-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-zinc-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-zinc-800 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-zinc-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-zinc-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-zinc-800 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-zinc-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-zinc-400"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Dark
            </span>
          </label>
        </div>
      </div>
      <Label>Topbar</Label>
      <p className="text-[0.8rem] text-muted-foreground">
        Select the topbar you'd like to use.
      </p>
      <div className="grid max-w-xl grid-cols-3 gap-8 pt-2">
        <div className="space-y-2">
          <label className="border-primary text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
            <Button
              type="button"
              onClick={() => setIsRainbowAnimationActive(false)}
              className="sr-only aspect-square size-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="h-1 w-full rounded-md bg-rainbow-gradient-light bg-200%" />
                <div className="space-y-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Rainbow
            </span>
          </label>
        </div>

        <div className="space-y-2">
          <label className="border-primary text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
            <Button
              type="button"
              onClick={() => setIsRainbowAnimationActive(true)}
              className="sr-only aspect-square size-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="h-1 animate-rainbow-flow rounded-md bg-rainbow-gradient-light bg-200%" />
                <div className="space-y-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="size-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Rainbow Animated
            </span>
          </label>
        </div>
      </div>
    </div>
  );
}
