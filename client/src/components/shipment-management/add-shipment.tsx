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

import { BillingTab } from "@/components/shipment-management/add-shipment/billing-info-tab";
import { ShipmentGeneralForm } from "@/components/shipment-management/add-shipment/general-info-tab";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { debounce } from "lodash";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";

const tabs = {
  general: {
    name: "General",
    component: (props: any) => <ShipmentGeneralForm {...props} />,
  },
  billing: {
    name: "Billing",
    component: (props: any) => <BillingTab {...props} />,
  },
};

export default function AddShipment() {
  const [tab, setTab] = useState<string>("general");

  const handleTabClick = (tabName: string) => {
    setTab(tabName);
  };
  const { control, setValue, watch } = useForm();

  const tabProps = {
    control,
    setValue,
    watch,
  };

  const ActiveTabComponent = tabs[tab as keyof typeof tabs].component(tabProps);

  const [isScrolled, setIsScrolled] = useState(false);
  const scrollThreshold = 80;

  const handleScroll = useMemo(
    () =>
      debounce(() => {
        setIsScrolled(window.scrollY > scrollThreshold);
      }, 30),
    [scrollThreshold],
  );

  useEffect(() => {
    window.addEventListener("scroll", handleScroll);
    return () => {
      handleScroll.cancel(); // Ensure debounce is cancelled on unmount
      window.removeEventListener("scroll", handleScroll);
    };
  }, [handleScroll]);

  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[240px_minmax(0,1fr)] lg:gap-10">
      <aside
        className={`transition-spacing fixed top-14 z-30 -ml-2 hidden h-[calc(100vh-10rem)] w-full shrink-0 duration-500 md:sticky md:block ${
          isScrolled ? "pt-10" : ""
        }`}
      >
        <div className="bg-card text-card-foreground mt-4 rounded-lg border p-3">
          <nav className="lg:flex-col lg:space-y-2">
            {Object.entries(tabs).map(([tabKey, tabInfo]) => (
              <div key={tabKey} className="space-y-1">
                <div
                  onClick={() => handleTabClick(tabKey)}
                  className={cn(
                    buttonVariants({ variant: "ghost" }),
                    tab === tabKey
                      ? "bg-muted [&_svg]:text-foreground"
                      : "hover:bg-muted",
                    "group justify-start flex items-center mx-2",
                  )}
                >
                  {tabInfo.name}
                </div>
              </div>
            ))}
          </nav>
        </div>
      </aside>
      <div className="relative lg:gap-10">
        <div className="mx-auto min-w-0">
          <div className="bg-card border-border m-4 rounded-md border md:col-span-2">
            {ActiveTabComponent}
          </div>
        </div>
      </div>
    </div>
  );
}
