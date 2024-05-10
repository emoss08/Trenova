import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { cleanObject } from "@/lib/utils";
import { ShipmentFormValues, ShipmentPageTab } from "@/types/shipment";
import {
  faBoxTaped,
  faCommentQuote,
  faFile,
  faMoneyBillTransfer,
  faOctagon,
  faWebhook,
} from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Suspense, lazy, useState } from "react";
import { FormProvider } from "react-hook-form";

import { useShipmentForm } from "@/lib/validations/ShipmentSchema";
import { useUserStore } from "@/stores/AuthStore";
import { ComponentLoader } from "../ui/component-loader";
import { ShipmentAsideMenu } from "./add-shipment/nav/nav-bar";

// Lazy load the tabs
const GeneralTab = lazy(
  () =>
    import("@/components/shipment-management/add-shipment/general-info-tab"),
);
const BillingTab = lazy(
  () =>
    import("@/components/shipment-management/add-shipment/billing-info-tab"),
);
const StopsTab = lazy(
  () => import("@/components/shipment-management/add-shipment/stop-info-tab"),
);

const tabs: Record<string, ShipmentPageTab> = {
  general: {
    name: "General Information",
    component: GeneralTab,
    icon: <FontAwesomeIcon icon={faBoxTaped} />,
    description: "General information about the shipment",
  },
  stops: {
    name: "Additional Stops",
    component: StopsTab,
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
  const [user] = useUserStore.use("user");

  const { shipmentForm, isShipmentControlLoading, shipmentControlData } =
    useShipmentForm({ user });

  if (isShipmentControlLoading && !shipmentControlData && !user) {
    return <ComponentLoader className="h-[60vh]" />;
  }

  const { control, handleSubmit } = shipmentForm;

  // TODO(WOLFRED): use react-hook-form-persist to persist form data in session storage until form is submitted

  // Mutation
  const mutation = useCustomMutation<ShipmentFormValues>(control, {
    method: "POST",
    path: "/shipments/",
    successMessage: "Shipment created successfully.",
    queryKeysToInvalidate: "shipments",
    closeModal: true,
    errorMessage: "Failed to create new shipment.",
  });

  const onSubmit = (values: ShipmentFormValues) => {
    const cleanedValues = cleanObject(values);
    mutation.mutate(cleanedValues);
  };

  const ActiveTabComponent = tabs[activeTab].component;

  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[300px_minmax(0,1fr)] lg:gap-10">
      <ShipmentAsideMenu
        tabs={tabs}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
      />
      <div className="relative mb-10 lg:gap-10">
        <FormProvider {...shipmentForm}>
          <form
            onSubmit={handleSubmit(onSubmit)}
            className="flex h-full flex-col overflow-y-visible"
          >
            <Suspense fallback={<ComponentLoader />}>
              <ActiveTabComponent />
            </Suspense>
            <div className="mt-4 flex flex-col-reverse pt-4 sm:flex-row sm:justify-end sm:space-x-2">
              <Button disabled type="button" variant="outline">
                Save & Add Another
              </Button>
              <Button type="submit" isLoading={mutation.isPending}>
                Save
              </Button>
            </div>
          </form>
        </FormProvider>
      </div>
    </div>
  );
}
