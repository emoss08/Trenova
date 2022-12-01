<template>
  <!--begin::Layout-->
  <div class="d-flex flex-column flex-lg-row">
    <!--begin::Content-->
    <div class="flex-lg-row-fluid me-lg-15 order-2 order-lg-1 mb-10 mb-lg-0">
      <!--begin::Form-->
      <form class="form" action="#" id="kt_subscriptions_create_new">
        <Customer></Customer>

        <Products></Products>

        <PaymentMethod></PaymentMethod>

        <!--        <Advanced></Advanced>-->
      </form>
      <!--end::Form-->
    </div>
    <!--end::Content-->

    <!--begin::Sidebar-->
    <div
      class="flex-column flex-lg-row-auto w-100 w-lg-250px w-xl-300px mb-10 order-1 order-lg-2"
    >
      <AddSummary></AddSummary>
    </div>
    <!--end::Sidebar-->
  </div>
  <!--end::Layout-->

  <!--begin::Modals-->
  <NewCardModal></NewCardModal>
  <CreateAccountModal></CreateAccountModal>
  <!--end::Modals-->
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Customer from "@/components/subscriptions/add/Customer.vue";
import Products from "@/components/subscriptions/add/Products.vue";
import PaymentMethod from "@/components/subscriptions/add/PaymentMethod.vue";
import AddSummary from "@/components/subscriptions/add/Summary.vue";
import NewCardModal from "@/components/modals/forms/NewCardModal.vue";
import CreateAccountModal from "@/components/modals/wizards/CreateAccountModal.vue";

export default defineComponent({
  name: "kt-add-subscription",
  components: {
    AddSummary,
    Customer,
    Products,
    PaymentMethod,
    NewCardModal,
    CreateAccountModal,
  },
});
</script>
