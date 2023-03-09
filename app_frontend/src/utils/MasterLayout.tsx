/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import { useRouter } from "next/router";
import { FC, PropsWithChildren, useEffect } from "react";
import { MenuComponent } from "./assets/ts/components";
import { PageDataProvider } from "@/utils/layout/PageData";
import { AsideDefault } from "@/components/layout/aside/AsideDefault";
import { HeaderWrapper } from "@/components/layout/header/HeaderWrapper";
import { Content } from "@/components/layout/Content";
import { Footer } from "@/components/layout/Footer";
import { ScrollTop } from "@/components/layout/ScrollTop";

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
    </PageDataProvider>
  );
};

export { MasterLayout };