import { LazyComponent } from "@/components/error-boundary";
import { Icon } from "@/components/ui/icons";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from "@/components/ui/sidebar";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { checkSectionErrors } from "@/lib/form";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { cn } from "@/lib/utils";
import {
  faCreditCard,
  faEnvelope,
  faUser,
} from "@fortawesome/pro-regular-svg-icons";
import { lazy, useState } from "react";
import { type FieldValues, type Path, useFormContext } from "react-hook-form";

const GeneralInformationForm = lazy(
  () => import("./customer-general-information"),
);

const BillingProfileForm = lazy(() => import("./customer-billing-profile"));

function createNavigationItems<T extends FieldValues>() {
  return [
    {
      id: "general",
      name: "General Information",
      description: "Essential customer identification details.",
      icon: <Icon icon={faUser} />,
      component: <GeneralInformationForm />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [
          "status",
          "code",
          "name",
          "description",
          "addressLine1",
          "addressLine2",
          "city",
          "stateId",
          "postalCode",
        ] as Path<T>[]),
    },
    {
      id: "billing-profile",
      name: "Billing Profile",
      description: "Configure billing settings for the customer.",
      icon: <Icon icon={faCreditCard} />,
      component: <BillingProfileForm />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [] as Path<T>[]),
    },
    {
      id: "email-profile",
      name: "Email Profile",
      description: "Configure email settings for the customer.",
      icon: <Icon icon={faEnvelope} />,
      component: <div>Coming soon</div>,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [] as Path<T>[]),
    },
  ];
}

export function CustomerForm() {
  const {
    formState: { errors },
  } = useFormContext<CustomerSchema>();
  const [activeSection, setActiveSection] = useState("general");
  const navigationItems = createNavigationItems<CustomerSchema>();
  const activeComponent = navigationItems.find(
    (item) => item.id === activeSection,
  )?.component;

  return (
    <div className="flex size-full flex-1">
      <TooltipProvider>
        <SidebarProvider className="h-[750px] min-h-full w-56 shrink-0 items-start">
          <Sidebar
            collapsible="none"
            className="hidden w-56 rounded-tl-lg border-r border-input/50 md:flex"
          >
            <SidebarContent>
              <SidebarGroup>
                <SidebarGroupContent>
                  <SidebarMenu>
                    {navigationItems.map((item) => {
                      const hasError = item.validateSection(errors as any);

                      return (
                        <Tooltip delayDuration={400} key={item.id}>
                          <TooltipTrigger asChild>
                            <SidebarMenuItem key={item.id}>
                              <SidebarMenuButton
                                asChild
                                isActive={activeSection === item.id}
                                onClick={() => setActiveSection(item.id)}
                                className={cn(
                                  "hover:bg-transparent text-muted-foreground size-full gap-0.5",
                                  hasError && "hover:text-red-500 text-red-600",
                                )}
                              >
                                <div className="flex flex-col items-start">
                                  <div className="flex items-center gap-2">
                                    {item.icon}
                                    {item.name}
                                  </div>
                                  <div className="w-[190px] truncate text-2xs text-muted-foreground">
                                    {item.description}
                                  </div>
                                </div>
                              </SidebarMenuButton>
                            </SidebarMenuItem>
                          </TooltipTrigger>
                          <TooltipContent
                            side="right"
                            className="flex items-center gap-2 text-xs"
                          >
                            <p>{item.description}</p>
                          </TooltipContent>
                        </Tooltip>
                      );
                    })}
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            </SidebarContent>
          </Sidebar>
        </SidebarProvider>
      </TooltipProvider>

      <main className="flex size-full">
        <LazyComponent>{activeComponent}</LazyComponent>
      </main>
    </div>
  );
}
