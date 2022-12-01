<template>
  <!--begin::Charts Widget 2-->
  <div :class="widgetClasses" class="card">
    <!--begin::Header-->
    <div class="card-header border-0 pt-5">
      <h3 class="card-title align-items-start flex-column">
        <span class="card-label fw-bold fs-3 mb-1">Recent Orders</span>

        <span class="text-muted fw-semobold fs-7"
          >More than 500 new orders</span
        >
      </h3>

      <!--begin::Toolbar-->
      <div class="card-toolbar" data-kt-buttons="true">
        <a
          class="btn btn-sm btn-color-muted btn-active btn-active-primary active px-4 me-1"
          id="kt_charts_widget_2_year_btn"
          >Year</a
        >

        <a
          class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4 me-1"
          id="kt_charts_widget_2_month_btn"
          >Month</a
        >

        <a
          class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4"
          id="kt_charts_widget_2_week_btn"
          >Week</a
        >
      </div>
      <!--end::Toolbar-->
    </div>
    <!--end::Header-->

    <!--begin::Body-->
    <div class="card-body">
      <!--begin::Chart-->
      <apexchart
        ref="chartRef"
        type="bar"
        :options="chart"
        :series="series"
        :height="height"
      ></apexchart>
      <!--end::Chart-->
    </div>
    <!--end::Body-->
  </div>
  <!--end::Charts Widget 2-->
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch, onBeforeMount } from "vue";
import { useThemeStore } from "@/stores/theme";
import type { ApexOptions } from "apexcharts";
import { getCSSVariableValue } from "@/assets/ts/_utils";
import type VueApexCharts from "vue3-apexcharts";

export default defineComponent({
  name: "widget-1",
  props: {
    widgetClasses: String,
    height: Number,
  },
  components: {},
  setup() {
    const chartRef = ref<typeof VueApexCharts | null>(null);
    let chart: ApexOptions = {};
    const store = useThemeStore();

    const series = [
      {
        name: "Net Profit",
        data: [44, 55, 57, 56, 61, 58],
      },
      {
        name: "Revenue",
        data: [76, 85, 101, 98, 87, 105],
      },
    ];

    const themeMode = computed(() => {
      return store.mode;
    });

    onBeforeMount(() => {
      Object.assign(chart, chartOptions());
    });

    const refreshChart = () => {
      if (!chartRef.value) {
        return;
      }

      Object.assign(chart, chartOptions());

      chartRef.value.refresh();
    };

    watch(themeMode, () => {
      refreshChart();
    });

    return {
      chart,
      series,
      chartRef,
    };
  },
});

const chartOptions = (): ApexOptions => {
  const labelColor = getCSSVariableValue("--kt-gray-500");
  const borderColor = getCSSVariableValue("--kt-gray-200");
  const baseColor = getCSSVariableValue("--kt-warning");
  const secondaryColor = getCSSVariableValue("--kt-gray-300");

  return {
    chart: {
      fontFamily: "inherit",
      type: "bar",
      toolbar: {
        show: false,
      },
    },
    plotOptions: {
      bar: {
        horizontal: false,
        columnWidth: "30%",
        borderRadius: 5,
      },
    },
    legend: {
      show: false,
    },
    dataLabels: {
      enabled: false,
    },
    stroke: {
      show: true,
      width: 2,
      colors: ["transparent"],
    },
    xaxis: {
      categories: ["Feb", "Mar", "Apr", "May", "Jun", "Jul"],
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      labels: {
        style: {
          colors: labelColor,
          fontSize: "12px",
        },
      },
    },
    yaxis: {
      labels: {
        style: {
          colors: labelColor,
          fontSize: "12px",
        },
      },
    },
    fill: {
      opacity: 1,
    },
    states: {
      normal: {
        filter: {
          type: "none",
          value: 0,
        },
      },
      hover: {
        filter: {
          type: "none",
          value: 0,
        },
      },
      active: {
        allowMultipleDataPointsSelection: false,
        filter: {
          type: "none",
          value: 0,
        },
      },
    },
    tooltip: {
      style: {
        fontSize: "12px",
      },
      y: {
        formatter: function (val) {
          return "$" + val + " thousands";
        },
      },
    },
    colors: [baseColor, secondaryColor],
    grid: {
      borderColor: borderColor,
      strokeDashArray: 4,
      yaxis: {
        lines: {
          show: true,
        },
      },
    },
  };
};
</script>
