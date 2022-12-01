<template>
  <!--begin::Layout-->
  <div class="d-flex flex-column flex-lg-row">
    <!--begin::Content-->
    <div class="flex-lg-row-fluid me-lg-15 order-2 order-lg-1 mb-10 mb-lg-0">
      <ViewDetails></ViewDetails>

      <Events></Events>

      <Invoices></Invoices>
    </div>
    <!--end::Content-->

    <!--begin::Sidebar-->
    <div
      class="flex-column flex-lg-row-auto w-lg-250px w-xl-300px mb-10 order-1 order-lg-2"
    >
      <ViewSummary></ViewSummary>
    </div>
    <!--end::Sidebar-->
  </div>
  <!--end::Layout-->
</template>

<script lang="ts">
import { defineComponent } from "vue";
import ViewDetails from "@/components/subscriptions/view/Details.vue";
import Events from "@/components/subscriptions/view/Events.vue";
import Invoices from "@/components/subscriptions/view/Invoices.vue";
import ViewSummary from "@/components/subscriptions/view/Summary.vue";

export default defineComponent({
  name: "kt-view-subscription",
  components: {
    ViewDetails,
    Events,
    Invoices,
    ViewSummary,
  },
});
</script>
