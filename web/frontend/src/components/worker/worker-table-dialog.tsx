/**
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

import { LinkGroupProps } from "@/types/sidebar-nav";
import { Suspense, lazy, useState } from "react";
import { ComponentLoader } from "../ui/component-loader";
import { ModalAsideMenu } from "../user-settings/sidebar-nav";

const PersonalInformation = lazy(
  () => import("./worker-personal-information-form"),
);
const ComplianceInformation = lazy(() => import("./worker-compliance-form"));

export function WorkerForm() {
  const [activeTab, setActiveTab] = useState("personal-information");

  const linkGroups: LinkGroupProps[] = [
    {
      title: "Worker Profile",
      links: [
        {
          key: "personal-information",
          href: "#personal-information",
          title: "Personal Information",
          component: <PersonalInformation />,
        },
        {
          key: "additional-information",
          href: "#additional-information",
          title: "Additional Information",
          component: <ComplianceInformation />,
        },
        {
          key: "employment-history",
          href: "#employment-history",
          title: "Employment History",
          component: <div>Coming soon</div>,
        },
      ],
    },
    {
      title: "Qualification and Certifications",
      links: [
        {
          key: "medical-exam",
          href: "#medical-exam",
          title: "Medical Examinations",
          component: <div>Coming soon</div>,
        },
        {
          key: "road-test",
          href: "#road-test",
          title: "Road Test Certifications",
          component: <div>Coming soon</div>,
        },
      ],
    },
    {
      title: "Driving Records",
      links: [
        {
          key: "driving-record",
          href: "#driving-record",
          title: "Driving Record (MVR)",
          component: <div>Coming soon</div>,
        },
        {
          key: "violation-accident-records",
          href: "#violation-accident-records",
          title: "Violation and Accident Records",
          component: <div>Coming soon</div>,
        },
      ],
    },
    {
      title: "Compliance and Testing",
      links: [
        {
          key: "drug-alcohol-tests",
          href: "#drug-alcohol-tests",
          title: "Drug and Alcohol Testing Records",
          component: <div>Coming soon</div>,
        },

        {
          key: "hos",
          href: "#hos",
          title: "Hours of Service (HOS)",
          component: <div>Coming soon</div>,
        },
      ],
    },
    {
      title: "Additional Documents",
      links: [
        {
          key: "worker-documents",
          href: "#worker-documents",
          title: "Miscellaneous Documents",
          component: <div>Coming soon</div>,
        },
      ],
    },
  ];

  const activeComponent = linkGroups
    .flatMap((group) => group.links)
    .find((link) => link.key === activeTab)?.component;

  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[270px_minmax(0,1fr)] lg:gap-10">
      <ModalAsideMenu
        linkGroups={linkGroups}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
      />
      <main className="relative lg:gap-10">
        <div className="mx-auto min-w-0">
          <Suspense fallback={<ComponentLoader className="h-[30vh]" />}>
            {activeComponent}
          </Suspense>
        </div>
      </main>
    </div>
  );
}
