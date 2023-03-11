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

import {
  IAside,
  IContent,
  IFooter,
  IHeader,
  IToolbar,
  ILayout,
  ILayoutCSSClasses,
  ILayoutHTMLAttributes,
  ILayoutCSSVariables
} from "@/models/layout";

import { DefaultLayoutConfig } from "./DefaultLayoutConfig";
import { getFromLocalStorage, setInLocalStorage } from "@/utils/helpers/LocalStorageHelper";

const LAYOUT_CONFIG_KEY = process.env.NEXT_PUBLIC_BASE_LAYOUT_CONFIG_KEY || "LayoutConfig";

export function getLayout(): ILayout {
  let ls = null;
  try {
    ls = getFromLocalStorage(LAYOUT_CONFIG_KEY)
  } catch (error) {
    console.error("Error getting layout configuration from localStorage: ", error);
  }
  if (ls) {
    try {
      return JSON.parse(ls) as ILayout;
    } catch (er) {
      console.error("Error parsing layout configuration from localStorage: ", er);
    }
  }
  return DefaultLayoutConfig;
}

function setLayout(config: ILayout): void {
  try {
    setInLocalStorage(LAYOUT_CONFIG_KEY, JSON.stringify(config));
  } catch (er) {
    console.error(er);
  }
}

export function getEmptyCssClasses() {
  return {
    header: [],
    headerContainer: [],
    headerMobile: [],
    headerMenu: [],
    aside: [],
    asideMenu: [],
    asideToggle: [],
    toolbar: [],
    toolbarContainer: [],
    content: [],
    contentContainer: [],
    footerContainer: [],
    sidebar: [],
    pageTitle: []
  };
}

export function getEmptyHTMLAttributes() {
  return {
    asideMenu: new Map(),
    headerMobile: new Map(),
    headerMenu: new Map(),
    headerContainer: new Map(),
    pageTitle: new Map()
  };
}

export function getEmptyCSSVariables() {
  return {
    body: new Map()
  };
}

export class LayoutSetup {
  public static isLoaded: boolean = false;
  public static config: ILayout = getLayout();
  public static classes: ILayoutCSSClasses = getEmptyCssClasses();
  public static attributes: ILayoutHTMLAttributes = getEmptyHTMLAttributes();
  public static cssVariables: ILayoutCSSVariables = getEmptyCSSVariables();

  private static initCSSClasses(): void {
    LayoutSetup.classes = getEmptyCssClasses();
  }

  private static initHTMLAttributes(): void {
    LayoutSetup.attributes = Object.assign({}, getEmptyHTMLAttributes());
  }

  private static initCSSVariables(): void {
    LayoutSetup.cssVariables = getEmptyCSSVariables();
  }

  private static initLayout(config: ILayout): void {
    if (typeof window !== "undefined") {
      Array.from(document.body.attributes).forEach((attr) => {
        document.body.removeAttribute(attr.name);
      });
      document.body.setAttribute("style", "");
      document.body.setAttribute("id", "mt_body");
      if (config.main?.body?.backgroundImage) {
        document.body.style.backgroundImage = `url('${config.main.body.backgroundImage}')`;
      }

      if (config.main?.body?.class) {
        document.body.classList.add(config.main.body.class);
      }
    }
    // if (config.loader.display) {
    //   document.body.classList.add('page-loading')
    //   document.body.classList.add('page-loading-enabled')
    // }
  }

  private static initHeader(config: IHeader): void {
    LayoutSetup.classes.headerContainer.push(
      config.width === "fluid" ? "container-fluid" : "container"
    );

    if (config.fixed.tabletAndMobile) {
      document.body.classList.add("header-tablet-and-mobile-fixed");
    }
  }

  private static initToolbar(config: IToolbar): void {
    if (!config.display) {
      return;
    }

    document.body.classList.add("toolbar-enabled");
    LayoutSetup.classes.toolbarContainer.push(
      config.width === "fluid" ? "container-fluid" : "container"
    );

    if (config.fixed.desktop) {
      document.body.classList.add("toolbar-fixed");
    }

    if (config.fixed.tabletAndMobileMode) {
      document.body.classList.add("toolbar-tablet-and-mobile-fixed");
    }

    // Height setup
    const type = config.layout;
    const typeOptions = config.layouts[type];
    if (typeOptions) {
      let bodyStyles: string = "";
      if (typeOptions.height) {
        bodyStyles += ` --bs-toolbar-height: ${typeOptions.height};`;
      }

      if (typeOptions.heightAndTabletMobileMode) {
        bodyStyles += ` --bs-toolbar-height-tablet-and-mobile: ${typeOptions.heightAndTabletMobileMode};`;
      }
      document.body.setAttribute("style", bodyStyles);
    }
  }

  private static initContent(config: IContent): void {
    LayoutSetup.classes.contentContainer.push(
      config.width === "fluid" ? "container-fluid" : "container"
    );
  }

  private static initAside(config: IAside): void {
    // Enable Aside
    document.body.classList.add("aside-enabled");
    // Fixed Aside
    if (config.fixed) {
      document.body.classList.add("aside-fixed");
    }

    // Default minimized
    if (config.minimized) {
      document.body.setAttribute("data-mt-aside-minimize", "on");
    }

    // Hoverable on minimize
    if (config.hoverable) {
      LayoutSetup.classes.aside.push("aside-hoverable");
    }
  }
  private static initFooter(config: IFooter): void {
    LayoutSetup.classes.footerContainer.push(`container${config.width === "fluid" ? "-fluid" : ""}`);
  }

  private static initConfig(config: ILayout): void {
    if (config.main?.darkSkinEnabled) {
      if (typeof window !== 'undefined') {
        document.body.classList.add('dark-skin');
      }
    }

    if (typeof window !== 'undefined') {
      LayoutSetup.initLayout(config);

      if (config.main?.type === 'default') {
        LayoutSetup.initHeader(config.header);
        LayoutSetup.initContent(config.content);
        LayoutSetup.initAside(config.aside);
        LayoutSetup.initFooter(config.footer);
      }
    }
  }


  public static updatePartialConfig(fieldsToUpdate: Partial<ILayout>): ILayout {
    const config = LayoutSetup.config;
    const updatedConfig = { ...config, ...fieldsToUpdate };
    LayoutSetup.initCSSClasses();
    LayoutSetup.initCSSVariables();
    LayoutSetup.initHTMLAttributes();
    LayoutSetup.isLoaded = false;
    LayoutSetup.config = updatedConfig;
    LayoutSetup.initConfig(Object.assign({}, updatedConfig));
    LayoutSetup.isLoaded = true; // remove loading there
    return updatedConfig;
  }

  public static setConfig(config: ILayout): void {
    setLayout(config);
  }

  public static bootstrap = (() => {
    LayoutSetup.updatePartialConfig(LayoutSetup.config);
  })();
}
