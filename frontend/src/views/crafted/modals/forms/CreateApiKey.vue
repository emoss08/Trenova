<template>
  <KTModalCard
    title="Create API Key Modal Example"
    description="Click on the below buttons to launch <br/>a new API Key creation example."
    :image="getIllustrationsPath('16.png')"
    button-text="Create API Key"
    modal-id="kt_modal_create_api_key"
  ></KTModalCard>

  <KTCreateAPIKeyModal></KTCreateAPIKeyModal>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import KTModalCard from "@/components/cards/Card.vue";
import KTCreateAPIKeyModal from "@/components/modals/forms/CreateAPIKeyModal.vue";
import { getIllustrationsPath } from "@/core/helpers/assets";

export default defineComponent({
  name: "create-api-key",
  components: {
    KTModalCard,
    KTCreateAPIKeyModal,
  },
  setup() {
    return {
      getIllustrationsPath,
      getAssetPath,
    };
  },
});
</script>
