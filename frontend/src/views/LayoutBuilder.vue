<template>
  <!--begin::Layout Builder Notice-->
  <div class="card mb-10">
    <div class="card-body d-flex align-items-center py-8">
      <!--begin::Icon-->
      <div
        class="d-flex h-80px w-80px flex-shrink-0 flex-center position-relative"
      >
        <span class="svg-icon svg-icon-primary position-absolute opacity-15">
          <inline-svg
            :src="getAssetPath('/media/icons/duotune/abstract/abs051.svg')"
            class="h-80px w-80px"
          />
        </span>
        <span class="svg-icon svg-icon-3x svg-icon-primary position-absolute">
          <inline-svg
            :src="getAssetPath('/media/icons/duotune/coding/cod009.svg')"
          />
        </span>
      </div>
      <!--end::Icon-->

      <!--begin::Description-->
      <div class="ms-6">
        <p class="list-unstyled text-gray-600 fw-semobold fs-6 p-0 m-0">
          The layout builder is to assist your set and configure your preferred
          project layout specifications and preview it in real time and export
          the HTML template with its includable partials of this demo. The
          downloaded version does not include the assets folder since the layout
          builder's main purpose is to help to generate the final HTML code
          without hassle.
        </p>
      </div>
      <!--end::Description-->
    </div>
  </div>
  <!--end::Layout Builder Notice-->

  <div class="card mb-10">
    <!--begin::Header-->
    <div class="card-header card-header-stretch">
      <ul
        class="nav nav-stretch nav-line-tabs fw-semobold border-0"
        role="tablist"
        id="kt_layout_builder_tabs"
        ref="kt_layout_builder_tabs"
      >
        <li class="nav-item">
          <a
            class="nav-link cursor-pointer"
            :class="{ active: tabIndex === 0 }"
            data-bs-toggle="tab"
            @click="setActiveTab($event)"
            data-tab-index="0"
            role="tab"
          >
            Main
          </a>
        </li>
        <li class="nav-item">
          <a
            class="nav-link cursor-pointer"
            :class="{ active: tabIndex === 1 }"
            data-bs-toggle="tab"
            @click="setActiveTab($event)"
            data-tab-index="1"
            role="tab"
          >
            Header
          </a>
        </li>
        <li class="nav-item">
          <a
            class="nav-link cursor-pointer"
            :class="{ active: tabIndex === 3 }"
            data-bs-toggle="tab"
            @click="setActiveTab($event)"
            data-tab-index="3"
            role="tab"
          >
            Aside
          </a>
        </li>
        <li class="nav-item">
          <a
            class="nav-link cursor-pointer"
            :class="{ active: tabIndex === 4 }"
            data-bs-toggle="tab"
            @click="setActiveTab($event)"
            data-tab-index="4"
            role="tab"
          >
            Content
          </a>
        </li>
        <li class="nav-item">
          <a
            class="nav-link cursor-pointer"
            :class="{ active: tabIndex === 5 }"
            data-bs-toggle="tab"
            @click="setActiveTab($event)"
            data-tab-index="5"
            role="tab"
          >
            Footer
          </a>
        </li>
      </ul>
    </div>
    <!--end::Header-->

    <!--begin::Form-->
    <form class="form" id="kt_layout_builder_form" @submit="submit($event)">
      <!--begin::Body-->
      <div class="card-body">
        <div class="tab-content pt-3" id="kt_tabs">
          <div
            class="tab-pane"
            :class="{ active: tabIndex === 0 }"
            id="kt_builder_main"
          >
            <div class="row mb-10">
              <label class="col-lg-3 col-form-label text-lg-end"
                >Illustration Set:</label
              >
              <div class="col-lg-8 pe-10">
                <div class="row g-9 mb-3">
                  <div class="col-6">
                    <label
                      class="btn btn-outline btn-outline-dashed btn-outline-default d-flex flex-column text-start p-6"
                      for="sketchy-1"
                    >
                      <span class="d-flex mb-2">
                        <span
                          class="form-check form-check-custom form-check-solid form-check-sm me-5"
                        >
                          <input
                            class="form-check-input"
                            type="radio"
                            v-model="config.illustrations.set"
                            value="sketchy-1"
                            name="illustrations-set"
                            id="sketchy-1"
                          />
                        </span>
                        <span class="fs-4 fw-bold">Sketchy</span>
                      </span>
                      <span
                          :style="{
                          backgroundImage: `url(${getAssetPath(
                            '/media/illustrations/sketchy-1/1.png'
                          )})`,
                        }"
                        class="d-block h-200px w-100 bgi-no-repeat bgi-size-contain bgi-position-center"
                      ></span>
                    </label>
                  </div>
                  <div class="col-6">
                    <label
                      class="btn btn-outline btn-outline-dashed btn-outline-default d-flex flex-column text-start p-6"
                      for="sigma-1"
                    >
                      <span class="d-flex mb-2">
                        <span
                          class="form-check form-check-custom form-check-solid form-check-sm me-5"
                        >
                          <input
                            class="form-check-input"
                            type="radio"
                            v-model="config.illustrations.set"
                            value="sigma-1"
                            name="illustrations-set"
                            id="sigma-1"
                          />
                        </span>
                        <span class="fs-4 fw-bold">Sigma</span>
                      </span>
                      <span
                          :style="{
                          backgroundImage: `url(${getAssetPath(
                            '/media/illustrations/sigma-1/1.png'
                          )})`,
                        }"
                        class="d-block h-200px w-100 bgi-no-repeat bgi-size-contain bgi-position-center"
                      ></span>
                    </label>
                  </div>
                  <div class="col-6">
                    <label
                      class="btn btn-outline btn-outline-dashed btn-outline-default d-flex flex-column text-start p-6"
                      for="dozzy-1"
                    >
                      <span class="d-flex mb-2">
                        <span
                          class="form-check form-check-custom form-check-solid form-check-sm me-5"
                        >
                          <input
                            class="form-check-input"
                            type="radio"
                            v-model="config.illustrations.set"
                            value="dozzy-1"
                            name="illustrations-set"
                            id="dozzy-1"
                          />
                        </span>
                        <span class="fs-4 fw-bold">Dozzy</span>
                      </span>
                      <span
                          :style="{
                          backgroundImage: `url(${getAssetPath(
                            '/media/illustrations/dozzy-1/1.png'
                          )})`,
                        }"
                        class="d-block h-200px w-100 bgi-no-repeat bgi-size-contain bgi-position-center"
                      ></span>
                    </label>
                  </div>
                  <div class="col-6">
                    <label
                      class="btn btn-outline btn-outline-dashed btn-outline-default d-flex flex-column text-start p-6"
                      for="unitedpalms-1"
                    >
                      <span class="d-flex mb-2">
                        <span
                          class="form-check form-check-custom form-check-solid form-check-sm me-5"
                        >
                          <input
                            class="form-check-input"
                            type="radio"
                            v-model="config.illustrations.set"
                            value="unitedpalms-1"
                            name="illustrations-set"
                            id="unitedpalms-1"
                          />
                        </span>
                        <span class="fs-4 fw-bold">United Palms</span>
                      </span>
                      <span
                          :style="{
                          backgroundImage: `url(${getAssetPath(
                            '/media/illustrations/unitedpalms-1/1.png'
                          )})`,
                        }"
                        class="d-block h-200px w-100 bgi-no-repeat bgi-size-contain bgi-position-center"
                      ></span>
                    </label>
                  </div>
                </div>
                <div class="form-text text-muted">
                  Select one of our in-house designed illustration set and apply
                  globally for all apps and pages
                </div>
              </div>
            </div>
          </div>

          <div
            class="tab-pane"
            :class="{ active: tabIndex === 1 }"
            id="kt_builder_header"
          >
            <div class="row mb-10">
              <label class="col-lg-3 col-form-label text-lg-end">Width:</label>
              <div class="col-lg-9 col-xl-4">
                <select
                  class="form-select form-select-solid"
                  v-model="config.header.width"
                >
                  <option value="fluid">Fluid</option>
                  <option value="fixed">Fixed</option>
                </select>
                <div class="form-text text-muted">
                  Select header width type.
                </div>
              </div>
            </div>
          </div>

          <div
            class="tab-pane"
            :class="{ active: tabIndex === 3 }"
            id="kt_builder_aside"
          >
            <div class="row mb-10">
              <label class="col-lg-3 col-form-label text-lg-end"
                >Menu Icon:</label
              >
              <div class="col-lg-9 col-xl-4">
                <select
                  class="form-select form-select-solid"
                  v-model="config.aside.menuIcon"
                >
                  <option value="svg">SVG Icon</option>
                  <option value="font">Font Icon</option>
                </select>
                <div class="form-text text-muted">Select menu icon type.</div>
              </div>
            </div>
            <div class="row mb-10">
              <label class="col-lg-3 col-form-label text-lg-end"
                >Minimized:</label
              >
              <div class="col-lg-9 col-xl-4">
                <span class="switch switch-icon">
                  <input type="hidden" v-model="config.aside.minimized" />
                  <div
                    class="form-check form-check-custom form-check-solid form-switch mb-2"
                  >
                    <input
                      class="form-check-input"
                      type="checkbox"
                      v-model="config.aside.minimized"
                    />
                  </div>
                </span>
                <div class="form-text text-muted">
                  Minimize secondary panel by default
                </div>
              </div>
            </div>
          </div>

          <div
            class="tab-pane"
            :class="{ active: tabIndex === 4 }"
            id="kt_builder_content"
          >
            <div class="row mb-10">
              <label class="col-lg-3 col-form-label text-lg-end">Width:</label>
              <div class="col-lg-9 col-xl-4">
                <select
                  class="form-select form-select-solid"
                  v-model="config.content.width"
                >
                  <option value="fluid">Fluid</option>
                  <option value="fixed">Fixed</option>
                </select>
                <div class="form-text text-muted">
                  Select layout width type.
                </div>
              </div>
            </div>
          </div>

          <div
            class="tab-pane"
            :class="{ active: tabIndex === 5 }"
            id="kt_builder_footer"
          >
            <div class="row mb-10">
              <label class="col-lg-3 col-form-label text-lg-end">Width:</label>
              <div class="col-lg-9 col-xl-4">
                <select
                  class="form-select form-select-solid"
                  v-model="config.footer.width"
                >
                  <option value="fluid">Fluid</option>
                  <option value="fixed">Fixed</option>
                </select>
                <div class="form-text text-muted">
                  Select layout width type.
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <!--end::Body-->

      <!--begin::Footer-->
      <div class="card-footer py-6">
        <div class="row">
          <div class="col-lg-3"></div>
          <div class="col-lg-9">
            <input
              type="hidden"
              id="kt_layout_builder_tab"
              name="layout-builder[tab]"
            />
            <input
              type="hidden"
              id="kt_layout_builder_action"
              name="layout-builder[action]"
            />

            <button
              type="submit"
              id="kt_layout_builder_preview"
              class="btn btn-primary me-2"
            >
              <span class="indicator-label"> Preview </span>
              <span class="indicator-progress">
                Please wait...
                <span
                  class="spinner-border spinner-border-sm align-middle ms-2"
                ></span>
              </span>
            </button>

            <button
              @click="reset($event)"
              type="button"
              id="kt_layout_builder_reset"
              class="btn btn-active-light btn-color-muted fw-semobold"
            >
              Reset
            </button>
          </div>
        </div>
      </div>
      <!--end::Footer-->
    </form>
    <!--end::Form-->
  </div>

  <!--begin::Card-->
  <div class="card">
    <!--begin::Card Body-->
    <div class="card-body p-10 p-lg-15">
      <!--begin::Heading-->
      <h1 class="anchor fw-bold mb-5" id="overview">Generated config</h1>
      <!--end::Heading-->
      <div class="py-5">
        This config is used as a <b>{{ themeName }}</b> layout config, you can
        update default values inside file
        <code>metronic/vue/theme/src/core/config/DefaultLayoutConfig.ts</code>
      </div>
      <CodeHighlighter lang="json">{{ config }}</CodeHighlighter>
    </div>
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, onMounted, ref } from "vue";
import { config } from "@/core/helpers/config";
import CodeHighlighter from "@/components/highlighters/CodeHighlighter.vue";
import { themeName } from "@/core/helpers/documentation";
import { LS_CONFIG_NAME_KEY } from "@/stores/config";

