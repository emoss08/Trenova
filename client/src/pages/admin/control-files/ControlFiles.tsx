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

import React, { Suspense, useEffect, useState } from "react";
import { Grid, Skeleton } from "@mantine/core";
import { useLocation, useNavigate } from "react-router-dom";
import { NavBar } from "@/components/ui/NavBar";
import { controlFileData } from "@/utils/apps/admin";

function ControlFiles() {
  const location = useLocation();
  const navigate = useNavigate();
  const initialTabLabel = location.hash.substring(1).replace(/-/g, " ");
  const initialTab =
    controlFileData.find(
      (tab) => tab.label.toLowerCase() === initialTabLabel,
    ) || controlFileData[0];

  const [activeTab, setActiveTab] = useState(initialTab);

  // Update activeTab when location.hash changes
  useEffect(() => {
    const newTabLabel = location.hash.substring(1).replace(/-/g, " ");
    const newTab = controlFileData.find(
      (tab) => tab.label.toLowerCase() === newTabLabel,
    );

    if (newTab) {
      setActiveTab(newTab);
    }
  }, [location.hash]);

  const ActiveComponent = activeTab.component;

  return (
    <Grid gutter="md">
      <Grid.Col span={12} sm={6} md={4} lg={3} xl={3}>
        <Suspense fallback={<Skeleton height={400} />}>
          <NavBar
            data={controlFileData}
            setActiveTab={setActiveTab}
            activeTab={activeTab}
            navigate={navigate}
          />
        </Suspense>
      </Grid.Col>
      <Grid.Col span={12} sm={6} md={8} lg={9} xl={9}>
        <Suspense fallback={<Skeleton height={500} />}>
          <ActiveComponent />
        </Suspense>
      </Grid.Col>
    </Grid>
  );
}

export default ControlFiles;
