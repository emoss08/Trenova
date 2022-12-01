<template>
  <!--begin::Help drawer-->
  <div
    id="kt_help"
    class="bg-body"
    data-kt-drawer="true"
    data-kt-drawer-name="help"
    data-kt-drawer-activate="true"
    data-kt-drawer-overlay="true"
    data-kt-drawer-width="{default:'350px', 'md': '525px'}"
    data-kt-drawer-direction="end"
    data-kt-drawer-toggle="#kt_help_toggle"
    data-kt-drawer-close="#kt_help_close"
  >
    <!--begin::Card-->
    <div class="card shadow-none rounded-0 w-100">
      <!--begin::Header-->
      <div class="card-header" id="kt_help_header">
        <h5 class="card-title fw-semobold text-gray-600">
          Learn & Get Inspired
        </h5>

        <div class="card-toolbar">
          <button
            type="button"
            class="btn btn-sm btn-icon explore-btn-dismiss me-n5"
            id="kt_help_close"
          >
            <span class="svg-icon svg-icon-2">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr061.svg')"
              />
            </span>
          </button>
        </div>
      </div>
      <!--end::Header-->

      <!--begin::Body-->
      <div class="card-body" id="kt_help_body">
        <!--begin::Content-->
        <div
          id="kt_help_scroll"
          class="hover-scroll-overlay-y"
          data-kt-scroll="true"
          data-kt-scroll-height="auto"
          data-kt-scroll-wrappers="#kt_help_body"
          data-kt-scroll-dependencies="#kt_help_header"
          data-kt-scroll-offset="5px"
        >
          <!--begin::Support-->
          <div
            class="rounded border border-dashed border-gray-300 p-6 p-lg-8 mb-10"
          >
            <!--begin::Heading-->
            <h2 class="fw-bold mb-5">
              Support at
              <a href="https://devs.keenthemes.com" class=""
                >devs.keenthemes.com</a
              >
            </h2>
            <!--end::Heading-->
            <!--begin::Description-->
            <div class="fs-5 fw-semobold mb-5">
              <span class="text-gray-500"
                >Join our developers community to find answer to your question
                and help others.</span
              >
              <a
                class="explore-link d-none"
                href="https://keenthemes.com/licensing"
                >FAQs</a
              >
            </div>
            <!--end::Description-->
            <!--begin::Link-->
            <a
              href="https://devs.keenthemes.com"
              class="btn btn-lg explore-btn-primary w-100"
              >Get Support</a
            >
            <!--end::Link-->
          </div>
          <!--end::Support-->
          <!--begin::Link-->
          <div class="d-flex align-items-center mb-7">
            <!--begin::Icon-->
            <div
              class="d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-warning"
            >
              <span
                class="svg-icon svg-icon-warning svg-icon-2x svg-icon-lg-3x"
              >
                <inline-svg
                  :src="
                    getAssetPath('/media/icons/duotune/abstract/abs027.svg')
                  "
                />
              </span>
            </div>
            <!--end::Icon-->
            <!--begin::Info-->
            <div class="d-flex flex-stack flex-grow-1 ms-4 ms-lg-6">
              <!--begin::Wrapper-->
              <div class="d-flex flex-column me-2 me-lg-5">
                <!--begin::Title-->
                <a
                  href="https://preview.keenthemes.com/metronic8/vue/docs/#/build"
                  class="text-dark text-hover-primary fw-bold fs-6 fs-lg-4 mb-1"
                  >Documentation &amp; Videos</a
                >
                <!--end::Title-->
                <!--begin::Description-->
                <div class="text-muted fw-semobold fs-7 fs-lg-6">
                  From guides and video tutorials, to live demos and code
                  examples to get started.
                </div>
                <!--end::Description-->
              </div>
              <!--end::Wrapper-->
              <span class="svg-icon svg-icon-gray-400 svg-icon-2">
                <inline-svg
                  :src="getAssetPath('/media/icons/duotune/arrows/arr064.svg')"
                />
              </span>
            </div>
            <!--end::Info-->
          </div>
          <!--end::Link-->
          <!--begin::Link-->
          <div class="d-flex align-items-center mb-7">
            <!--begin::Icon-->
            <div
              class="d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-primary"
            >
              <span
                class="svg-icon svg-icon-primary svg-icon-2x svg-icon-lg-3x"
              >
                <inline-svg
                  :src="
                    getAssetPath('/media/icons/duotune/ecommerce/ecm007.svg')
                  "
                />
              </span>
            </div>
            <!--end::Icon-->
            <!--begin::Info-->
            <div class="d-flex flex-stack flex-grow-1 ms-4 ms-lg-6">
              <!--begin::Wrapper-->
              <div class="d-flex flex-column me-2 me-lg-5">
                <!--begin::Title-->
                <a
                  href="https://preview.keenthemes.com/metronic8/vue/docs/#/utilities"
                  class="text-dark text-hover-primary fw-bold fs-6 fs-lg-4 mb-1"
                  >Plugins &amp; Components</a
                >
                <!--end::Title-->
                <!--begin::Description-->
                <div class="text-muted fw-semobold fs-7 fs-lg-6">
                  Check out our 300+ in-house components and customized
                  3rd-party plugins.
                </div>
                <!--end::Description-->
              </div>
              <!--end::Wrapper-->
              <span class="svg-icon svg-icon-gray-400 svg-icon-2">
                <inline-svg
                  :src="getAssetPath('/media/icons/duotune/arrows/arr064.svg')"
                />
              </span>
            </div>
            <!--end::Info-->
          </div>
          <!--end::Link-->
          <!--begin::Link-->
          <div class="d-flex align-items-center mb-7">
            <!--begin::Icon-->
            <div
              class="d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-info"
            >
              <span class="svg-icon svg-icon-info svg-icon-2x svg-icon-lg-3x">
                <inline-svg
                  :src="getAssetPath('/media/icons/duotune/art/art006.svg')"
                />
              </span>
            </div>
            <!--end::Icon-->
            <!--begin::Info-->
            <div class="d-flex flex-stack flex-grow-1 ms-4 ms-lg-6">
              <!--begin::Wrapper-->
              <div class="d-flex flex-column me-2 me-lg-5">
                <!--begin::Title-->
                <router-link
                  to="/builder"
                  class="text-dark text-hover-primary fw-bold fs-6 fs-lg-4 mb-1"
                  >Layout Builder</router-link
                >
                <!--end::Title-->
                <!--begin::Description-->
                <div class="text-muted fw-semobold fs-7 fs-lg-6">
                  Dynamically modify and preview layout
                </div>
                <!--end::Description-->
              </div>
              <!--end::Wrapper-->
              <!--begin::Svg Icon | path: icons/duotune/arrows/arr064.svg-->
              <span class="svg-icon svg-icon-gray-400 svg-icon-2">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                >
                  <rect
                    opacity="0.5"
                    x="18"
                    y="13"
                    width="13"
                    height="2"
                    rx="1"
                    transform="rotate(-180 18 13)"
                    fill="black"
                  ></rect>
                  <path
                    d="M15.4343 12.5657L11.25 16.75C10.8358 17.1642 10.8358 17.8358 11.25 18.25C11.6642 18.6642 12.3358 18.6642 12.75 18.25L18.2929 12.7071C18.6834 12.3166 18.6834 11.6834 18.2929 11.2929L12.75 5.75C12.3358 5.33579 11.6642 5.33579 11.25 5.75C10.8358 6.16421 10.8358 6.83579 11.25 7.25L15.4343 11.4343C15.7467 11.7467 15.7467 12.2533 15.4343 12.5657Z"
                    fill="black"
                  ></path>
                </svg>
              </span>
              <!--end::Svg Icon-->
            </div>
            <!--end::Info-->
          </div>
          <!--end::Link-->
          <!--begin::Link-->
          <div class="d-flex align-items-center mb-7">
            <!--begin::Icon-->
            <div
              class="d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-danger"
            >
              <span class="svg-icon svg-icon-danger svg-icon-2x svg-icon-lg-3x">
                <inline-svg
                  :src="
                    getAssetPath('/media/icons/duotune/electronics/elc009.svg')
                  "
                />
              </span>
            </div>
            <!--end::Icon-->
            <!--begin::Info-->
            <div class="d-flex flex-stack flex-grow-1 ms-4 ms-lg-6">
              <!--begin::Wrapper-->
              <div class="d-flex flex-column me-2 me-lg-5">
                <!--begin::Title-->
                <a
                  href="https://preview.keenthemes.com/metronic8/vue/docs/#/changelog"
                  class="text-dark text-hover-primary fw-bold fs-6 fs-lg-4 mb-1"
                  >What's New</a
                >
                <!--end::Title-->
                <!--begin::Description-->
                <div class="text-muted fw-semobold fs-7 fs-lg-6">
                  Latest features and improvements added with our users feedback
                  in mind.
                </div>
                <!--end::Description-->
              </div>
              <!--end::Wrapper-->
              <span class="svg-icon svg-icon-gray-400 svg-icon-2">
                <inline-svg
                  :src="getAssetPath('/media/icons/duotune/arrows/arr064.svg')"
                />
              </span>
            </div>
            <!--end::Info-->
          </div>
          <!--end::Link-->
        </div>
        <!--end::Content-->
      </div>
      <!--end::Body-->
    </div>
    <!--end::Card-->
  </div>
  <!--end::Help drawer-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-explore",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
