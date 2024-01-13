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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { useHeaderStore } from "@/stores/HeaderStore";
import { MenuIcon } from "lucide-react";
import { Button } from "../ui/button";

export function AsideMenuSheet() {
  const [open, setMenuOpen] = useHeaderStore.use("asideMenuOpen");

  return (
    <Sheet open={open} onOpenChange={setMenuOpen}>
      <SheetTrigger asChild>
        <Button
          size="icon"
          variant="outline"
          className="h-9 w-9 border-muted-foreground/40 hover:border-muted-foreground/80"
        >
          <MenuIcon className="h-5 w-5" />
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[400px] sm:w-[540px]" side="left">
        <SheetHeader>
          <SheetTitle>Add New Customer</SheetTitle>
          <SheetDescription>
            Use this form to add a new customer to the system.
          </SheetDescription>
        </SheetHeader>
        Test
      </SheetContent>
    </Sheet>
  );
}
