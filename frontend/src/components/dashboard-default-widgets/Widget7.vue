<template>
  <!--begin::Tables widget 16-->
  <div class="card card-flush" :class="className">
    <!--begin::Header-->
    <div class="card-header pt-5">
      <!--begin::Title-->
      <h3 class="card-title align-items-start flex-column">
        <span class="card-label fw-bold text-gray-800"
          >Authors Achievements</span
        >

        <span class="text-gray-400 mt-1 fw-semibold fs-6"
          >Avg. 69.34% Conv. Rate</span
        >
      </h3>
      <!--end::Title-->

      <!--begin::Toolbar-->
      <div class="card-toolbar">
        <!--begin::Menu-->
        <button
          class="btn btn-icon btn-color-gray-400 btn-active-color-primary justify-content-end"
          data-kt-menu-trigger="click"
          data-kt-menu-placement="bottom-end"
          data-kt-menu-overflow="true"
        >
          <span class="svg-icon svg-icon-1 svg-icon-gray-300 me-n1">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen023.svg')"
            />
          </span>
        </button>

        <Dropdown2 />
        <!--end::Menu-->
      </div>
      <!--end::Toolbar-->
    </div>
    <!--end::Header-->

    <!--begin::Body-->
    <div class="card-body pt-6">
      <!--begin::Nav-->
      <ul class="nav nav-pills nav-pills-custom mb-3">
        <template v-for="(item, i) in items" :key="i">
          <!--begin::Item-->
          <li class="nav-item mb-3 me-3 me-lg-6">
            <!--begin::Link-->
            <a
              class="nav-link btn btn-outline btn-flex btn-color-muted btn-active-color-primary flex-column overflow-hidden w-80px h-85px pt-5 pb-2"
              :class="[i === 0 && 'active']"
              :id="`kt_stats_widget_16_tab_link_${i}`"
              data-bs-toggle="pill"
              :href="`#kt_stats_widget_16_tab_${i}`"
            >
              <!--begin::Icon-->
              <div class="nav-icon mb-3">
                <i :class="`${item.icon} fs-1 p-0`"></i>
              </div>
              <!--end::Icon-->

              <!--begin::Title-->
              <span class="nav-text text-gray-800 fw-bold fs-6 lh-1">
                {{ item.title }}
              </span>
              <!--end::Title-->

              <!--begin::Bullet-->
              <span
                class="bullet-custom position-absolute bottom-0 w-100 h-4px bg-primary"
              ></span>
              <!--end::Bullet-->
            </a>
            <!--end::Link-->
          </li>
          <!--end::Item-->
        </template>
      </ul>
      <!--end::Nav-->

      <!--begin::Tab Content-->
      <div class="tab-content">
        <template v-for="(item, i) in items" :key="i">
          <!--begin::Tap pane-->
          <div
            class="tab-pane fade"
            :class="[i === 0 && 'active show']"
            :id="`kt_stats_widget_16_tab_${i}`"
          >
            <!--begin::Table container-->
            <div class="table-responsive">
              <!--begin::Table-->
              <table class="table table-row-dashed align-middle gs-0 gy-3 my-0">
                <!--begin::Table head-->
                <thead>
                  <tr class="fs-7 fw-bold text-gray-400 border-bottom-0">
                    <th class="p-0 pb-3 min-w-150px text-start">AUTHOR</th>
                    <th class="p-0 pb-3 min-w-100px text-end pe-13">CONV.</th>
                    <th class="p-0 pb-3 w-125px text-end pe-7">PROGRESS</th>
                    <th class="p-0 pb-3 w-50px text-end">VIEW</th>
                  </tr>
                </thead>
                <!--end::Table head-->

                <!--begin::Table body-->
                <tbody>
                  <template v-for="(row, j) in item.table" :key="j">
                    <tr>
                      <td>
                        <div class="d-flex align-items-center">
                          <div class="symbol symbol-50px me-3">
                            <img :src="row.agent.avatar" class="" alt="" />
                          </div>

                          <div class="d-flex justify-content-start flex-column">
                            <router-link
                              to="/crafted/pages/profile/overview"
                              class="text-gray-800 fw-bold text-hover-primary mb-1 fs-6"
                              >{{ row.agent.name }}</router-link
                            >
                            <span
                              class="text-gray-400 fw-semibold d-block fs-7"
                              >{{ row.agent.city }}</span
                            >
                          </div>
                        </div>
                      </td>

                      <td class="text-end pe-13">
                        <span class="text-gray-600 fw-bold fs-6"
                          >{{ row.price }}%</span
                        >
                      </td>

                      <td class="text-end pe-0">
                        <!--begin::Label-->
                        <span
                          v-if="row.icon"
                          class="badge badge-light-success fs-base"
                        >
                          <span
                            class="svg-icon svg-icon-5 svg-icon-success ms-n1"
                          >
                            <inline-svg
                              :src="
                                getAssetPath(
                                  '/media/icons/duotune/arrows/arr066.svg'
                                )
                              "
                            />
                          </span>
                          {{ row.statistics }} %
                        </span>
                        <!--end::Label-->
                        <!--begin::Label-->
                        <span v-else class="badge badge-light-danger fs-base">
                          <span
                            class="svg-icon svg-icon-5 svg-icon-danger ms-n1"
                          >
                            <inline-svg
                              :src="
                                getAssetPath(
                                  '/media/icons/duotune/arrows/arr065.svg'
                                )
                              "
                            />
                          </span>
                          {{ row.statistics }} %
                        </span>
                        <!--end::Label-->
                      </td>

                      <td class="text-end">
                        <a
                          href="#"
                          class="btn btn-sm btn-icon btn-bg-light btn-active-color-primary w-30px h-30px"
                        >
                          <span class="svg-icon svg-icon-5 svg-icon-gray-700">
                            <inline-svg
                              :src="
                                getAssetPath(
                                  '/media/icons/duotune/arrows/arr001.svg'
                                )
                              "
                            />
                          </span>
                        </a>
                      </td>
                    </tr>
                  </template>
                </tbody>
                <!--end::Table body-->
              </table>
              <!--end::Table-->
            </div>
            <!--end::Table container-->
          </div>
          <!--end::Tap pane-->
        </template>
      </div>
      <!--end::Tab Content-->
    </div>
    <!--end: Card Body-->
  </div>
  <!--end::Tables widget 16-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import Dropdown2 from "@/components/dropdown/Dropdown2.vue";

