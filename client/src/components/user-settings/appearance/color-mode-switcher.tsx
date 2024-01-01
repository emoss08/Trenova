/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

export function ColorBlindSwitcher() {
  return (
    <div className="space-y-1">
      <Label>Color Blind</Label>
      <p className="text-[0.8rem] text-muted-foreground">
        Select a color blind mode to enhance readability.
      </p>
      <div className="grid max-w-xl grid-cols-3 gap-8 pt-2">
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <Button
              type="button"
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="space-y-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-orange-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-orange-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-orange-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-orange-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-orange-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-orange-400"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Deuteranomaly
            </span>
          </label>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <Button
              type="button"
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="space-y-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-red-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-red-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-red-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-red-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-red-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-red-400"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Protanomaly
            </span>
          </label>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <Button
              type="button"
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="space-y-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-yellow-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-yellow-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-yellow-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-yellow-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-yellow-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-yellow-400"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Tritanopia
            </span>
          </label>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <Button
              type="button"
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="space-y-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-green-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-green-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-green-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-green-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-green-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-green-400"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Protanopia & Deuteranopia
            </span>
          </label>
        </div>
      </div>
    </div>
  );
}
