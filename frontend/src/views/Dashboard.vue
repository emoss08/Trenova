<template>
  <!--begin::Dashboard-->
  <div class="row g-5 g-xl-8">
    <div class="col-xl-4">
      <StatisticsWidget5
        widget-classes="card-xl-stretch mb-xl-8"
        svg-icon="media/icons/duotune/ecommerce/ecm002.svg"
        color="body-white"
        icon-color="primary"
        title="Shopping Cart"
        description="Lands, Houses, Ranchos, Farms"
      />
    </div>

    <div class="col-xl-4">
      <StatisticsWidget5
        widget-classes="card-xl-stretch mb-xl-8"
        svg-icon="media/icons/duotune/general/gen008.svg"
        color="primary"
        icon-color="white"
        title="Appartments"
        description="Flats, Shared Rooms, Duplex"
      />
    </div>

    <div class="col-xl-4">
      <StatisticsWidget5
        widget-classes="card-xl-stretch mb-xl-8"
        svg-icon="media/icons/duotune/arrows/arr070.svg"
        color="dark"
        icon-color="gray-100"
        title="Sales Stats"
        description="50% Increased for FY20"
      />
    </div>
  </div>

  <div class="row g-5 g-xl-8">
    <div class="col-xl-4">
      <ListsWidget1 widget-classes="card-xl-stretch mb-xl-8" />
    </div>

    <div class="col-xl-8">
      <TablesWidget5 widget-classes="card-xl-stretch mb-5 mb-xl-8" />
    </div>
  </div>

  <div class="row g-5 g-xl-8">
    <div class="col-xl-4">
      <ListsWidget3 widget-classes="card-xl-stretch mb-xl-8" />
    </div>

    <div class="col-xl-8">
      <ChartsWidget1
        widget-classes="card-xl-stretch mb-5 mb-xl-8"
        :height="400"
      />
    </div>
  </div>

  <div class="row g-5 g-xl-8">
    <div class="col-xl-6">
      <ListsWidget7 widget-classes="card-xl-stretch mb-xl-8" />
    </div>

    <div class="col-xl-6">
      <ListsWidget6 widget-classes="card-xl-stretch mb-5 mb-xl-8" />
    </div>
  </div>
  <!--end::Dashboard-->
</template>

<script lang="ts">
import { defineComponent } from "vue";
import StatisticsWidget5 from "@/components/widgets/statsistics/Widget5.vue";
import ListsWidget1 from "@/components/widgets/lists/Widget1.vue";
import TablesWidget5 from "@/components/widgets/tables/Widget5.vue";
import ListsWidget3 from "@/components/widgets/lists/Widget3.vue";
import ChartsWidget1 from "@/components/widgets/charts/Widget1.vue";
import ListsWidget7 from "@/components/widgets/lists/Widget7.vue";
import ListsWidget6 from "@/components/widgets/lists/Widget6.vue";

export default defineComponent({
  name: "dashboard-main",
  components: {
    StatisticsWidget5,
    ListsWidget1,
    TablesWidget5,
    ListsWidget3,
    ChartsWidget1,
    ListsWidget7,
    ListsWidget6,
  },
});
</script>
