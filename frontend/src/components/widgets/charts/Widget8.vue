<template>
  <!--begin::Charts Widget 8-->
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
          id="kt_charts_widget_8_year_btn"
          >Year</a
        >

        <a
          class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4 me-1"
          id="kt_charts_widget_8_month_btn"
          >Month</a
        >

        <a
          class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4"
          id="kt_charts_widget_8_week_btn"
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
        type="area"
        :options="chart"
        :series="series"
      ></apexchart>
      <!--end::Chart-->
    </div>
    <!--end::Body-->
  </div>
  <!--end::Charts Widget 8-->
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
  },
  components: {},
  setup() {
    const chartRef = ref<typeof VueApexCharts | null>(null);
    let chart: ApexOptions = {};
    const store = useThemeStore();

    const series = [
      {
        name: "Net Profit",
        data: [30, 30, 50, 50, 35, 35],
      },
      {
        name: "Revenue",
        data: [55, 20, 20, 20, 70, 70],
      },
      {
        name: "Expenses",
        data: [60, 60, 40, 40, 30, 30],
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
  const strokeColor = getCSSVariableValue("--kt-gray-300");

  const color1 = getCSSVariableValue("--kt-warning");
  const color1Light = getCSSVariableValue("--kt-warning-light");

  const color2 = getCSSVariableValue("--kt-success");
  const color2Light = getCSSVariableValue("--kt-success-light");

  const color3 = getCSSVariableValue("--kt-primary");
  const color3Light = getCSSVariableValue("--kt-primary-light");

  return {
    chart: {
      fontFamily: "inherit",
      type: "area",
      toolbar: {
        show: false,
      },
      zoom: {
        enabled: false,
      },
      sparkline: {
        enabled: true,
      },
    },
    plotOptions: {},
    legend: {
      show: false,
    },
    dataLabels: {
      enabled: false,
    },
    fill: {
      type: "solid",
      opacity: 1,
    },
    stroke: {
      curve: "smooth",
      show: true,
      width: 2,
      colors: [color1, color2, color3],
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
        show: false,
        style: {
          colors: labelColor,
          fontSize: "12px",
        },
      },
      crosshairs: {
        show: false,
        position: "front",
        stroke: {
          color: strokeColor,
          width: 1,
          dashArray: 3,
        },
      },
      tooltip: {
        enabled: false,
      },
    },
    yaxis: {
      labels: {
        show: false,
        style: {
          colors: labelColor,
          fontSize: "12px",
        },
      },
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
    colors: [color1Light, color2Light, color3Light],
    grid: {
      borderColor: borderColor,
      strokeDashArray: 4,
      padding: {
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
      },
    },
    markers: {
      colors: [color1, color2, color3],
      strokeColors: [color1, color2, color3],
      strokeWidth: 3,
    },
  };
};
</script>
