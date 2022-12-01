<template>
  <!--begin::Customer-->
  <div class="card card-flush pt-3 mb-5 mb-lg-10">
    <!--begin::Card header-->
    <div class="card-header">
      <!--begin::Card title-->
      <div class="card-title">
        <h2 class="fw-bold">Customer</h2>
      </div>
      <!--begin::Card title-->
    </div>
    <!--end::Card header-->

    <!--begin::Card body-->
    <div class="card-body pt-0">
      <!--begin::Description-->
      <div class="text-gray-500 fw-semobold fs-5 mb-5">
        Select or add a customer to a subscription:
      </div>
      <!--end::Description-->

      <!--begin::Selected customer-->
      <div class="d-flex align-items-center p-3 mb-2">
        <!--begin::Avatar-->
        <div class="symbol symbol-60px symbol-circle me-3">
          <img alt="Pic" :src="getAssetPath('/media/avatars/300-5.jpg')" />
        </div>
        <!--end::Avatar-->

        <!--begin::Info-->
        <div class="d-flex flex-column">
          <!--begin::Name-->
          <a href="#" class="fs-4 fw-bold text-gray-900 text-hover-primary me-2"
            >Sean Bean</a
          >
          <!--end::Name-->

          <!--begin::Email-->
          <a href="#" class="fw-semobold text-gray-600 text-hover-primary"
            >sean@dellito.com</a
          >
          <!--end::Email-->
        </div>
        <!--end::Info-->
      </div>
      <!--end::Selected customer-->

      <!--begin::Customer add buttons-->
      <div class="mb-7 d-none">
        <a
          href="#"
          class="btn btn-primary"
          data-bs-toggle="modal"
          data-bs-target="#kt_modal_customer_search"
          >Select Customer</a
        >

        <span class="fw-bold text-gray-500 mx-2">or</span>

        <router-link
          to="/customers/list"
          class="btn btn-light-primary"
          data-bs-toggle="modal"
          data-bs-target="#kt_modal_users_search"
          >Add New Customer</router-link
        >
      </div>
      <!--end::Customer add buttons-->

      <!--begin::Customer change button-->
      <div class="mb-10">
        <a
          href="#"
          class="btn btn-light-primary"
          data-bs-toggle="modal"
          data-bs-target="#kt_modal_customer_search"
          >Change Customer</a
        >
      </div>
      <!--end::Customer change button-->

      <Notice
        classes="rounded-3"
        color="primary"
        title="This is a very important privacy notice!"
        :body="body"
      ></Notice>
    </div>
    <!--end::Card body-->
  </div>
  <!--end::Customer-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref } from "vue";
import Notice from "@/components/Notice.vue";

export default defineComponent({
  name: "kt-customer",
  components: {
    Notice,
  },
  setup() {
    const body = ref(
      `Writing headlines for blog posts is much science and probably cool audience. <a href="#" class="fw-bold">Learn more</a>.`
    );

    return {
      body,
      getAssetPath,
    };
  },
});
</script>
