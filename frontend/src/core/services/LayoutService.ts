import objectPath from "object-path";
import { config } from "@/core/helpers/config";
import { useBodyStore } from "@/stores/body";
import { useConfigStore } from "@/stores/config";

class LayoutService {
  public static bodyStore: any;
  public static configStore: any;

  /**
   * @description initialize default layout
   */
  public static init(): void {
    this.bodyStore = useBodyStore();
    this.configStore = useConfigStore();

    //empty body element classes and attributes
    LayoutService.emptyElementClassesAndAttributes(document.body);

    LayoutService.initLayout();
    LayoutService.initHeader();
    LayoutService.initToolbar();
    LayoutService.initAside();
    LayoutService.initFooter();
  }

  /**
   * @description init layout
   */
  public static initLayout(): void {
    this.bodyStore.addBodyAttribute({
      qualifiedName: "id",
      value: "kt_body",
    });
  }

  /**
   * @description init header
   */
  public static initHeader(): void {
    if (objectPath.get(config.value, "header.fixed.desktop")) {
      this.bodyStore.addBodyClassname("header-fixed");
    }

    if (objectPath.get(config.value, "header.fixed.tabletAndMobile")) {
      this.bodyStore.addBodyClassname("header-tablet-and-mobile-fixed");
    }
  }

  /**
   * @description init toolbar
   */
  public static initToolbar(): void {
    if (!objectPath.get(config.value, "toolbar.display")) {
      return;
    }

    this.bodyStore.addBodyClassname("toolbar-enabled");

    if (objectPath.get(config.value, "toolbar.fixed")) {
      this.bodyStore.addBodyClassname("toolbar-fixed");
    }

    this.bodyStore.addBodyClassname("toolbar-tablet-and-mobile-fixed");
  }

  /**
   * @description init aside
   */
  public static initAside(): void {
    if (!objectPath.get(config.value, "aside.display")) {
      return;
    }

    // Enable Aside
    this.bodyStore.addBodyClassname("aside-enabled");

    // Minimized
    if (
      objectPath.get(config.value, "aside.minimized") &&
      objectPath.get(config.value, "aside.toggle")
    ) {
      this.bodyStore.addBodyAttribute({
        qualifiedName: "data-kt-aside-minimize",
        value: "on",
      });
    }

    if (objectPath.get(config.value, "aside.fixed")) {
      // Fixed Aside
      this.bodyStore.addBodyClassname("aside-fixed");
    }

    // Default minimized
    if (objectPath.get(config.value, "aside.minimized")) {
      this.bodyStore.addBodyAttribute({
        qualifiedName: "data-kt-aside-minimize",
        value: "on",
      });
    }
  }

  /**
   * @description init footer
   */
  public static initFooter(): void {
    // Fixed header
    if (objectPath.get(config.value, "footer.width") === "fixed") {
      this.bodyStore.addBodyClassname("footer-fixed");
    }
  }

  public static emptyElementClassesAndAttributes(element: HTMLElement): void {
    element.className = "";
    for (let i = element.attributes.length; i-- > 0; )
      element.removeAttributeNode(element.attributes[i]);
  }
}

export default LayoutService;
