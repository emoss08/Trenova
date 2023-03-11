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

import { useRouter } from "next/router";
import React, { FC, PropsWithChildren, useEffect } from "react";
import { MenuComponent } from "./assets/ts/components";
import { PageDataProvider } from "@/utils/layout/PageData";
import { AsideDefault } from "@/components/layout/aside/AsideDefault";
import { HeaderWrapper } from "@/components/layout/header/HeaderWrapper";
import { Content } from "@/components/layout/Content";
import { Footer } from "@/components/layout/Footer";
import { ScrollTop } from "@/components/layout/ScrollTop";
import KeepAliveConnection from "@/utils/components/KeepAliveConnection";

const MasterLayout: FC<PropsWithChildren> = ({ children }) => {
  const router = useRouter();
  useEffect(() => {
    setTimeout(() => {
      MenuComponent.reinitialization();
    }, 500);
  }, []);

  useEffect(() => {
    setTimeout(() => {
      MenuComponent.reinitialization();
    }, 500);
  }, [router.pathname]);

  return (
    <PageDataProvider>
      <div className="page d-flex flex-row flex-column-fluid">
        <AsideDefault />
        <div className="wrapper d-flex flex-column flex-row-fluid" id="mt_wrapper">
          <HeaderWrapper />

          <div id="mt_content" className="content d-flex flex-column flex-column-fluid">
            <div className="post d-flex flex-column-fluid" id="mt_post">
              <Content>
                {/* Replace <Outlet /> with the children prop */}
                {children}
              </Content>
            </div>
          </div>
          <Footer />
        </div>
      </div>

      {/* begin:: Drawers */}
      {/*<ActivityDrawer />*/}
      {/*<RightToolbar />*/}
      {/*<DrawerMessenger />*/}
      {/* end:: Drawers */}

      {/* begin:: Modals */}
      {/*<InviteUsers />*/}
      {/*<UpgradePlan />*/}
      {/* end:: Modals */}
      <ScrollTop />
      <KeepAliveConnection />
    </PageDataProvider>
  );
};

export { MasterLayout };