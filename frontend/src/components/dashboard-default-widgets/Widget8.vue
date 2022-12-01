<template>
  <div class="card card-flush" :class="className">
    <!--begin::Body-->
    <div class="card-body py-9">
      <!--begin::Row-->
      <div class="row gx-9 h-100">
        <!--begin::Col-->
        <div class="col-sm-6 mb-10 mb-sm-0">
          <!--begin::Image-->
          <div
            class="bgi-no-repeat bgi-position-center bgi-size-cover card-rounded min-h-400px min-h-sm-100 h-100"
            :style="{
              backgroundSize: '100% 100%',
              backgroundImage: `url(${image})`,
            }"
          ></div>
          <!--end::Image-->
        </div>
        <!--end::Col-->

        <!--begin::Col-->
        <div class="col-sm-6">
          <!--begin::Wrapper-->
          <div class="d-flex flex-column h-100">
            <!--begin::Header-->
            <div class="mb-7">
              <!--begin::Headin-->
              <div class="d-flex flex-stack mb-6">
                <!--begin::Title-->
                <div class="flex-shrink-0 me-5">
                  <span
                    class="text-gray-400 fs-7 fw-bold me-2 d-block lh-1 pb-1"
                    >Featured</span
                  >

                  <span class="text-gray-800 fs-1 fw-bold">9 Degree</span>
                </div>
                <!--end::Title-->

                <span
                  class="badge badge-light-primary flex-shrink-0 align-self-center py-3 px-4 fs-7"
                  >In Process</span
                >
              </div>
              <!--end::Heading-->

              <!--begin::Items-->
              <div class="d-flex align-items-center flex-wrap d-grid gap-2">
                <!--begin::Item-->
                <div class="d-flex align-items-center me-5 me-xl-13">
                  <!--begin::Symbol-->
                  <div class="symbol symbol-30px symbol-circle me-3">
                    <img
                      :src="getAssetPath('/media/avatars/300-3.jpg')"
                      class=""
                      alt=""
                    />
                  </div>
                  <!--end::Symbol-->

                  <!--begin::Info-->
                  <div class="m-0">
                    <span class="fw-semibold text-gray-400 d-block fs-8"
                      >Manager</span
                    >
                    <router-link
                      to="/crafted/pages/profile/overview"
                      class="fw-bold text-gray-800 text-hover-primary fs-7"
                      >Robert Fox</router-link
                    >
                  </div>
                  <!--end::Info-->
                </div>
                <!--end::Item-->

                <!--begin::Item-->
                <div class="d-flex align-items-center">
                  <!--begin::Symbol-->
                  <div class="symbol symbol-30px symbol-circle me-3">
                    <span class="symbol-label bg-success">
                      <span class="svg-icon svg-icon-5 svg-icon-white">
                        <inline-svg
                          :src="
                            getAssetPath(
                              '/media/icons/duotune/abstract/abs042.svg'
                            )
                          "
                        />
                      </span>
                    </span>
                  </div>
                  <!--end::Symbol-->

                  <!--begin::Info-->
                  <div class="m-0">
                    <span class="fw-semibold text-gray-400 d-block fs-8"
                      >Budget</span
                    >
                    <span class="fw-bold text-gray-800 fs-7">$64.800</span>
                  </div>
                  <!--end::Info-->
                </div>
                <!--end::Item-->
              </div>
              <!--end::Items-->
            </div>
            <!--end::Header-->

            <!--begin::Body-->
            <div class="mb-6">
              <!--begin::Text-->
              <span class="fw-semibold text-gray-600 fs-6 mb-8 d-block">
                Flat cartoony illustrations with vivid unblended colors and
                asymmetrical beautiful purple hair lady
              </span>
              <!--end::Text-->

              <!--begin::Stats-->
              <div class="d-flex">
                <!--begin::Stat-->
                <div
                  class="border border-gray-300 border-dashed rounded min-w-100px w-100 py-2 px-4 me-6 mb-3"
                >
                  <!--begin::Date-->
                  <span class="fs-6 text-gray-700 fw-bold">Feb 6, 2021 </span>
                  <!--end::Date-->

                  <!--begin::Label-->
                  <div class="fw-semibold text-gray-400">Due Date</div>
                  <!--end::Label-->
                </div>
                <!--end::Stat-->

                <!--begin::Stat-->
                <div
                  class="border border-gray-300 border-dashed rounded min-w-100px w-100 py-2 px-4 mb-3"
                >
                  <!--begin::Number-->
                  <span class="fs-6 text-gray-700 fw-bold"
                    >$ <span class="ms-n1">284,900</span></span
                  >
                  <!--end::Number-->

                  <!--begin::Label-->
                  <div class="fw-semibold text-gray-400">Budget</div>
                  <!--end::Label-->
                </div>
                <!--end::Stat-->
              </div>
              <!--end::Stats-->
            </div>
            <!--end::Body-->

            <!--begin::Footer-->
            <div class="d-flex flex-stack mt-auto bd-highlight">
              <!--begin::Users group-->
              <div class="symbol-group symbol-hover flex-nowrap">
                <div class="symbol-group symbol-hover flex-nowrap">
                  <div class="symbol symbol-35px symbol-circle">
                    <img
                      alt="Pic"
                      :src="getAssetPath('/media/avatars/300-2.jpg')"
                    />
                  </div>
                  <div class="symbol symbol-35px symbol-circle">
                    <img
                      alt="Pic"
                      :src="getAssetPath('/media/avatars/300-3.jpg')"
                    />
                  </div>
                  <div class="symbol symbol-35px symbol-circle">
                    <span
                      class="symbol-label bg-primary text-inverse-primary fw-bold"
                      >S</span
                    >
                  </div>
                </div>
              </div>
              <!--end::Users group-->

              <!--begin::Actions-->
              <a
                href="<?php echo Theme::getPageUrl('apps/projects/project')?>"
                class="text-primary opacity-75-hover fs-6 fw-semibold"
              >
                View Project

                <span class="svg-icon svg-icon-4 svg-icon-gray-800 ms-1">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/arrows/arr095.svg')
                    "
                  />
                </span>
              </a>
              <!--end::Actions-->
            </div>
            <!--end::Footer-->
          </div>
          <!--end::Wrapper-->
        </div>
        <!--end::Col-->
      </div>
      <!--end::Row-->
    </div>
    <!--end::Body-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "default-dashboard-widget-8",
  components: {},
  props: {
    className: { type: String, required: false },
    image: { type: String, required: true },
  },
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
