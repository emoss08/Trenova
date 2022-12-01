<template>
  <!--begin::Statistics Widget 1-->
  <div
    :class="widgetClasses"
    :style="{
      backgroundImage: `url(${getAssetPath(
        '/media/svg/shapes/' + background
      )})`,
    }"
    class="card bgi-no-repeat"
    style="background-position: right top; background-size: 30% auto"
  >
    <!--begin::Body-->
    <div class="card-body">
      <a
        href="#"
        class="card-title fw-bold text-muted text-hover-primary fs-4"
        >{{ title }}</a
      >

      <div class="fw-bold text-primary my-6">{{ time }}</div>

      <p class="text-dark-75 fw-semobold fs-5 m-0">
        <span v-html="description"></span>
      </p>
    </div>
    <!--end::Body-->
  </div>
  <!--end::Statistics Widget 1-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-widget-1",
  props: {
    widgetClasses: String,
    background: String,
    title: String,
    time: String,
    description: String,
  },
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
