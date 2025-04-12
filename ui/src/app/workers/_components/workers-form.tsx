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
import { checkSectionErrors } from "@/lib/form";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import {
  faBriefcase,
  faCalendar,
  faIdCard,
  faUser,
} from "@fortawesome/pro-regular-svg-icons";
import { lazy, Suspense, useState } from "react";
import { FieldValues, Path, useFormContext } from "react-hook-form";

const PersonalInformationForm = lazy(
  () => import("./workers-personal-information-form"),
);
const EmploymentDetailsForm = lazy(
  () => import("./workers-employment-details-form"),
);
const LicenseInformationForm = lazy(
  () => import("./workers-license-information-form"),
);
const WorkerPTOForm = lazy(() => import("./workers-pto-form"));

function createNavigationItems<T extends FieldValues>() {
  return [
    {
      id: "personal",
      name: "Personal Information",
      icon: <Icon icon={faUser} />,
      component: <PersonalInformationForm />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [
          "firstName",
          "lastName",
          "addressLine1",
          "addressLine2",
          "city",
          "stateId",
          "postalCode",
        ] as Path<T>[]),
    },
    {
      id: "employment",
      name: "Employment Details",
      icon: <Icon icon={faBriefcase} />,
      component: <EmploymentDetailsForm />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [
          "status",
          "type",
          "gender",
          "profile.dob",
          "profile.hireDate",
          "profile.terminationDate",
        ] as Path<T>[]),
    },
    {
      id: "license",
      name: "License Information",
      icon: <Icon icon={faIdCard} />,
      component: <LicenseInformationForm />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [
          "profile.licenseNumber",
          "profile.licenseStateId",
          "profile.licenseExpiry",
          "profile.hazmatExpiry",
          "profile.lastMvrCheck",
          "profile.lastDrugTest",
        ] as Path<T>[]),
    },
    {
      id: "pto",
      name: "PTO Management",
      icon: <Icon icon={faCalendar} />,
      component: <WorkerPTOForm />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [] as Path<T>[]),
    },
  ];
}

export function WorkerForm() {
  const {
    formState: { errors },
  } = useFormContext<WorkerSchema>();
  const [activeSection, setActiveSection] = useState("personal");
  const navigationItems = createNavigationItems<WorkerSchema>();
  const activeComponent = navigationItems.find(
    (item) => item.id === activeSection,
  )?.component;

  return (
    <div className="flex size-full flex-1">
      <SidebarProvider className="h-[550px] min-h-full w-48 shrink-0 items-start">
        <Sidebar
          collapsible="none"
          className="hidden w-52 rounded-tl-lg border-r border-input/50 md:flex"
        >
          <SidebarContent>
            <SidebarGroup>
              <SidebarGroupContent>
                <SidebarMenu>
                  {navigationItems.map((item) => {
                    const hasError = item.validateSection(errors as any);

                    return (
                      <SidebarMenuItem key={item.id}>
                        <SidebarMenuButton
                          asChild
                          isActive={activeSection === item.id}
                          onClick={() => setActiveSection(item.id)}
                          className={cn(
                            "hover:bg-transparent text-muted-foreground",
                            hasError && "hover:text-red-500 text-red-600",
                          )}
                        >
                          <span>
                            {item.icon}
                            {item.name}
                          </span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </SidebarContent>
        </Sidebar>
      </SidebarProvider>

      <main className="flex size-full">
        <Suspense fallback={<div>Loading...</div>}>{activeComponent}</Suspense>
      </main>
    </div>
  );
}
