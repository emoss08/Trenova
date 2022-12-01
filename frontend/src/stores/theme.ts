import { ref } from "vue";
import { defineStore } from "pinia";
import { ThemeModeComponent } from "@/assets/ts/layout";

export const THEME_MODE_LS_KEY = "kt_theme_mode_value";
export const THEME_MENU_MODE_LS_KEY = "kt_theme_mode_menu";

export const useThemeStore = defineStore("theme", () => {
  const mode = ref<"light" | "dark" | "system">(
    localStorage.getItem(THEME_MODE_LS_KEY)
      ? (localStorage.getItem(THEME_MODE_LS_KEY) as "light" | "dark" | "system")
      : ThemeModeComponent.getSystemMode()
  );

  function setThemeMode(payload: "light" | "dark" | "system") {
    localStorage.setItem(THEME_MODE_LS_KEY, payload);
    localStorage.setItem(THEME_MENU_MODE_LS_KEY, payload);
    document.documentElement.setAttribute("data-theme", payload);
    ThemeModeComponent.init();
    mode.value = payload;
  }

  return {
    mode,
    setThemeMode,
  };
});