const LS_BUILDER_TAB_NAME =
  "layoutBuilderTabIndex_" + import.meta.env.VITE_APP_DEMO;

export default defineComponent({
  name: "layout-builder",
  components: {
    CodeHighlighter,
  },
  setup() {
    const tabIndex = ref(0);

    onMounted(() => {
      // set the tab from previous
      tabIndex.value = parseInt(
        localStorage.getItem(LS_BUILDER_TAB_NAME) || "0"
      );
    });

    /**
     * Reset config
     * @param event
     */
    const reset = (event: any) => {
      event.preventDefault();
      // remove existing saved config
      localStorage.removeItem(LS_CONFIG_NAME_KEY);
      window.location.reload();
    };

    /**
     * Set active tab when the tab get clicked
     * @param event
     */
    const setActiveTab = (event: any) => {
      tabIndex.value = parseInt(event.target.getAttribute("data-tab-index"));
      // keep active tab
      localStorage.setItem(LS_BUILDER_TAB_NAME, tabIndex.value.toString());
    };

    /**
     * Submit form
     * @param event
     */
    const submit = (event: any) => {
      event.preventDefault();
      // save new config to localStorage
      localStorage.setItem(LS_CONFIG_NAME_KEY, JSON.stringify(config.value));
      window.location.reload();
    };

    return {
      tabIndex,
      config,
      reset,
      setActiveTab,
      submit,
      themeName,
      getAssetPath,
    };
  },
});
</script>
