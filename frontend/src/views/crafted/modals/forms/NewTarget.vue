<template>
  <KTModalCard
    title="New Target Modal Example"
    description="Click on the below buttons to launch <br/>a new target example."
    :image="getIllustrationsPath('17.png')"
    button-text="Add New Target"
    modal-id="kt_modal_new_target"
  ></KTModalCard>

  <KTNewTargetModal></KTNewTargetModal>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import KTModalCard from "@/components/cards/Card.vue";
import KTNewTargetModal from "@/components/modals/forms/NewTargetModal.vue";
import { getIllustrationsPath } from "@/core/helpers/assets";

export default defineComponent({
  name: "new-target",
  components: {
    KTModalCard,
    KTNewTargetModal,
  },
  setup() {
    return {
      getIllustrationsPath,
      getAssetPath,
    };
  },
});
</script>
