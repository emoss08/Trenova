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

import { RateTable } from "@/components/rate-management/rate-table";
import { ComponentLoader } from "@/components/ui/component-loader";

import { Suspense, lazy } from "react";

const TotalActiveRateCard = lazy(
  () =>
    import("../../components/rate-management/cards/total-active-rates-card"),
);

export default function RateManagement() {
  return (
    <>
      <Suspense fallback={<ComponentLoader />}>
        <div className="mb-10 grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
          <TotalActiveRateCard />
          <TotalActiveRateCard />
          <TotalActiveRateCard />
        </div>
      </Suspense>
      <RateTable />
    </>
  );
}
