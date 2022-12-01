<template>
  <KTModalsCard
    title="Share & Earn Modal Example"
    description="Click on the below buttons to launch <br/>a share & earn example."
    :image="getIllustrationsPath('9.png')"
    button-text="Share & Earn"
    modal-id="kt_modal_share_earn"
  ></KTModalsCard>

  <KTShareAndEarnModal></KTShareAndEarnModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import KTModalsCard from "@/components/cards/Card.vue";
import KTShareAndEarnModal from "@/components/modals/general/ShareAndEarnModal.vue";
import { getIllustrationsPath } from "@/core/helpers/assets";

export default defineComponent({
  name: "share-and-earn",
  components: {
    KTModalsCard,
    KTShareAndEarnModal,
  },
  setup() {
    return {
      getIllustrationsPath,
    };
  },
});
</script>
