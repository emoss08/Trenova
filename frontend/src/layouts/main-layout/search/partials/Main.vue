<template>
  <div class="mb-4">
    <!--begin::Heading-->
    <div class="d-flex flex-stack fw-semobold mb-4">
      <!--begin::Label-->
      <span class="text-muted fs-6 me-2">Recently Searched:</span>
      <!--end::Label-->
    </div>
    <!--end::Heading-->
    <!--begin::Items-->
    <div class="scroll-y mh-200px mh-lg-325px">
      <!--begin::Item-->
      <div class="d-flex align-items-center mb-5">
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="
                  getAssetPath('/media/icons/duotune/electronics/elc004.svg')
                "
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <a href="#" class="fs-6 text-gray-800 text-hover-primary fw-semobold"
            >BoomApp by Keenthemes</a
          >
          <span class="fs-7 text-muted fw-semobold">#45789</span>
        </div>
        <!--end::Title-->
      </div>
      <!--end::Item-->
      <!--begin::Item-->
      <div class="d-flex align-items-center mb-5">
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/graphs/gra001.svg')"
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <a href="#" class="fs-6 text-gray-800 text-hover-primary fw-semobold"
            >"Kept API Project Meeting</a
          >
          <span class="fs-7 text-muted fw-semobold">#84050</span>
        </div>
        <!--end::Title-->
      </div>
      <!--end::Item-->
      <!--begin::Item-->
      <div class="d-flex align-items-center mb-5">
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/graphs/gra006.svg')"
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <a href="#" class="fs-6 text-gray-800 text-hover-primary fw-semobold"
            >"KPI Monitoring App Launch</a
          >
          <span class="fs-7 text-muted fw-semobold">#84250</span>
        </div>
        <!--end::Title-->
      </div>
      <!--end::Item-->
      <!--begin::Item-->
      <div class="d-flex align-items-center mb-5">
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/graphs/gra002.svg')"
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <a href="#" class="fs-6 text-gray-800 text-hover-primary fw-semobold"
            >Project Reference FAQ</a
          >
          <span class="fs-7 text-muted fw-semobold">#67945</span>
        </div>
        <!--end::Title-->
      </div>
      <!--end::Item-->
      <!--begin::Item-->
      <div class="d-flex align-items-center mb-5">
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="
                  getAssetPath('/media/icons/duotune/communication/com010.svg')
                "
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <a href="#" class="fs-6 text-gray-800 text-hover-primary fw-semobold"
            >"FitPro App Development</a
          >
          <span class="fs-7 text-muted fw-semobold">#84250</span>
        </div>
        <!--end::Title-->
      </div>
      <!--end::Item-->
      <!--begin::Item-->
      <div class="d-flex align-items-center mb-5">
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/finance/fin001.svg')"
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <a href="#" class="fs-6 text-gray-800 text-hover-primary fw-semobold"
            >Shopix Mobile App</a
          >
          <span class="fs-7 text-muted fw-semobold">#45690</span>
        </div>
        <!--end::Title-->
      </div>
      <!--end::Item-->
      <!--begin::Item-->
      <div class="d-flex align-items-center mb-5">
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/graphs/gra002.svg')"
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <a href="#" class="fs-6 text-gray-800 text-hover-primary fw-semobold"
            >"Landing UI Design" Launch</a
          >
          <span class="fs-7 text-muted fw-semobold">#24005</span>
        </div>
        <!--end::Title-->
      </div>
      <!--end::Item-->
    </div>
    <!--end::Items-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-main",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
