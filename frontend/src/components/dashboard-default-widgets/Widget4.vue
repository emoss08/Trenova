<template>
  <div class="card card-flush" :class="className">
    <!--begin::Header-->
    <div class="card-header pt-5">
      <!--begin::Title-->
      <h3 class="card-title align-items-start flex-column">
        <span class="card-label fw-bold text-dark">Highlights</span>
        <span class="text-gray-400 mt-1 fw-semibold fs-6"
          >Latest social statistics</span
        >
      </h3>
      <!--end::Title-->

      <!--begin::Toolbar-->
      <div class="card-toolbar">
        <!--begin::Menu-->
        <button
          class="btn btn-icon btn-color-gray-400 btn-active-color-primary justify-content-end"
          data-kt-menu-trigger="click"
          data-kt-menu-placement="bottom-end"
          data-kt-menu-overflow="true"
        >
          <span class="svg-icon svg-icon-1">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen023.svg')"
            />
          </span>
        </button>
        <Dropdown2 />
        <!--end::Menu-->
      </div>
      <!--end::Toolbar-->
    </div>
    <!--end::Header-->

    <!--begin::Body-->
    <div class="card-body pt-5">
      <template v-for="(item, i) in items" :key="i">
        <!--begin::Item-->
        <div class="d-flex flex-stack">
          <!--begin::Section-->
          <div class="text-gray-700 fw-semibold fs-6 me-2">
            {{ item.description }}
          </div>
          <!--end::Section-->

          <!--begin::Statistics-->
          <div class="d-flex align-items-senter">
            <span
              v-if="item.inc"
              class="svg-icon svg-icon-2 svg-icon-success me-2"
            >
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr094.svg')"
              />
            </span>

            <span v-else class="svg-icon svg-icon-2 svg-icon-danger me-2">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr093.svg')"
              />
            </span>

            <!--begin::Number-->
            <span class="text-gray-900 fw-bolder fs-6">{{
              item.statistics
            }}</span>
            <!--end::Number-->
          </div>
          <!--end::Statistics-->
        </div>
        <!--end::Item-->

        <div
          v-if="items.length - 1 !== i"
          class="separator separator-dashed my-3"
        ></div>
      </template>
    </div>
    <!--end::Body-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import Dropdown2 from "@/components/dropdown/Dropdown2.vue";

export default defineComponent({
  name: "default-dashboard-widget-4",
  components: {
    Dropdown2,
  },
  props: {
    className: { type: String, required: false },
  },
  setup() {
    const items = [
      {
        description: "Avg. Client Rating",
        statistics: "7.8",
        inc: true,
        number: "true",
      },
      {
        description: "Instagram Followers",
        statistics: "730k",
        inc: false,
        number: "false",
      },
      {
        description: "Google Ads CPC",
        statistics: "$2.09",
        inc: true,
        number: "false",
      },
    ];

    return {
      items,
      getAssetPath,
    };
  },
});
</script>
