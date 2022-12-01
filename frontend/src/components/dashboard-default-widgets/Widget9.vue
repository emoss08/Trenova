<template>
  <div class="card card-flush" :class="className">
    <!--begin::Header-->
    <div class="card-header pt-5">
      <!--begin::Title-->
      <h3 class="card-title align-items-start flex-column">
        <span class="card-label fw-bold text-dark">Performance</span>
        <span class="text-gray-400 mt-1 fw-semibold fs-6"
          >1,046 Inbound Calls today</span
        >
      </h3>
      <!--end::Title-->
      <!--begin::Toolbar-->
      <div class="card-toolbar">
        <div class="btn btn-sm btn-light d-flex align-items-center px-4">
          <div class="text-gray-600 fw-bold">1 Jul 2022 - 31 Jul 2022</div>
          <span class="svg-icon svg-icon-1 ms-2 me-0">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen014.svg')"
            />
          </span>
        </div>
      </div>
      <!--end::Toolbar-->
    </div>
    <!--end::Header-->
    <!--begin::Card body-->
    <div class="card-body d-flex align-items-end p-0">
      <!--begin::Chart-->
      <apexchart
        class="min-h-auto w-100 ps-4 pe-6"
        ref="chartRef"
        :options="chart"
        :series="series"
        :height="height"
      ></apexchart>
      <!--end::Chart-->
    </div>
    <!--end::Card body-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { computed, defineComponent, onMounted, ref, watch } from "vue";
import type { ApexOptions } from "apexcharts";
import { getCSSVariableValue } from "@/assets/ts/_utils";
import type VueApexCharts from "vue3-apexcharts";
import { useThemeStore } from "@/stores/theme";

export default defineComponent({
  name: "default-dashboard-widget-9",
  components: {},
  props: {
    className: { type: String, required: false },
    height: { type: Number, required: true },
  },
  setup(props) {
    const chartRef = ref<typeof VueApexCharts | null>(null);
    let chart: ApexOptions = {};
    const store = useThemeStore();

    const series = [
      {
        name: "Inbound Calls",
        data: [
          65, 80, 80, 60, 60, 45, 45, 80, 80, 70, 70, 90, 90, 80, 80, 80, 60,
          60, 50,
        ],
      },
      {
        name: "Outbound Calls",
        data: [
          90, 110, 110, 95, 95, 85, 85, 95, 95, 115, 115, 100, 100, 115, 115,
          95, 95, 85, 85,
        ],
      },
    ];

    const themeMode = computed(() => {
      return store.mode;
    });

    onMounted(() => {
      Object.assign(chart, chartOptions(props.height));

      setTimeout(() => {
        refreshChart();
      }, 200);
    });

    const refreshChart = () => {
      if (!chartRef.value) {
        return;
      }

      Object.assign(chart, chartOptions(props.height));

      chartRef.value.refresh();
    };

    watch(themeMode, () => {
      refreshChart();
    });

    return {
      chart,
      chartRef,
      series,
      getAssetPath,
    };
  },
});

const chartOptions = (height: number): ApexOptions => {
  const labelColor = getCSSVariableValue("--kt-gray-500");
  const borderColor = getCSSVariableValue("--kt-border-dashed-color");
  const baseprimaryColor = getCSSVariableValue("--kt-primary");
  const lightprimaryColor = getCSSVariableValue("--kt-primary");
  const basesuccessColor = getCSSVariableValue("--kt-success");
  const lightsuccessColor = getCSSVariableValue("--kt-success");

  return {
    chart: {
      fontFamily: "inherit",
      type: "area",
      height: height,
      toolbar: {
        show: false,
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
      type: "gradient",
      gradient: {
        shadeIntensity: 1,
        opacityFrom: 0.4,
        opacityTo: 0.2,
        stops: [15, 120, 100],
      },
    },
    stroke: {
      curve: "smooth",
      show: true,
      width: 3,
      colors: [baseprimaryColor, basesuccessColor],
    },
    xaxis: {
      categories: [
        "",
        "8 AM",
        "81 AM",
        "9 AM",
        "10 AM",
        "11 AM",
        "12 PM",
        "13 PM",
        "14 PM",
        "15 PM",
        "16 PM",
        "17 PM",
        "18 PM",
        "18:20 PM",
        "18:20 PM",
        "19 PM",
        "20 PM",
        "21 PM",
        "",
      ],
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      tickAmount: 6,
      labels: {
        rotate: 0,
        rotateAlways: true,
        style: {
          colors: labelColor,
          fontSize: "12px",
        },
      },
      crosshairs: {
        position: "front",
        stroke: {
          width: 1,
          dashArray: 3,
        },
      },
      tooltip: {
        enabled: true,
        formatter: undefined,
        offsetY: 0,
        style: {
          fontSize: "12px",
        },
      },
    },
    yaxis: {
      max: 120,
      min: 30,
      tickAmount: 6,
      labels: {
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
    },
    colors: [lightprimaryColor, lightsuccessColor],
    grid: {
      borderColor: borderColor,
      strokeDashArray: 4,
      yaxis: {
        lines: {
          show: true,
        },
      },
    },
    markers: {
      strokeWidth: 3,
    },
  };
};
</script>
