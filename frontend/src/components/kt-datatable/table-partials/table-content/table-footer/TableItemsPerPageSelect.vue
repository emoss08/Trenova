<template>
  <div
    class="col-sm-12 col-md-5 d-flex align-items-center justify-content-center justify-content-md-start"
  >
    <label for="items-per-page">
      <select
        class="form-select form-select-sm form-select-solid"
        v-if="itemsPerPageDropdownEnabled"
        v-model="itemsCountInTable"
        name="items-per-page"
        id="items-per-page"
      >
        <option :value="10">10</option>
        <option :value="25">25</option>
        <option :value="50">50</option>
      </select>
    </label>
  </div>
</template>

<script lang="ts">
import {
  defineComponent,
  ref,
  onMounted,
  type WritableComputedRef,
  computed,
} from "vue";

export default defineComponent({
  name: "table-items-per-page-select",
  components: {},
  props: {
    itemsPerPage: { type: Number, default: 10 },
    itemsPerPageDropdownEnabled: {
      type: Boolean,
      required: false,
      default: true,
    },
  },
  emits: ["update:itemsPerPage"],
  setup(props, { emit }) {
    const inputItemsPerPage = ref(10);

    onMounted(() => {
      inputItemsPerPage.value = props.itemsPerPage;
    });

    const itemsCountInTable: WritableComputedRef<number> = computed({
      get(): number {
        return props.itemsPerPage;
      },
      set(value: number): void {
        emit("update:itemsPerPage", value);
      },
    });

    return {
      itemsCountInTable,
    };
  },
});
</script>
