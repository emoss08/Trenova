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

import { Tab } from "bootstrap";
import { useEffect, useRef } from "react";

import { useLayout } from "@/utils/layout/LayoutProvider";
import {
  DrawerComponent,
  MenuComponent,
  ScrollComponent,
  ScrollTopComponent,
  StickyComponent,
  SwapperComponent,
  ToggleComponent
} from "./assets/ts/components";

export function MasterInit() {
  const { config } = useLayout();
  const isFirstRun = useRef(true);

  useEffect(() => {
    const pluginsInitialization = () => {
      isFirstRun.current = false;
      setTimeout(() => {
        if (typeof document !== "undefined") {
          ToggleComponent.bootstrap();
          ScrollTopComponent.bootstrap();
          DrawerComponent.bootstrap();
          StickyComponent.bootstrap();
          MenuComponent.bootstrap();
          ScrollComponent.bootstrap();
          SwapperComponent.bootstrap();
          if (typeof document !== "undefined") {
            document.querySelectorAll("[data-bs-toggle=\"tab\"]").forEach((tab) => {
              window.bootstrap.Tab.getOrCreateInstance(tab);
            });
          }
        }
      }, 1000);
    };

    if (isFirstRun.current) {
      isFirstRun.current = false;
      pluginsInitialization();
    }
  }, [config]);

  return <></>;
}


