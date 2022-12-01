<template>
  <!--begin::Menu-->
  <div
    class="menu menu-sub menu-sub-dropdown menu-column menu-rounded menu-title-gray-700 menu-icon-muted menu-active-bg menu-state-primary fw-semibold py-4 fs-base w-175px"
    data-kt-menu="true"
    data-kt-element="theme-mode-menu"
  >
    <!--begin::Menu item-->
    <div class="menu-item px-3 my-0">
      <router-link
        :to="path"
        :class="{ active: themeMode === 'light' }"
        class="menu-link px-3 py-2"
        @click="setMode('light')"
      >
        <span class="menu-icon" data-kt-element="icon">
          <span class="svg-icon svg-icon-3">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen060.svg')"
            />
          </span>
        </span>
        <span class="menu-title">Light</span>
      </router-link>
    </div>
    <!--end::Menu item-->
    <!--begin::Menu item-->
    <div class="menu-item px-3 my-0">
      <router-link
        :to="path"
        :class="{ active: themeMode === 'dark' }"
        class="menu-link px-3 py-2"
        @click="setMode('dark')"
      >
        <span class="menu-icon" data-kt-element="icon">
          <span class="svg-icon svg-icon-3">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen061.svg')"
            />
          </span>
        </span>
        <span class="menu-title">Dark</span>
      </router-link>
    </div>
    <!--end::Menu item-->
    <!--begin::Menu item-->
    <div class="menu-item px-3 my-0">
      <router-link
        :to="path"
        :class="{ active: themeMode === 'system' }"
        class="menu-link px-3 py-2"
        @click="setMode('system')"
      >
        <span class="menu-icon" data-kt-element="icon">
          <span class="svg-icon svg-icon-3">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen062.svg')"
            />
          </span>
        </span>
        <span class="menu-title">System</span>
      </router-link>
    </div>
    <!--end::Menu item-->
  </div>
  <!--end::Menu-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, computed } from "vue";
import { useThemeStore } from "@/stores/theme";
import { useConfigStore } from "@/stores/config";
import { useRoute } from "vue-router";
import { ThemeModeComponent } from "@/assets/ts/layout";

export default defineComponent({
  name: "kt-theme-switcher",
  component: {},
  setup() {
    const storeTheme = useThemeStore();
    const storeConfig = useConfigStore();
    const route = useRoute();

    const themeMode = computed(() => {
      return storeTheme.mode;
    });

    const path = computed(() => route.path);

    const setMode = (mode: "dark" | "light" | "system") => {
      let configMode = mode;
      if (mode === "system") {
        configMode = ThemeModeComponent.getSystemMode();
      }

      storeConfig.setLayoutConfigProperty("general.mode", configMode);

      storeTheme.setThemeMode(configMode);
    };

    return {
      themeMode,
      setMode,
      path,
      getAssetPath,
    };
  },
});
</script>
