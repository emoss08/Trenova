<template>
  <!--begin::Card-->
  <div
    class="card card-flush pt-3 mb-0"
    id="kt_add_summary"
    data-kt-sticky="true"
    data-kt-sticky-name="add-subscription-summary"
    data-kt-sticky-offset="{default: false, lg: '200px'}"
    data-kt-sticky-width="{lg: '250px', xl: '300px'}"
    data-kt-sticky-left="auto"
    data-kt-sticky-top="150px"
    data-kt-sticky-animation="false"
    data-kt-sticky-zindex="95"
  >
    <!--begin::Card header-->
    <div class="card-header">
      <!--begin::Card title-->
      <div class="card-title">
        <h2>Summary</h2>
      </div>
      <!--end::Card title-->
    </div>
    <!--end::Card header-->

    <!--begin::Card body-->
    <div class="card-body pt-0 fs-6">
      <!--begin::Section-->
      <div class="mb-7">
        <!--begin::Title-->
        <h5 class="mb-3">Customer details</h5>
        <!--end::Title-->

        <!--begin::Details-->
        <div class="d-flex align-items-center mb-1">
          <!--begin::Name-->
          <router-link
            to="/apps/subscriptions/view-subscription"
            class="fw-bold text-gray-800 text-hover-primary me-2"
          >
            Sean Bean
          </router-link>
          <!--end::Name-->

          <!--begin::Status-->
          <span class="badge badge-light-success">Active</span>
          <!--end::Status-->
        </div>
        <!--end::Details-->

        <!--begin::Email-->
        <a href="#" class="fw-semobold text-gray-600 text-hover-primary"
          >sean@dellito.com</a
        >
        <!--end::Email-->
      </div>
      <!--end::Section-->

      <!--begin::Seperator-->
      <div class="separator separator-dashed mb-7"></div>
      <!--end::Seperator-->

      <!--begin::Section-->
      <div class="mb-7">
        <!--begin::Title-->
        <h5 class="mb-3">Product details</h5>
        <!--end::Title-->

        <!--begin::Details-->
        <div class="mb-0">
          <!--begin::Plan-->
          <span class="badge badge-light-info me-2">Basic Bundle</span>
          <!--end::Plan-->

          <!--begin::Price-->
          <span class="fw-semobold text-gray-600">$149.99 / Year</span>
          <!--end::Price-->
        </div>
        <!--end::Details-->
      </div>
      <!--end::Section-->

      <!--begin::Seperator-->
      <div class="separator separator-dashed mb-7"></div>
      <!--end::Seperator-->

      <!--begin::Section-->
      <div class="mb-10">
        <!--begin::Title-->
        <h5 class="mb-3">Payment Details</h5>
        <!--end::Title-->

        <!--begin::Details-->
        <div class="mb-0">
          <!--begin::Card info-->
          <div class="fw-semobold text-gray-600 d-flex align-items-center">
            Mastercard
            <img
              :src="getAssetPath('/media/svg/card-logos/mastercard.svg')"
              class="w-35px ms-2"
              alt=""
            />
          </div>
          <!--end::Card info-->

          <!--begin::Card expiry-->
          <div class="fw-semobold text-gray-600">Expires Dec 2024</div>
          <!--end::Card expiry-->
        </div>
        <!--end::Details-->
      </div>
      <!--end::Section-->

      <!--begin::Actions-->
      <div class="mb-0">
        <button
          type="submit"
          class="btn btn-primary"
          id="kt_subscriptions_create_button"
        >
          <!--begin::Indicator-->
          <span class="indicator-label">Create Subscription</span>
          <span class="indicator-progress"
            >Please wait...
            <span
              class="spinner-border spinner-border-sm align-middle ms-2"
            ></span
          ></span>
          <!--end::Indicator-->
        </button>
      </div>
      <!--end::Actions-->
    </div>
    <!--end::Card body-->
  </div>
  <!--end::Card-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-summary",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
