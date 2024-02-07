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

import { Button, buttonVariants } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { cn } from "@/lib/utils";
import { shipmentSchema } from "@/lib/validations/ShipmentSchema";
import { useUserStore } from "@/stores/AuthStore";
import { ShipmentFormProps, ShipmentFormValues } from "@/types/order";
import {
  faBoxTaped,
  faCommentQuote,
  faFile,
  faMoneyBillTransfer,
  faOctagon,
  faWebhook,
} from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { yupResolver } from "@hookform/resolvers/yup";
import { debounce } from "lodash-es";
import { Suspense, lazy, useCallback, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Skeleton } from "../ui/skeleton";

type Tab = {
  name: string;
  component: React.ComponentType<ShipmentFormProps>;
  icon: JSX.Element;
  description: string;
};

// Lazy load the tabs
const GeneralTab = lazy(
  () =>
    import("@/components/shipment-management/add-shipment/general-info-tab"),
);
const BillingTab = lazy(
  () =>
    import("@/components/shipment-management/add-shipment/billing-info-tab"),
);

const tabs: Record<string, Tab> = {
  general: {
    name: "General Information",
    component: GeneralTab,
    icon: <FontAwesomeIcon icon={faBoxTaped} />,
    description: "General information about the shipment",
  },
  stops: {
    name: "Additional Stops",
    component: () => <div>Stops</div>,
    icon: <FontAwesomeIcon icon={faOctagon} />,
    description: "Stops for the shipment",
  },
  billing: {
    name: "Billing Information",
    component: BillingTab,
    icon: <FontAwesomeIcon icon={faMoneyBillTransfer} />,
    description: "Billing information for the shipment",
  },
  comments: {
    name: "Comments",
    component: () => <div>Comments</div>,
    icon: <FontAwesomeIcon icon={faCommentQuote} />,
    description: "Comments about the shipment",
  },
  documents: {
    name: "Documents",
    component: () => <div>Documents</div>,
    icon: <FontAwesomeIcon icon={faFile} />,
    description: "Documents for the shipment",
  },
  edi: {
    name: "EDI Information",
    component: () => <div>EDI Information</div>,
    icon: <FontAwesomeIcon icon={faWebhook} />,
    description: "EDI information for the shipment",
  },
};

export default function AddShipment() {
  const [activeTab, setActiveTab] = useState<string>("general");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [isScrolled, setIsScrolled] = useState(false);
  const [user] = useUserStore.use("user");

  const { control, setValue, watch, reset, handleSubmit } =
    useForm<ShipmentFormValues>({
      resolver: yupResolver(shipmentSchema),
      defaultValues: {
        status: "N",
        proNumber: "",
        originLocation: "",
        originAddress: "",
        destinationLocation: "",
        destinationAddress: "",
        entryMethod: "MANUAL",
        ratingUnits: 1,
        autoRate: false,
        enteredBy: user?.id,
      },
    });

  const handleTabClick = useCallback((tabName: string) => {
    setActiveTab(tabName);
  }, []);

  const mutation = useCustomMutation<ShipmentFormValues>(
    control,
    {
      method: "POST",
      path: "/shipment/",
      successMessage: "Shipment created successfully.",
      queryKeysToInvalidate: ["shipments"],
      closeModal: true,
      errorMessage: "Failed to create new shipment.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = useCallback(
    (values: ShipmentFormValues) => {
      setIsSubmitting(true);
      mutation.mutate(values);
    },
    [mutation],
  );

  const ActiveTabComponent = tabs[activeTab].component;

  const handleScroll = useCallback(
    debounce(() => {
      setIsScrolled(window.scrollY > 80);
    }, 100),
    [],
  );

  useEffect(() => {
    window.addEventListener("scroll", handleScroll);
    return () => {
      handleScroll.cancel();
      window.removeEventListener("scroll", handleScroll);
    };
  }, [handleScroll]);

  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[300px_minmax(0,1fr)] lg:gap-10">
      <aside
        className={cn(
          "transition-spacing fixed top-14 z-30 -ml-2 hidden h-[calc(100vh-10rem)] w-full shrink-0 duration-500 md:sticky md:block",
          isScrolled && "pt-10",
        )}
      >
        <div className="bg-card text-card-foreground rounded-lg border p-2">
          <nav className="lg:flex-col lg:space-y-2">
            {Object.entries(tabs).map(([tabKey, tabInfo]) => (
              <div key={tabKey} className="space-y-2">
                <div
                  onClick={() => handleTabClick(tabKey)}
                  className={cn(
                    buttonVariants({ variant: "ghost", size: "nosize" }),
                    activeTab === tabKey
                      ? "bg-muted [&_svg]:text-foreground"
                      : "hover:bg-muted",
                    "group flex flex-col items-start mx-2 my-1 p-2 text-wrap cursor-pointer select-none",
                  )}
                >
                  <div className="flex items-center space-x-2">
                    <span>{tabInfo.icon}</span>
                    <span>{tabInfo.name}</span>
                  </div>
                  <div className="text-muted-foreground text-xs">
                    {tabInfo.description}
                  </div>
                </div>
              </div>
            ))}
          </nav>
        </div>
      </aside>
      <div className="relative mb-10 lg:gap-10">
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex h-full flex-col overflow-y-auto lg:pr-[13rem]"
        >
          <Suspense fallback={<Skeleton className="h-[60vh] w-full" />}>
            <ActiveTabComponent
              control={control}
              setValue={setValue}
              watch={watch}
            />
          </Suspense>
          <div className="mt-4 flex flex-col-reverse pt-4 sm:flex-row sm:justify-end sm:space-x-2">
            <Button type="button" variant="outline">
              Save & Add Another
            </Button>
            <Button type="submit" isLoading={isSubmitting}>
              Save
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
