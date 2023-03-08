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

import { createContext, FC, useContext, useEffect, useState } from 'react';
import { DefaultLayoutConfig } from '@/utils/layout/DefaultLayoutConfig';
import {
  getEmptyCssClasses,
  getEmptyCSSVariables,
  getEmptyHTMLAttributes,
  LayoutSetup,
} from '@/utils/layout/LayoutSetup'
import {
  ILayout,
  ILayoutCSSVariables,
  ILayoutCSSClasses,
  ILayoutHTMLAttributes,
} from "@/models/layout";
import { WithChildren } from '../types';

export interface LayoutContextModel {
  config: ILayout;
  classes: ILayoutCSSClasses;
  attributes: ILayoutHTMLAttributes;
  cssVariables: ILayoutCSSVariables;
  setLayout: (config: LayoutSetup) => void;
}

const LayoutContext = createContext<LayoutContextModel>({
  config: DefaultLayoutConfig,
  classes: getEmptyCssClasses(),
  attributes: getEmptyHTMLAttributes(),
  cssVariables: getEmptyCSSVariables(),
  setLayout: (config: LayoutSetup) => {},
});

export const LayoutProvider: FC<WithChildren> = ({ children }) => {
  const [config, setConfig] = useState(LayoutSetup.config);
  const [classes, setClasses] = useState(LayoutSetup.classes);
  const [attributes, setAttributes] = useState(LayoutSetup.attributes);
  const [cssVariables, setCSSVariables] = useState(LayoutSetup.cssVariables);

  const setLayout = (_themeConfig: Partial<ILayout>) => {
    const bodyClasses = Array.from(document.body.classList);
    bodyClasses.forEach((cl) => document.body.classList.remove(cl));
    LayoutSetup.updatePartialConfig(_themeConfig);
    setConfig(Object.assign({}, LayoutSetup.config));
    setClasses(LayoutSetup.classes);
    setAttributes(LayoutSetup.attributes);
    setCSSVariables(LayoutSetup.cssVariables);
  };

  const value: LayoutContextModel = {
    config,
    classes,
    attributes,
    cssVariables,
    setLayout,
  };

  useEffect(() => {
    const disableSplashScreen = () => {
      const splashScreen = document.getElementById('splash-screen');
      if (splashScreen) {
        splashScreen.style.setProperty('display', 'none');
      }
    };

    disableSplashScreen();
  }, []);

  return <LayoutContext.Provider value={value}>{children}</LayoutContext.Provider>;
};

export const useLayout = () => useContext(LayoutContext);