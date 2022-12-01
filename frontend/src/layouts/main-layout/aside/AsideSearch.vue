<template>
  <!--begin::Search-->
  <div
    ref="searchRef"
    class="d-flex align-items-center w-100"
    data-kt-search="true"
    data-kt-search-keypress="true"
    data-kt-search-min-length="2"
    data-kt-search-enter="enter"
    data-kt-search-layout="menu"
    data-kt-search-responsive="false"
    data-kt-menu-trigger="auto"
    data-kt-menu-permanent="true"
    data-kt-menu-placement="bottom-start"
  >
    <InlineForm />

    <!--begin::Menu-->
    <div
      data-kt-search-element="content"
      class="menu menu-sub menu-sub-dropdown w-300px w-md-350px py-7 px-7 overflow-hidden"
    >
      <!--begin::Wrapper-->
      <div data-kt-search-element="wrapper">
        <Results />

        <AsideMain />

        <Empty />
      </div>
      <!--end::Wrapper-->

      <AdvancedOptions />

      <Preferences />
    </div>
    <!--end::Menu-->
  </div>
  <!--end::Search-->
</template>

<script lang="ts">
import { defineComponent, onMounted, nextTick, ref } from "vue";
import Results from "@/layouts/main-layout/aside/search/Results.vue";
import AsideMain from "@/layouts/main-layout/aside/search/Main.vue";
import Empty from "@/layouts/main-layout/aside/search/Empty.vue";
import AdvancedOptions from "@/layouts/main-layout/aside/search/AdvancedOptions.vue";
import Preferences from "@/layouts/main-layout/aside/search/Preferences.vue";
import InlineForm from "@/layouts/main-layout/aside/search/InlineForm.vue";
import { SearchComponent } from "@/assets/ts/components";

export default defineComponent({
  name: "kt-search",
  components: {
    Results,
    AsideMain,
    Empty,
    AdvancedOptions,
    Preferences,
    InlineForm,
  },
  setup() {
    const searchRef = ref<HTMLElement | null>(null);

    const processs = (search: SearchComponent) => {
      setTimeout(function () {
        const number = Math.floor(Math.random() * 6) + 1;

        // Hide recently viewed
        search.suggestionElement.classList.add("d-none");

        if (number === 3) {
          // Hide results
          search.resultsElement.classList.add("d-none");
          // Show empty message
          search.emptyElement.classList.remove("d-none");
        } else {
          // Show results
          search.resultsElement.classList.remove("d-none");
          // Hide empty message
          search.emptyElement.classList.add("d-none");
        }

        // Complete search
        search.complete();
      }, 1500);
    };

    const clear = (search: SearchComponent) => {
      // Show recently viewed
      search.suggestionElement.classList.remove("d-none");
      // Hide results
      search.resultsElement.classList.add("d-none");
      // Hide empty message
      search.emptyElement.classList.add("d-none");
    };

    onMounted(() => {
      nextTick(() => {
        // Initialize search handler
        const searchObject = SearchComponent.createInsance("[data-kt-search]");

        // Search handler
        searchObject?.on("kt.search.process", processs);

        // Clear handler
        searchObject?.on("kt.search.cleared", clear);
      });
    });

    return {
      searchRef,
    };
  },
});
</script>
