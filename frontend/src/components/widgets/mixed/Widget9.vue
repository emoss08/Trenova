<template>
  <!--begin::Mixed Widget 9-->
  <div :class="widgetClasses" class="card">
    <!--begin::Body-->
    <div class="card-body d-flex flex-column">
      <div class="flex-grow-1">
        <!--begin::Info-->
        <div class="d-flex align-items-center pe-2 mb-5">
          <span class="text-muted fw-bold fs-5 flex-grow-1">
            {{ time }}
          </span>

          <div class="symbol symbol-50px">
            <span class="symbol-label bg-light">
              <img :src="image" class="h-50 align-self-center" alt="" />
            </span>
          </div>
        </div>
        <!--end::Info-->

        <!--begin::Link-->
        <a href="#" class="text-dark fw-bold text-hover-primary fs-4">
          {{ title }}
        </a>
        <!--end::Link-->

        <!--begin::Desc-->
        <p class="py-3">
          <span v-html="description"></span>
        </p>
        <!--end::Desc-->
      </div>

      <!--begin::Team-->
      <div class="d-flex align-items-center">
        <a
          href="#"
          class="symbol symbol-35px me-2"
          data-bs-toggle="tooltip"
          title="Ana Stone"
        >
          <img :src="getAssetPath('/media/avatars/300-6.jpg')" alt="" />
        </a>

        <a
          href="#"
          class="symbol symbol-35px me-2"
          data-bs-toggle="tooltip"
          title="Mark Larson"
        >
          <img :src="getAssetPath('/media/avatars/300-5.jpg')" alt="" />
        </a>

        <a
          href="#"
          class="symbol symbol-35px me-2"
          data-bs-toggle="tooltip"
          title="Sam Harris"
        >
          <img :src="getAssetPath('/media/avatars/300-9.jpg')" alt="" />
        </a>
      </div>
      <!--end::Team-->
    </div>
    <!--end::Body-->
  </div>
  <!--end::Mixed Widget 9-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "widget-1",
  components: {},
  props: {
    widgetClasses: String,
    time: String,
    image: String,
    title: String,
    description: String,
  },
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
