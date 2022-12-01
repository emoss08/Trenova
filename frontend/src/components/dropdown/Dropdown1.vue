<template>
  <!--begin::Menu 1-->
  <div
    class="menu menu-sub menu-sub-dropdown w-250px w-md-300px"
    data-kt-menu="true"
  >
    <!--begin::Header-->
    <div class="px-7 py-5">
      <div class="fs-5 text-dark fw-bold">Filter Options</div>
    </div>
    <!--end::Header-->

    <!--begin::Menu separator-->
    <div class="separator border-gray-200"></div>
    <!--end::Menu separator-->

    <!--begin::Form-->
    <div class="px-7 py-5">
      <!--begin::Input group-->
      <div class="mb-10">
        <!--begin::Label-->
        <label class="form-label fw-semobold">Status:</label>
        <!--end::Label-->

        <!--begin::Input-->
        <div>
          <select
            class="form-select form-select-solid select2-hidden-accessible"
            v-model="data.status"
          >
            <option label="Approved" value="1">Approved</option>
            <option label="Pending" value="2">Pending</option>
            <option label="In Process" value="3">In Process</option>
            <option label="Rejected" value="4">Rejected</option>
          </select>
        </div>
        <!--end::Input-->
      </div>
      <!--end::Input group-->

      <!--begin::Input group-->
      <div class="mb-10">
        <!--begin::Label-->
        <label class="form-label fw-semobold">Member Type:</label>
        <!--end::Label-->

        <!--begin::Options-->
        <div class="d-flex">
          <!--begin::Options-->
          <label
            class="form-check form-check-sm form-check-custom form-check-solid me-5"
          >
            <input
              class="form-check-input"
              type="checkbox"
              value="1"
              v-model="data.author"
            />
            <span class="form-check-label user-select-none"> Author </span>
          </label>
          <!--end::Options-->

          <!--begin::Options-->
          <label
            class="form-check form-check-sm form-check-custom form-check-solid"
          >
            <input
              class="form-check-input"
              type="checkbox"
              value="2"
              v-model="data.customer"
            />
            <span class="form-check-label user-select-none"> Customer </span>
          </label>
          <!--end::Options-->
        </div>
        <!--end::Options-->
      </div>
      <!--end::Input group-->

      <!--begin::Input group-->
      <div class="mb-10">
        <!--begin::Label-->
        <label class="form-label fw-semobold">Notifications:</label>
        <!--end::Label-->

        <!--begin::Switch-->
        <div
          class="form-check form-switch form-switch-sm form-check-custom form-check-solid"
        >
          <label class="form-check-label">
            <input
              class="form-check-input"
              type="checkbox"
              name="notifications"
              v-model="data.notifications"
            />
            <span class="form-check-label user-select-none">Enabled</span>
          </label>
        </div>
        <!--end::Switch-->
      </div>
      <!--end::Input group-->

      <!--begin::Actions-->
      <div class="d-flex justify-content-end">
        <button
          type="reset"
          class="btn btn-sm btn-light btn-active-light-primary me-2"
          data-kt-menu-dismiss="true"
        >
          Reset
        </button>

        <button
          type="submit"
          class="btn btn-sm btn-primary"
          data-kt-menu-dismiss="true"
        >
          Apply
        </button>
      </div>
      <!--end::Actions-->
    </div>
    <!--end::Form-->
  </div>
  <!--end::Menu 1-->
</template>

<script lang="ts">
import { defineComponent, ref } from "vue";

interface Filter {
  status: string;
  author: boolean;
  customer: boolean;
  notifications: boolean;
}

export default defineComponent({
  name: "dropdown-1",
  components: {},
  setup() {
    const data = ref<Filter>({
      status: "1",
      author: true,
      customer: true,
      notifications: true,
    });

    return {
      data,
    };
  },
});
</script>
