<template>
  <KTModalCard
    title="View Users Modal Example"
    description="Click on the below buttons to launch <br/>user lists example."
    :image="getIllustrationsPath('10.png')"
    button-text="View Users"
    modal-id="kt_modal_view_users"
  ></KTModalCard>

  <KTViewUsersModal></KTViewUsersModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import KTModalCard from "@/components/cards/Card.vue";
import KTViewUsersModal from "@/components/modals/general/ViewUsersModal.vue";
import { getIllustrationsPath } from "@/core/helpers/assets";

export default defineComponent({
  name: "view-users",
  components: {
    KTModalCard,
    KTViewUsersModal,
  },
  setup() {
    return {
      getIllustrationsPath,
    };
  },
});
</script>
