<template>
  <!--begin::User-->
  <div
    class="aside-user d-flex align-items-sm-center justify-content-center py-5"
  >
    <!--begin::Symbol-->
    <div class="symbol symbol-50px">
      <img :src="getAssetPath('/media/avatars/300-1.jpg?>')" alt="" />
    </div>
    <!--end::Symbol-->

    <!--begin::Wrapper-->
    <div class="aside-user-info flex-row-fluid flex-wrap ms-5">
      <!--begin::Section-->
      <div class="d-flex">
        <!--begin::Info-->
        <div class="flex-grow-1 me-2">
          <!--begin::Username-->
          <a href="#" class="text-white text-hover-primary fs-6 fw-semobold"
            >Paul Melone</a
          >
          <!--end::Username-->

          <!--begin::Description-->
          <span class="text-gray-600 fw-semobold d-block fs-8 mb-1"
            >Python Dev</span
          >
          <!--end::Description-->

          <!--begin::Label-->
          <div class="d-flex align-items-center text-success fs-9">
            <span class="bullet bullet-dot bg-success me-1"></span>online
          </div>
          <!--end::Label-->
        </div>
        <!--end::Info-->

        <!--begin::User menu-->
        <div class="me-n2">
          <!--begin::Action-->
          <a
            href="#"
            class="btn btn-icon btn-sm btn-active-color-primary mt-n2"
            data-kt-menu-trigger="click"
            data-kt-menu-placement="bottom-start"
            data-kt-menu-overflow="true"
          >
            <span class="svg-icon svg-icon-muted svg-icon-1">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/coding/cod001.svg')"
              />
            </span>
          </a>

          <UserMenu />
          <!--end::Action-->
        </div>
        <!--end::User menu-->
      </div>
      <!--end::Section-->
    </div>
    <!--end::Wrapper-->
  </div>
  <!--end::User-->

  <!--begin::Aside search-->
  <div class="aside-search py-5">
    <AsideSearch />
  </div>
  <!--end::Aside search-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import UserMenu from "@/layouts/main-layout/menus/UserAccountMenu.vue";
import AsideSearch from "@/layouts/main-layout/aside/AsideSearch.vue";

export default defineComponent({
  name: "kt--aside-toolbar",
  components: {
    UserMenu,
    AsideSearch,
  },
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
