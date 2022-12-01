<template>
  <div class="card card-flush" :class="className">
    <!--begin::Header-->
    <div class="card-header pt-5">
      <!--begin::Title-->
      <h3 class="card-title align-items-start flex-column">
        <span class="card-label fw-bold text-dark">Performance Overview</span>
        <span class="text-gray-400 mt-1 fw-semibold fs-6"
          >Users from all channels</span
        >
      </h3>
      <!--end::Title-->

      <!--begin::Toolbar-->
      <div class="card-toolbar">
        <ul class="nav" id="kt_chart_widget_8_tabs">
          <li class="nav-item">
            <a
              class="nav-link btn btn-sm btn-color-muted btn-active btn-active-light fw-bold px-4 me-1"
              data-bs-toggle="tab"
              id="kt_chart_widget_8_week_toggle"
              href="#kt_chart_widget_8_week_tab"
              >Month</a
            >
          </li>

          <li class="nav-item">
            <a
              class="nav-link btn btn-sm btn-color-muted btn-active btn-active-light fw-bold px-4 me-1 active"
              data-bs-toggle="tab"
              id="kt_chart_widget_8_month_toggle"
              href="#kt_chart_widget_8_month_tab"
              >Week</a
            >
          </li>
        </ul>
      </div>
      <!--end::Toolbar-->
    </div>
    <!--end::Header-->

    <!--begin::Body-->
    <div class="card-body pt-6">
      <!--begin::Tab content-->
      <div class="tab-content">
        <!--begin::Tab pane-->
        <div
          class="tab-pane fade"
          id="kt_chart_widget_8_week_tab"
          role="tabpanel"
        >
          <!--begin::Statistics-->
          <div class="mb-5">
            <!--begin::Statistics-->
            <div class="d-flex align-items-center mb-2">
              <span class="fs-1 fw-semibold text-gray-400 me-1 mt-n1">$</span>

              <span class="fs-3x fw-bold text-gray-800 me-2 lh-1 ls-n2"
                >18,89</span
              >

              <span class="badge badge-light-success fs-base">
                <span class="svg-icon svg-icon-5 svg-icon-success ms-n1">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/arrows/arr066.svg')
                    "
                  />
                </span>
                4,8%
              </span>
            </div>
            <!--end::Statistics-->

            <!--begin::Description-->
            <span class="fs-6 fw-semibold text-gray-400"
              >Avarage cost per interaction</span
            >
            <!--end::Description-->
          </div>
          <!--end::Statistics-->

          <!--begin::Chart-->
          <apexchart
            ref="chartRef1"
            type="bubble"
            :options="chart"
            :series="series1"
            :height="height"
          ></apexchart>
          <!--end::Chart-->

          <!--begin::Items-->
          <div class="d-flex flex-wrap pt-5">
            <!--begin::Item-->
            <div class="d-flex flex-column me-7 me-lg-16 pt-sm-3 pt-6">
              <!--begin::Item-->
              <div class="d-flex align-items-center mb-3 mb-sm-6">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-primary me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Social Campaigns</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->

              <!--begin::Item-->
              <div class="d-flex align-items-center">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-danger me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-<gray-600 fs-6">Google Ads</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->
            </div>
            <!--ed::Item-->

            <!--begin::Item-->
            <div class="d-flex flex-column me-7 me-lg-16 pt-sm-3 pt-6">
              <!--begin::Item-->
              <div class="d-flex align-items-center mb-3 mb-sm-6">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-success me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Email Newsletter</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->

              <!--begin::Item-->
              <div class="d-flex align-items-center">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-warning me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Courses</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->
            </div>
            <!--ed::Item-->

            <!--begin::Item-->
            <div class="d-flex flex-column pt-sm-3 pt-6">
              <!--begin::Item-->
              <div class="d-flex align-items-center mb-3 mb-sm-6">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-info me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">TV Campaign</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->

              <!--begin::Item-->
              <div class="d-flex align-items-center">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-success me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Radio</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->
            </div>
            <!--ed::Item-->
          </div>
          <!--ed::Items-->
        </div>
        <!--end::Tab pane-->

        <!--begin::Tab pane-->
        <div
          class="tab-pane fade active show"
          id="kt_chart_widget_8_month_tab"
          role="tabpanel"
        >
          <!--begin::Statistics-->
          <div class="mb-5">
            <!--begin::Statistics-->
            <div class="d-flex align-items-center mb-2">
              <span class="fs-1 fw-semibold text-gray-400 me-1 mt-n1">$</span>

              <span class="fs-3x fw-bold text-gray-800 me-2 lh-1 ls-n2"
                >8,55</span
              >

              <span class="badge badge-light-success fs-base">
                <span class="svg-icon svg-icon-5 svg-icon-success ms-n1">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/arrows/arr066.svg')
                    "
                  />
                </span>
                2.2%
              </span>
            </div>
            <!--end::Statistics-->

            <!--begin::Description-->
            <span class="fs-6 fw-semibold text-gray-400"
              >Avarage cost per interaction</span
            >
            <!--end::Description-->
          </div>
          <!--end::Statistics-->

          <!--begin::Chart-->
          <apexchart
            ref="chartRef2"
            type="bubble"
            :options="chart"
            :series="series2"
            :height="height"
          ></apexchart>
          <!--end::Chart-->

          <!--begin::Items-->
          <div class="d-flex flex-wrap pt-5">
            <!--begin::Item-->
            <div class="d-flex flex-column me-7 me-lg-16 pt-sm-3 pt-6">
              <!--begin::Item-->
              <div class="d-flex align-items-center mb-3 mb-sm-6">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-primary me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Social Campaigns</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->

              <!--begin::Item-->
              <div class="d-flex align-items-center">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-danger me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Google Ads</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->
            </div>
            <!--ed::Item-->

            <!--begin::Item-->
            <div class="d-flex flex-column me-7 me-lg-16 pt-sm-3 pt-6">
              <!--begin::Item-->
              <div class="d-flex align-items-center mb-3 mb-sm-6">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-success me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Email Newsletter</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->

              <!--begin::Item-->
              <div class="d-flex align-items-center">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-warning me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Courses</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->
            </div>
            <!--ed::Item-->

            <!--begin::Item-->
            <div class="d-flex flex-column pt-sm-3 pt-6">
              <!--begin::Item-->
              <div class="d-flex align-items-center mb-3 mb-sm-6">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-info me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">TV Campaign</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->

              <!--begin::Item-->
              <div class="d-flex align-items-center">
                <!--begin::Bullet-->
                <span
                  class="bullet bullet-dot bg-success me-2 h-10px w-10px"
                ></span>
                <!--end::Bullet-->

                <!--begin::Label-->
                <span class="fw-bold text-gray-600 fs-6">Radio</span>
                <!--end::Label-->
              </div>
              <!--ed::Item-->
            </div>
            <!--ed::Item-->
          </div>
          <!--ed::Items-->
        </div>
        <!--end::Tab pane-->
      </div>
      <!--end::Tab content-->
    </div>
    <!--end::Body-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { computed, defineComponent, onMounted, ref, watch } from "vue";
