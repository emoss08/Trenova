<template>
  <KTModalCard
    title="New Card Modal Example"
    description="Click on the below buttons to launch <br/>a new card example."
    :image="getIllustrationsPath('6.png')"
    button-text="Add New Card"
    modal-id="kt_modal_new_card"
  ></KTModalCard>

  <KTNewCardModal></KTNewCardModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import KTModalCard from "@/components/cards/Card.vue";
import KTNewCardModal from "@/components/modals/forms/NewCardModal.vue";
import { getIllustrationsPath } from "@/core/helpers/assets";

export default defineComponent({
  name: "new-card",
  components: {
    KTModalCard,
    KTNewCardModal,
  },
  setup() {
    return {
      getIllustrationsPath,
    };
  },
});
</script>
