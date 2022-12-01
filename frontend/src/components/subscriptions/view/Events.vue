<template>
  <div class="card card-flush pt-3 mb-5 mb-xl-10">
    <!--begin::Card header-->
    <div class="card-header">
      <!--begin::Card title-->
      <div class="card-title">
        <h2>Recent Events</h2>
      </div>
      <!--end::Card title-->
      <!--begin::Card toolbar-->
      <div class="card-toolbar">
        <a href="#" class="btn btn-light-primary">View All Events</a>
      </div>
      <!--end::Card toolbar-->
    </div>
    <!--end::Card header-->
    <!--begin::Card body-->
    <div class="card-body pt-0">
      <!--begin::Table wrapper-->
      <div class="table-responsive">
        <!--begin::Table-->
        <table
          class="table align-middle table-row-dashed fs-6 text-gray-600 fw-semobold gy-5"
          id="kt_table_customers_events"
        >
          <!--begin::Table body-->
          <tbody>
            <!--begin::Table row-->
            <tr>
              <!--begin::Event=-->
              <td class="min-w-400px">
                Invoice
                <a
                  href="#"
                  class="fw-bold text-gray-800 text-hover-primary me-1"
                  >7786-3830</a
                >status has changed from
                <span class="badge badge-light-primary me-1">In Transit</span>to
                <span class="badge badge-light-success">Approved</span>
              </td>
              <!--end::Event=-->
              <!--begin::Timestamp=-->
              <td class="pe-0 text-gray-600 text-end min-w-200px">
                25 Oct 2021, 9:23 pm
              </td>
              <!--end::Timestamp=-->
            </tr>
            <!--end::Table row-->
            <!--begin::Table row-->
            <tr>
              <!--begin::Event=-->
              <td class="min-w-400px">
                Invoice
                <a
                  href="#"
                  class="fw-bold text-gray-800 text-hover-primary me-1"
                  >9357-7929</a
                >status has changed from
                <span class="badge badge-light-info me-1">In Progress</span>to
                <span class="badge badge-light-primary">In Transit</span>
              </td>
              <!--end::Event=-->
              <!--begin::Timestamp=-->
              <td class="pe-0 text-gray-600 text-end min-w-200px">
                10 Mar 2021, 5:20 pm
              </td>
              <!--end::Timestamp=-->
            </tr>
            <!--end::Table row-->
            <!--begin::Table row-->
            <tr>
              <!--begin::Event=-->
              <td class="min-w-400px">
                <a
                  href="#"
                  class="fw-bold text-gray-800 text-hover-primary me-1"
                  >Brian Cox</a
                >has made payment to
                <a href="#" class="fw-bold text-gray-800 text-hover-primary"
                  >7277-8716</a
                >
              </td>
              <!--end::Event=-->
              <!--begin::Timestamp=-->
              <td class="pe-0 text-gray-600 text-end min-w-200px">
                10 Mar 2021, 5:20 pm
              </td>
              <!--end::Timestamp=-->
            </tr>
            <!--end::Table row-->
            <!--begin::Table row-->
            <tr>
              <!--begin::Event=-->
              <td class="min-w-400px">
                <a
                  href="#"
                  class="fw-bold text-gray-800 text-hover-primary me-1"
                  >Melody Macy</a
                >has made payment to
                <a href="#" class="fw-bold text-gray-800 text-hover-primary"
                  >2516-2975</a
                >
              </td>
              <!--end::Event=-->
              <!--begin::Timestamp=-->
              <td class="pe-0 text-gray-600 text-end min-w-200px">
                24 Jun 2021, 5:30 pm
              </td>
              <!--end::Timestamp=-->
            </tr>
            <!--end::Table row-->
            <!--begin::Table row-->
            <tr>
              <!--begin::Event=-->
              <td class="min-w-400px">
                Invoice
                <a
                  href="#"
                  class="fw-bold text-gray-800 text-hover-primary me-1"
                  >4464-4371</a
                >is <span class="badge badge-light-info">In Progress</span>
              </td>
              <!--end::Event=-->
              <!--begin::Timestamp=-->
              <td class="pe-0 text-gray-600 text-end min-w-200px">
                21 Feb 2021, 5:30 pm
              </td>
              <!--end::Timestamp=-->
            </tr>
            <!--end::Table row-->
          </tbody>
          <!--end::Table body-->
        </table>
        <!--end::Table-->
      </div>
      <!--end::Table wrapper-->
    </div>
    <!--end::Card body-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-events",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