import type VueApexCharts from "vue3-apexcharts";
import type { ApexOptions } from "apexcharts";
import { useThemeStore } from "@/stores/theme";
import { getCSSVariableValue } from "@/assets/ts/_utils";

export default defineComponent({
  name: "default-dashboard-widget-6",
  components: {},
  props: {
    className: { type: String, required: false },
    height: { type: String, required: false, default: "425px" },
  },
  setup(props) {
    const chartRef1 = ref<typeof VueApexCharts | null>(null);
    const chartRef2 = ref<typeof VueApexCharts | null>(null);
    let chart: ApexOptions = {};
    const store = useThemeStore();

    const series1 = [
      {
        name: "Social Campaigns",
        data: [[100, 250, 30]],
      },
      {
        name: "Email Newsletter",
        data: [[225, 300, 35]],
      },
      {
        name: "TV Campaign",
        data: [[300, 350, 25]],
      },
      {
        name: "Google Ads",
        data: [[350, 350, 20]],
      },
      {
        name: "Courses",
        data: [[450, 400, 25]],
      },
      {
        name: "Radio",
        data: [[550, 350, 35]],
      },
    ];

    const series2 = [
      {
        name: "Social Campaigns",
        data: [[125, 300, 40]],
      },
      {
        name: "Email Newsletter",
        data: [[250, 350, 35]],
      },
      {
        name: "TV Campaign",
        data: [[350, 450, 30]],
      },
      {
        name: "Google Ads",
        data: [[450, 250, 25]],
      },
      {
        name: "Courses",
        data: [[500, 500, 30]],
      },
      {
        name: "Radio",
        data: [[600, 250, 28]],
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
      if (!chartRef1.value || !chartRef2.value) {
        return;
      }

      Object.assign(chart, chartOptions(props.height));

      chartRef1.value.refresh();
      chartRef2.value.refresh();
    };

    watch(themeMode, () => {
      refreshChart();
    });

    return {
      chart,
      chartRef1,
      chartRef2,
      series1,
      series2,
      getAssetPath,
    };
  },
});

const chartOptions = (height: string): ApexOptions => {
  const borderColor = getCSSVariableValue("--kt-border-dashed-color");

  return {
    chart: {
      fontFamily: "inherit",
      type: "bubble",
      height: height,
      toolbar: {
        show: false,
      },
    },
    plotOptions: {
      bubble: {},
    },
    stroke: {
      show: false,
      width: 0,
    },
    legend: {
      show: false,
    },
    dataLabels: {
      enabled: false,
    },
    xaxis: {
      type: "numeric",
      tickAmount: 7,
      min: 0,
      max: 700,
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: true,
        height: 0,
      },
      labels: {
        show: true,
        trim: true,
        style: {
          colors: getCSSVariableValue("--kt-gray-500"),
          fontSize: "13px",
        },
      },
    },
    yaxis: {
      tickAmount: 7,
      min: 0,
      max: 700,
      labels: {
        style: {
          colors: getCSSVariableValue("--kt-gray-500"),
          fontSize: "13px",
        },
      },
    },
    tooltip: {
      style: {
        fontSize: "12px",
      },
      x: {
        formatter: function (val) {
          return "Clicks: " + val;
        },
      },
      y: {
        formatter: function (val) {
          return "$" + val + "K";
        },
      },
      z: {
        title: "Impression: ",
      },
    },
    colors: [
      getCSSVariableValue("--kt-primary"),
      getCSSVariableValue("--kt-success"),
      getCSSVariableValue("--kt-warning"),
      getCSSVariableValue("--kt-danger"),
      getCSSVariableValue("--kt-info"),
      "#43CED7",
    ],
    fill: {
      opacity: 1,
    },
    markers: {
      strokeWidth: 0,
    },
    grid: {
      borderColor: borderColor,
      strokeDashArray: 4,
      padding: {
        right: 20,
      },
      yaxis: {
        lines: {
          show: true,
        },
      },
    },
  };
};
</script>
