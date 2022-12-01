<template>
  <KTModalCard
    title="New Address Modal Example"
    description="Click on the below buttons to launch <br/>a new address example."
    :image="getIllustrationsPath('19.png')"
    button-text="Add New Address"
    modal-id="kt_modal_new_address"
  ></KTModalCard>

  <KTNewAddressModal></KTNewAddressModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import KTModalCard from "@/components/cards/Card.vue";
import KTNewAddressModal from "@/components/modals/forms/NewAddressModal.vue";
import { getIllustrationsPath } from "@/core/helpers/assets";

export default defineComponent({
  name: "new-address",
  components: {
    KTModalCard,
    KTNewAddressModal,
  },
  setup() {
    return {
      getIllustrationsPath,
    };
  },
});
</script>
