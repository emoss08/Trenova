<template>
  <KTModalCard
    title="Create App Modal Example"
    description="Click on the below buttons to launch <br/>create app modal example."
    :image="getIllustrationsPath('15.png')"
    button-text="Create App"
    modal-id="kt_modal_create_app"
  ></KTModalCard>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import KTModalCard from "@/components/cards/Card.vue";
import { getIllustrationsPath } from "@/core/helpers/assets";

export default defineComponent({
  name: "create-app",
  components: {
    KTModalCard,
  },
  setup() {
    return {
      getIllustrationsPath,
    };
  },
});
</script>
