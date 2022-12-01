<template>
  <div
    class="col-sm-12 col-md-7 d-flex align-items-center justify-content-center justify-content-md-end"
  >
    <div class="dataTables_paginate paging_simple_numbers">
      <ul class="pagination">
        <li
          class="paginate_button page-item"
          :class="{ disabled: isInFirstPage }"
          :style="{ cursor: !isInFirstPage ? 'pointer' : 'auto' }"
        >
          <a class="page-link" @click="onClickFirstPage">
            <span class="svg-icon">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr079.svg')"
              />
            </span>
          </a>
        </li>

        <li
          class="paginate_button page-item"
          :class="{ disabled: isInFirstPage }"
          :style="{ cursor: !isInFirstPage ? 'pointer' : 'auto' }"
        >
          <a class="page-link" @click="onClickPreviousPage">
            <span class="svg-icon">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr074.svg')"
              />
            </span>
          </a>
        </li>

        <li
          v-for="(page, i) in pages"
          class="paginate_button page-item"
          :class="{
            active: isPageActive(page.name),
          }"
          :style="{ cursor: !page.isDisabled ? 'pointer' : 'auto' }"
          :key="i"
        >
          <a class="page-link" @click="onClickPage(page.name)">
            {{ page.name }}
          </a>
        </li>

        <li
          class="paginate_button page-item"
          :class="{ disabled: isInLastPage }"
          :style="{ cursor: !isInLastPage ? 'pointer' : 'auto' }"
        >
          <a class="paginate_button page-link" @click="onClickNextPage">
            <span class="svg-icon">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr071.svg')"
              />
            </span>
          </a>
        </li>

        <li
          class="paginate_button page-item"
          :class="{ disabled: isInLastPage }"
          :style="{ cursor: !isInLastPage ? 'pointer' : 'auto' }"
        >
          <a class="paginate_button page-link" @click="onClickLastPage">
            <span class="svg-icon">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr080.svg')"
              />
            </span>
          </a>
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, computed } from "vue";

export default defineComponent({
  name: "table-pagination",
  props: {
    maxVisibleButtons: {
      type: Number,
      required: false,
      default: 5,
    },
    totalPages: {
      type: Number,
      required: true,
    },
    total: {
      type: Number,
      required: true,
    },
    perPage: {
      type: Number,
      required: true,
    },
    currentPage: {
      type: Number,
      required: true,
    },
  },
  emits: ["page-change"],
  setup(props, { emit }) {
    const startPage = computed(() => {
      if (
        props.totalPages < props.maxVisibleButtons ||
        props.currentPage === 1 ||
        props.currentPage <= Math.floor(props.maxVisibleButtons / 2) ||
        (props.currentPage + 2 > props.totalPages &&
          props.totalPages === props.maxVisibleButtons)
      ) {
        return 1;
      }

      if (props.currentPage + 2 > props.totalPages) {
        return props.totalPages - props.maxVisibleButtons + 1;
      }

      return props.currentPage - 2;
    });

    const endPage = computed(() => {
      return Math.min(
        startPage.value + props.maxVisibleButtons - 1,
        props.totalPages
      );
    });

    const pages = computed(() => {
      const range: Array<{
        name: number;
        isDisabled: boolean;
      }> = [];

      for (let i = startPage.value; i <= endPage.value; i += 1) {
        range.push({
          name: i,
          isDisabled: i === props.currentPage,
        });
      }

      return range;
    });

    const isInFirstPage = computed(() => {
      return props.currentPage === 1;
    });
    const isInLastPage = computed(() => {
      return props.currentPage === props.totalPages;
    });

    const onClickFirstPage = () => {
      emit("page-change", 1);
    };
    const onClickPreviousPage = () => {
      emit("page-change", props.currentPage - 1);
    };
    const onClickPage = (page: number) => {
      emit("page-change", page);
    };
    const onClickNextPage = () => {
      emit("page-change", props.currentPage + 1);
    };
    const onClickLastPage = () => {
      emit("page-change", props.totalPages);
    };
    const isPageActive = (page: number) => {
      return props.currentPage === page;
    };

    return {
      startPage,
      endPage,
      pages,
      isInFirstPage,
      isInLastPage,
      onClickFirstPage,
      onClickPreviousPage,
      onClickPage,
      onClickNextPage,
      onClickLastPage,
      isPageActive,
      getAssetPath,
    };
  },
});
</script>
