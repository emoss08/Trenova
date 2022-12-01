<template>
  <!--begin::Card-->
  <div class="card h-100" :class="cardClasses">
    <!--begin::Card header-->
    <div class="card-header flex-nowrap border-0 pt-9">
      <!--begin::Card title-->
      <div class="card-title m-0">
        <!--begin::Icon-->
        <div class="symbol symbol-45px w-45px bg-light me-5">
          <img :src="icon" alt="image" class="p-3" />
        </div>
        <!--end::Icon-->

        <!--begin::Title-->
        <a
          href="#"
          class="fs-4 fw-semobold text-hover-primary text-gray-600 m-0"
        >
          {{ title }}
        </a>
        <!--end::Title-->
      </div>
      <!--end::Card title-->

      <!--begin::Card toolbar-->
      <div class="card-toolbar m-0">
        <!--begin::Menu-->
        <button
          type="button"
          class="btn btn-clean btn-sm btn-icon btn-icon-primary btn-active-light-primary me-n3"
          data-kt-menu-trigger="click"
          data-kt-menu-placement="bottom-end"
          data-kt-menu-flip="top-end"
        >
          <span class="svg-icon svg-icon-3 svg-icon-primary">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen024.svg')"
            />
          </span>
        </button>

        <Dropdown3></Dropdown3>
        <!--end::Menu-->
      </div>
      <!--end::Card toolbar-->
    </div>
    <!--end::Card header-->

    <!--begin::Card body-->
    <div class="card-body d-flex flex-column px-9 pt-6 pb-8">
      <!--begin::Heading-->
      <div class="fs-2tx fw-bold mb-3">
        {{ content }}
      </div>
      <!--end::Heading-->

      <!--begin::Stats-->
      <div class="d-flex align-items-center flex-wrap mb-5 mt-auto fs-6">
        <span :class="`svg-icon-${color}`" class="svg-icon svg-icon-3 me-1">
          <inline-svg :src="arrow" />
        </span>

        <!--begin::Number-->
        <div :class="`text-${color}`" class="fw-bold me-2">
          {{ footerData }}
        </div>
        <!--end::Number-->

        <!--begin::Label-->
        <div class="fw-semobold text-gray-400">
          {{ footerText }}
        </div>
        <!--end::Label-->
      </div>
      <!--end::Stats-->

      <!--begin::Indicator-->
      <div class="d-flex align-items-center fw-semobold">
        <span class="badge bg-light text-gray-700 px-3 py-2 me-2">
          {{ indicatorValue }}
        </span>

        <span class="text-gray-400 fs-7">{{ indicatorLabel }}</span>
      </div>
      <!--end::Indicator-->
    </div>
    <!--end::Card body-->
  </div>
  <!--end::Card-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import Dropdown3 from "@/components/dropdown/Dropdown3.vue";

export default defineComponent({
  name: "card-2",
  components: {
    Dropdown3,
  },
  props: {
    cardClasses: String,

    icon: String,

    title: String,

    content: String,

    color: String,

    arrow: String,

    footerData: String,

    footerText: String,

    indicatorValue: String,

    indicatorLabel: String,
  },
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
