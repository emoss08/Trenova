<template>
  <!--begin::List Widget 2-->
  <div class="card" :class="widgetClasses">
    <!--begin::Header-->
    <div class="card-header border-0">
      <h3 class="card-title fw-bold text-dark">Authors</h3>

      <div class="card-toolbar">
        <!--begin::Menu-->
        <button
          type="button"
          class="btn btn-sm btn-icon btn-color-primary btn-active-light-primary"
          data-kt-menu-trigger="click"
          data-kt-menu-placement="bottom-end"
          data-kt-menu-flip="top-end"
        >
          <span class="svg-icon svg-icon-2">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen024.svg')"
            />
          </span>
        </button>
        <Dropdown2></Dropdown2>
        <!--end::Menu-->
      </div>
    </div>
    <!--end::Header-->

    <!--begin::Body-->
    <div class="card-body pt-2">
      <template v-for="(item, index) in list" :key="index">
        <!--begin::Item-->
        <div
          :class="{ 'mb-7': list.length - 1 !== index }"
          class="d-flex align-items-center"
        >
          <!--begin::Avatar-->
          <div class="symbol symbol-50px me-5">
            <img :src="item.avatar" class="" alt="" />
          </div>
          <!--end::Avatar-->

          <!--begin::Text-->
          <div class="flex-grow-1">
            <a href="#" class="text-dark fw-bold text-hover-primary fs-6">{{
              item.name
            }}</a>

            <span class="text-muted d-block fw-semobold">{{
              item.description
            }}</span>
          </div>
          <!--end::Text-->
        </div>
        <!--end::Item-->
      </template>
    </div>
    <!--end::Body-->
  </div>
  <!--end::List Widget 2-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref } from "vue";
import Dropdown2 from "@/components/dropdown/Dropdown2.vue";

export default defineComponent({
  name: "kt-widget-2",
  components: {
    Dropdown2,
  },
  props: {
    widgetClasses: String,
  },
  setup() {
    const list = ref([
      {
        avatar: getAssetPath("/media/avatars/300-6.jpg"),
        name: "Emma Smith",
        description: "Project Manager",
      },
      {
        avatar: getAssetPath("/media/avatars/300-5.jpg"),
        name: "Sean Bean",
        description: "PHP, SQLite, Artisan CLI",
      },
      {
        avatar: getAssetPath("/media/avatars/300-11.jpg"),
        name: "Brian Cox",
        description: "PHP, SQLite, Artisan CLI",
      },
      {
        avatar: getAssetPath("/media/avatars/300-9.jpg"),
        name: "Francis Mitcham",
        description: "PHP, SQLite, Artisan CLI",
      },
      {
        avatar: getAssetPath("/media/avatars/300-23.jpg"),
        name: "Dan Wilson",
        description: "PHP, SQLite, Artisan CLI",
      },
    ]);

    return {
      list,
      getAssetPath,
    };
  },
});
</script>
