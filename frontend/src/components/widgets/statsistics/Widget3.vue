<template>
  <!--begin::Statistics Widget 3-->
  <div :class="widgetClasses" class="card">
    <!--begin::Body-->
    <div class="card-body d-flex flex-column p-0">
      <div class="d-flex flex-stack flex-grow-1 card-p">
        <div class="d-flex flex-column me-2">
          <a href="#" class="text-dark text-hover-primary fw-bold fs-3">{{
            title
          }}</a>

          <span class="text-muted fw-semobold mt-1">{{ description }}</span>
        </div>

        <span class="symbol symbol-50px">
          <span
            :class="`bg-light-${color} text-${color}`"
            class="symbol-label fs-5 fw-bold"
            >{{ change }}</span
          >
        </span>
      </div>

      <!--begin::Chart-->
      <apexchart
        ref="chartRef"
        class="statistics-widget-3-chart card-rounded-bottom"
        :options="chart"
        :series="series"
        :height="height"
        type="area"
      ></apexchart>
      <!--end::Chart-->
    </div>
    <!--end::Body-->
  </div>
  <!--end::Statistics Widget 3-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref, computed, watch, onBeforeMount } from "vue";
import { useThemeStore } from "@/stores/theme";
import type { ApexOptions } from "apexcharts";
import { getCSSVariableValue } from "@/assets/ts/_utils";
import type VueApexCharts from "vue3-apexcharts";

export default defineComponent({
  name: "kt-widget-3",
  props: {
    widgetClasses: String,
    title: String,
    description: String,
    change: String,
    color: String,
    height: String,
  },
  components: {},
  setup(props) {
    const chartRef = ref<typeof VueApexCharts | null>(null);
    let chart: ApexOptions = {};
    const store = useThemeStore();

    const series = [
      {
        name: "Net Profit",
        data: [30, 45, 32, 70, 40],
      },
    ];

    const themeMode = computed(() => {
      return store.mode;
    });

    onBeforeMount(() => {
      Object.assign(chart, chartOptions(props.color, props.height));
    });

    const refreshChart = () => {
      if (!chartRef.value) {
        return;
      }

      Object.assign(chart, chartOptions(props.color, props.height));

      chartRef.value.refresh();
    };

    watch(themeMode, () => {
      refreshChart();
    });

    return {
      chart,
      series,
      chartRef,
      getAssetPath,
    };
  },
});

const chartOptions = (
  color: string = "primary",
  height: string = "auto"
): ApexOptions => {
  const labelColor = getCSSVariableValue("--kt-gray-800");
  const baseColor = getCSSVariableValue(`--kt-${color}`);
  const lightColor = getCSSVariableValue(`--kt-${color}-light`);

  return {
    chart: {
      fontFamily: "inherit",
      type: "area",
      height: height,
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
    stroke: {
      curve: "smooth",
      show: true,
      width: 3,
      colors: [baseColor],
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
          color: "#E4E6EF",
          width: 1,
          dashArray: 3,
        },
      },
      tooltip: {
        enabled: false,
      },
    },
    yaxis: {
      min: 0,
      max: 80,
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
    fill: {
      type: "gradient",
      gradient: {
        stops: [0, 100],
      },
    },
    colors: [baseColor],
    markers: {
      colors: [baseColor],
      strokeColors: [lightColor],
      strokeWidth: 3,
    },
  };
};
</script>