export default defineComponent({
  name: "default-dashboard-widget-7",
  components: {
    Dropdown2,
  },
  props: {
    className: { type: String, required: false },
  },
  setup() {
    const items = [
      {
        title: "SaaS",
        icon: "fonticon-drive",
        index: "1",
        table: [
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-3.jpg"),
              name: "Guy Hawkins",
              city: "Haiti",
            },
            price: "78.34",
            icon: true,
            statistics: 5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-2.jpg"),
              name: "Jane Cooper",
              city: "Monaco",
            },
            price: "63.83",
            icon: false,
            statistics: 10,
            chartCcolor: "danger",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-9.jpg"),
              name: "Jacob Jones",
              city: "Poland",
            },
            price: "92.56",
            icon: true,
            statistics: 9,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-7.jpg"),
              name: "Cody Fishers",
              city: "Mexico",
            },
            price: "63.08",
            icon: false,
            statistics: 9.5,
            chartCcolor: "success",
          },
        ],
      },
      {
        title: "Crypto",
        icon: "fonticon-bank",
        index: "2",
        table: [
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-25.jpg"),
              name: "Brooklyn Simmons",
              city: "Poland",
            },
            price: "85.23",
            icon: true,
            statistics: 9.5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-24.jpg"),
              name: "Esther Howard",
              city: "Mexico",
            },
            price: "74.83",
            icon: false,
            statistics: 10,
            chartCcolor: "danger",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-20.jpg"),
              name: "Annette Black",
              city: "Haiti",
            },
            price: "90.06",
            icon: true,
            statistics: 3,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-17.jpg"),
              name: "Marvin McKinney",
              city: "Monaco",
            },
            price: "54.08",
            icon: false,
            statistics: 9.5,
            chartCcolor: "success",
          },
        ],
      },
      {
        title: "Social",
        icon: "fonticon-like-1",
        index: "3",
        table: [
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-11.jpg"),
              name: "Jacob Jones",
              city: "New York",
            },
            price: "52.34",
            icon: true,
            statistics: 10.5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-23.jpg"),
              name: "Ronald Richards",
              city: "Madrid",
            },
            price: "77.65",
            icon: true,
            statistics: 9.5,
            chartCcolor: "danger",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-4.jpg"),
              name: "Leslie Alexander",
              city: "Pune",
            },
            price: "82.47",
            icon: false,
            statistics: 10.5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-1.jpg"),
              name: "Courtney Henry",
              city: "Mexico",
            },
            price: "67.84",
            icon: true,
            statistics: 11.5,
            chartCcolor: "success",
          },
        ],
      },
      {
        title: "Mobile",
        icon: "fonticon-remote-control",
        index: "4",
        table: [
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-12.jpg"),
              name: "Arlene McCoy",
              city: "London",
            },
            price: "53.44",
            icon: false,
            statistics: 9.5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-21.jpg"),
              name: "Marvin McKinneyr",
              city: "Monaco",
            },
            price: "74.64",
            icon: false,
            statistics: 9.5,
            chartCcolor: "danger",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-30.jpg"),
              name: "Jacob Jones",
              city: "PManila",
            },
            price: "88.56",
            icon: true,
            statistics: 5.5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-14.jpg"),
              name: "Esther Howard",
              city: "Iceland",
            },
            price: "63.16",
            icon: false,
            statistics: 1.5,
            chartCcolor: "success",
          },
        ],
      },
      {
        title: "Others",
        icon: "fonticon-telegram",
        index: "5",
        table: [
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-6.jpg"),
              name: "Jane Cooper",
              city: "Haiti",
            },
            price: "68.54",
            icon: true,
            statistics: 0.5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-10.jpg"),
              name: "Esther Howard",
              city: "Kiribati",
            },
            price: "55.83",
            icon: false,
            statistics: 1.5,
            chartCcolor: "danger",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-9.jpg"),
              name: "Jacob Jones",
              city: "Poland",
            },
            price: "93.46",
            icon: true,
            statistics: 9.5,
            chartCcolor: "success",
          },
          {
            agent: {
              avatar: getAssetPath("/media/avatars/300-3.jpg"),
              name: "Ralph Edwards",
              city: "Mexico",
            },
            price: "64.48",
            icon: false,
            statistics: 9.5,
            chartCcolor: "success",
          },
        ],
      },
    ];

    return {
      items,
      getAssetPath,
    };
  },
});
</script>
