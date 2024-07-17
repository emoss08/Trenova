/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
