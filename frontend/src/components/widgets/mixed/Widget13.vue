<template>
  <!--begin::Mixed Widget 13-->
  <div
    :class="widgetClasses"
    class="card theme-dark-bg-body"
    :style="`background-color: ${widgetColor}`"
  >
    <!--begin::Body-->
    <div class="card-body d-flex flex-column">
      <!--begin::Wrapper-->
      <div class="d-flex flex-column flex-grow-1">
        <!--begin::Title-->
        <a href="#" class="text-dark text-hover-primary fw-bold fs-3"
          >Contributors</a
        >
        <!--end::Title-->

        <!--begin::Chart-->
        <apexchart
          ref="chartRef"
          class="mixed-widget-14-chart"
          :options="chart"
          :series="series"
          :height="chartHeight"
          type="bar"
        ></apexchart>
        <!--end::Chart-->
      </div>
      <!--end::Wrapper-->

      <!--begin::Stats-->
      <div class="pt-5">
        <!--begin::Number-->
        <span class="text-dark fw-bold fs-3x me-2 lh-0">47</span>
        <!--end::Number-->

        <!--begin::Text-->
        <span class="text-dark fw-bold fs-6 lh-0">- 12% this week</span>
        <!--end::Text-->
      </div>
      <!--end::Stats-->
    </div>
  </div>
  <!--end::Mixed Widget 13-->
</template>

<script lang="ts">
import { defineComponent, ref, onBeforeMount, computed, watch } from "vue";
import type VueApexCharts from "vue3-apexcharts";
import type { ApexOptions } from "apexcharts";
import { useThemeStore } from "@/stores/theme";

export default defineComponent({
  name: "widget-13",
  props: {
    widgetClasses: String,
    widgetColor: String,
    chartHeight: String,
  },
  setup(props) {
    const chartRef = ref<typeof VueApexCharts | null>(null);
    let chart: ApexOptions = {};
    const store = useThemeStore();

    const series = [
      {
        name: "Inflation",
        data: [
          1, 2.1, 1, 2.1, 4.1, 6.1, 4.1, 4.1, 2.1, 4.1, 2.1, 3.1, 1, 1, 2.1,
        ],
      },
    ];

    const themeMode = computed(() => {
      return store.mode;
    });

    onBeforeMount(() => {
      Object.assign(chart, chartOptions(props.chartHeight));
    });

    const refreshChart = () => {
      if (!chartRef.value) {
        return;
      }

      Object.assign(chart, chartOptions(props.chartHeight));

      chartRef.value.refresh();
    };

    watch(themeMode, () => {
      refreshChart();
    });

    return {
      chart,
      series,
      chartRef,
      refreshChart,
    };
  },
});

const chartOptions = (chartHeight: string = "auto"): ApexOptions => {
  return {
    chart: {
      fontFamily: "inherit",
      height: chartHeight,
      type: "bar",
      toolbar: {
        show: false,
      },
    },
    grid: {
      show: false,
      padding: {
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
      },
    },
    colors: ["#ffffff"],
    plotOptions: {
      bar: {
        dataLabels: {
          position: "top", // top, center, bottom
        },
        columnWidth: "20%",
      },
    },
    dataLabels: {
      enabled: false,
      formatter: function (val) {
        return val + "%";
      },
      offsetY: -20,
      style: {
        fontSize: "12px",
        colors: ["#304758"],
      },
    },
    xaxis: {
      labels: {
        show: false,
      },
      categories: [
        "Jan",
        "Feb",
        "Mar",
        "Apr",
        "May",
        "Jun",
        "Jul",
        "Aug",
        "Sep",
        "Oct",
        "Nov",
        "Dec",
        "Jan",
        "Feb",
        "Mar",
      ],
      position: "top",
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      crosshairs: {
        show: false,
      },
      tooltip: {
        enabled: false,
      },
    },
    yaxis: {
      show: false,
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      labels: {
        show: false,
        formatter: function (val) {
          return val + "%";
        },
      },
    },
  };
};
</script>
