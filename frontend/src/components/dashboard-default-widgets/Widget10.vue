<template>
  <div class="card card-flush" :class="className">
    <!--begin::Header-->
    <div class="card-header pt-7">
      <!--begin::Title-->
      <h3 class="card-title align-items-start flex-column">
        <span class="card-label fw-bold text-gray-800">Projects Stats</span>

        <span class="text-gray-400 mt-1 fw-semibold fs-6"
          >Updated 37 minutes ago</span
        >
      </h3>
      <!--end::Title-->

      <!--begin::Toolbar-->
      <div class="card-toolbar">
        <a
          href="<?php echo Theme::getPageUrl('apps/ecommerce/catalog/add-product')?>"
          class="btn btn-sm btn-light"
          >History</a
        >
      </div>
      <!--end::Toolbar-->
    </div>
    <!--end::Header-->

    <!--begin::Body-->
    <div class="card-body pt-6">
      <!--begin::Table container-->
      <div class="table-responsive">
        <!--begin::Table-->
        <table class="table table-row-dashed align-middle gs-0 gy-3 my-0">
          <!--begin::Table head-->
          <thead>
            <tr class="fs-7 fw-bold text-gray-400 border-bottom-0">
              <th class="p-0 pb-3 min-w-175px text-start">ITEM</th>
              <th class="p-0 pb-3 min-w-100px text-end">BUDGET</th>
              <th class="p-0 pb-3 min-w-100px text-end">PROGRESS</th>
              <th class="p-0 pb-3 min-w-175px text-end pe-12">STATUS</th>
              <th class="p-0 pb-3 w-50px text-end">VIEW</th>
            </tr>
          </thead>
          <!--end::Table head-->

          <!--begin::Table body-->
          <tbody>
            <template v-for="(row, i) in table" :key="i">
              <tr>
                <td>
                  <div class="d-flex align-items-center">
                    <div class="symbol symbol-50px me-3">
                      <img :src="row.img" class="" alt="" />
                    </div>

                    <div class="d-flex justify-content-start flex-column">
                      <a
                        href="#"
                        class="text-gray-800 fw-bold text-hover-primary mb-1 fs-6"
                        >{{ row.title }}</a
                      >
                      <span class="text-gray-400 fw-semibold d-block fs-7">{{
                        row.name
                      }}</span>
                    </div>
                  </div>
                </td>

                <td class="text-end pe-0">
                  <span class="text-gray-600 fw-bold fs-6"
                    >$ {{ row.price }}</span
                  >
                </td>

                <td class="text-end pe-0">
                  <!--begin::Label-->
                  <span
                    v-if="row.icon"
                    class="badge badge-light-success fs-base"
                  >
                    <span class="svg-icon svg-icon-5 svg-icon-success ms-n1">
                      <inline-svg
                        :src="
                          getAssetPath('/media/icons/duotune/arrows/arr066.svg')
                        "
                      />
                    </span>
                    {{ row.statistics }} %
                  </span>
                  <!--end::Label-->
                  <!--begin::Label-->
                  <span v-else class="badge badge-light-danger fs-base">
                    <span class="svg-icon svg-icon-5 svg-icon-danger ms-n1">
                      <inline-svg
                        :src="
                          getAssetPath('/media/icons/duotune/arrows/arr065.svg')
                        "
                      />
                    </span>
                    {{ row.statistics }} %
                  </span>
                  <!--end::Label-->
                </td>

                <td class="text-end pe-12">
                  <span
                    :class="`badge py-3 px-4 fs-7 badge-light-${row.status.state}`"
                    >{{ row.status.label }}</span
                  >
                </td>

                <td class="text-end">
                  <a
                    href="#"
                    class="btn btn-sm btn-icon btn-bg-light btn-active-color-primary w-30px h-30px"
                  >
                    <span class="svg-icon svg-icon-5 svg-icon-gray-700">
                      <inline-svg
                        :src="
                          getAssetPath('/media/icons/duotune/arrows/arr001.svg')
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
      </div>
      <!--end::Table-->
    </div>
    <!--end: Card Body-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "default-dashboard-widget-10",
  components: {},
  props: {
    className: { type: String, required: false },
  },
  setup() {
    const table = [
      {
        img: getAssetPath("/media/stock/600x600/img-49.jpg"),
        title: "Mivy App",
        name: "Jane Cooper",
        price: "32,400",
        statistics: "9.2",
        icon: true,
        chartColor: "success",
        status: {
          label: "In Process",
          state: "primary",
        },
      },
      {
        img: getAssetPath("/media/stock/600x600/img-40.jpg"),
        title: "Avionica",
        name: "Esther Howard",
        price: "256,910",
        statistics: "0.4",
        icon: false,
        chartColor: "danger",
        status: {
          label: "On Hold",
          state: "warning",
        },
      },
      {
        img: getAssetPath("/media/stock/600x600/img-39.jpg"),
        title: "Charto CRM",
        name: "Jenny Wilson",
        price: "8,220",
        statistics: "9.2",
        icon: true,
        chartColor: "success",
        status: {
          label: "In Process",
          state: "primary",
        },
      },
      {
        img: getAssetPath("/media/stock/600x600/img-47.jpg"),
        title: "Tower Hill",
        name: "Cody Fisher",
        price: "74,000",
        statistics: "9.2",
        icon: true,
        chartColor: "success",
        status: {
          label: "Completed",
          state: "success",
        },
      },
      {
        img: getAssetPath("/media/stock/600x600/img-48.jpg"),
        title: "9 Degree",
        name: "Savannah Nguyen",
        price: "183,300",
        statistics: "0.4",
        icon: false,
        chartColor: "danger",
        status: {
          label: "In Process",
          state: "primary",
        },
      },
    ];

    return {
      table,
      getAssetPath,
    };
  },
});
</script>
