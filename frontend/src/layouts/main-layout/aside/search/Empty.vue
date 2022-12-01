<template>
  <!--begin::Empty-->
  <div data-kt-search-element="empty" class="text-center d-none">
    <!--begin::Icon-->
    <div class="pt-10 pb-10">
      <span class="svg-icon svg-icon-4x opacity-50">
        <inline-svg
          :src="getAssetPath('/media/icons/duotune/files/fil024.svg')"
        />
      </span>
    </div>
    <!--end::Icon-->

    <!--begin::Message-->
    <div class="pb-15 fw-semobold">
      <h3 class="text-gray-600 fs-5 mb-2">No result found</h3>
      <div class="text-muted fs-7">Please try again with a different query</div>
    </div>
    <!--end::Message-->
  </div>
  <!--end::Empty-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-empty",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
