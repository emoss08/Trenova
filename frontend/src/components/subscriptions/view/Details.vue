<template>
  <!--begin::Card-->
  <div class="card card-flush pt-3 mb-5 mb-xl-10">
    <!--begin::Card header-->
    <div class="card-header">
      <!--begin::Card title-->
      <div class="card-title">
        <h2 class="fw-bold">Product Details</h2>
      </div>
      <!--begin::Card title-->

      <!--begin::Card toolbar-->
      <div class="card-toolbar">
        <router-link to="/subscriptions/add" class="btn btn-light-primary"
          >Update Product</router-link
        >
      </div>
      <!--end::Card toolbar-->
    </div>
    <!--end::Card header-->

    <!--begin::Card body-->
    <div class="card-body pt-3">
      <!--begin::Section-->
      <div class="mb-10">
        <!--begin::Title-->
        <h5 class="mb-4">Billing Address:</h5>
        <!--end::Title-->

        <!--begin::Details-->
        <div class="d-flex flex-wrap py-5">
          <!--begin::Row-->
          <div class="flex-equal me-5">
            <!--begin::Details-->
            <table class="table fs-6 fw-semobold gs-0 gy-2 gx-2 m-0">
              <!--begin::Row-->
              <tr>
                <td class="text-gray-400 min-w-175px w-175px">Bill to:</td>
                <td class="text-gray-800 min-w-200px">
                  <router-link
                    to="/subscriptions/view"
                    class="text-gray-800 text-hover-primary"
                  >
                    e.smith@kpmg.com.au</router-link
                  >
                </td>
              </tr>
              <!--end::Row-->

              <!--begin::Row-->
              <tr>
                <td class="text-gray-400">Customer Name:</td>
                <td class="text-gray-800">Emma Smith</td>
              </tr>
              <!--end::Row-->

              <!--begin::Row-->
              <tr>
                <td class="text-gray-400">Address:</td>
                <td class="text-gray-800">
                  Floor 10, 101 Avenue of the Light Square, New York, NY, 10050.
                </td>
              </tr>
              <!--end::Row-->

              <!--begin::Row-->
              <tr>
                <td class="text-gray-400">Phone:</td>
                <td class="text-gray-800">(555) 555-1234</td>
              </tr>
              <!--end::Row-->
            </table>
            <!--end::Details-->
          </div>
          <!--end::Row-->

          <!--begin::Row-->
          <div class="flex-equal">
            <!--begin::Details-->
            <table class="table fs-6 fw-semobold gs-0 gy-2 gx-2 m-0">
              <!--begin::Row-->
              <tr>
                <td class="text-gray-400 min-w-175px w-175px">
                  Subscribed Product:
                </td>
                <td class="text-gray-800 min-w-200px">
                  <a href="#" class="text-gray-800 text-hover-primary"
                    >Basic Bundle</a
                  >
                </td>
              </tr>
              <!--end::Row-->

              <!--begin::Row-->
              <tr>
                <td class="text-gray-400">Subscription Fees:</td>
                <td class="text-gray-800">$149.99 / Year</td>
              </tr>
              <!--end::Row-->

              <!--begin::Row-->
              <tr>
                <td class="text-gray-400">Billing method:</td>
                <td class="text-gray-800">Annually</td>
              </tr>
              <!--end::Row-->

              <!--begin::Row-->
              <tr>
                <td class="text-gray-400">Currency:</td>
                <td class="text-gray-800">USD - US Dollar</td>
              </tr>
              <!--end::Row-->
            </table>
            <!--end::Details-->
          </div>
          <!--end::Row-->
        </div>
        <!--end::Row-->
      </div>
      <!--end::Section-->

      <!--begin::Section-->
      <div class="mb-0">
        <!--begin::Title-->
        <h5 class="mb-4">Subscribed Products:</h5>
        <!--end::Title-->

        <!--begin::Product table-->
        <div class="table-responsive">
          <!--begin::Table-->
          <table class="table align-middle table-row-dashed fs-6 gy-4 mb-0">
            <!--begin::Table head-->
            <thead>
              <!--begin::Table row-->
              <tr
                class="border-bottom border-gray-200 text-start text-gray-400 fw-bold fs-7 text-uppercase gs-0"
              >
                <th class="min-w-150px">Product</th>
                <th class="min-w-125px">Subscription ID</th>
                <th class="min-w-125px">Qty</th>
                <th class="min-w-125px">Total</th>
                <th class="text-end min-w-70px">Actions</th>
              </tr>
              <!--end::Table row-->
            </thead>
            <!--end::Table head-->

            <!--begin::Table body-->
            <tbody class="fw-semobold text-gray-800">
              <tr>
                <td>
                  <label class="w-150px">Basic Bundle</label>
                  <div class="fw-normal text-gray-600">Basic yearly bundle</div>
                </td>
                <td>
                  <span class="badge badge-light-danger">sub_4567_8765</span>
                </td>
                <td>1</td>
                <td>$149.99 / Year</td>
                <td class="text-end">
                  <!--begin::Action-->
                  <a
                    href="#"
                    class="btn btn-icon btn-active-light-primary w-30px h-30px"
                    data-kt-menu-trigger="click"
                    data-kt-menu-placement="bottom-end"
                  >
                    <span class="svg-icon svg-icon-3">
                      <inline-svg
                        :src="
                          getAssetPath(
                            '/media/icons/duotune/general/gen019.svg'
                          )
                        "
                      />
                    </span>
                  </a>
                  <UserMenu></UserMenu>
                  <!--end::Action-->
                </td>
              </tr>
              <tr>
                <td>
                  <label class="w-150px">Pro Bundle</label>
                  <div class="fw-normal text-gray-400">Basic yearly bundle</div>
                </td>
                <td>
                  <span class="badge badge-light-danger">sub_4567_3433</span>
                </td>
                <td>5</td>
                <td>$949.99 / Year</td>
                <td class="text-end">
                  <!--begin::Action-->
                  <a
                    href="#"
                    class="btn btn-icon btn-active-light-primary w-30px h-30px"
                    data-kt-menu-trigger="click"
                    data-kt-menu-placement="bottom-end"
                  >
                    <span class="svg-icon svg-icon-3">
                      <inline-svg
                        :src="
                          getAssetPath(
                            '/media/icons/duotune/general/gen019.svg'
                          )
                        "
                      />
                    </span>
                  </a>
                  <UserMenu></UserMenu>
                  <!--end::Action-->
                </td>
              </tr>
            </tbody>
            <!--end::Table body-->
          </table>
          <!--end::Table-->
        </div>
        <!--end::Product table-->
      </div>
      <!--end::Section-->
    </div>
    <!--end::Card body-->
  </div>
  <!--end::Card-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import UserMenu from "@/layouts/main-layout/menus/UserAccountMenu.vue";

export default defineComponent({
  name: "kt-details",
  components: {
    UserMenu,
  },
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
