<template>
  <!--begin::Pricing-->
  <div class="card card-flush pt-3 mb-5 mb-lg-10">
    <!--begin::Card header-->
    <div class="card-header">
      <!--begin::Card title-->
      <div class="card-title">
        <h2 class="fw-bold">Products</h2>
      </div>
      <!--begin::Card title-->

      <!--begin::Card toolbar-->
      <div class="card-toolbar">
        <button
          type="button"
          class="btn btn-light-primary"
          data-bs-toggle="modal"
          data-bs-target="#kt_modal_add_product"
        >
          Add Product
        </button>
      </div>
      <!--end::Card toolbar-->
    </div>
    <!--end::Card header-->

    <!--begin::Card body-->
    <div class="card-body pt-0">
      <!--begin::Table wrapper-->
      <div class="table-responsive">
        <!--begin::Table-->
        <table
          class="table align-middle table-row-dashed fs-6 fw-semobold gy-4"
          id="kt_subscription_products_table"
        >
          <!--begin::Table head-->
          <thead>
            <tr class="text-start text-muted fw-bold fs-7 text-uppercase gs-0">
              <th class="min-w-300px">Product</th>
              <th class="min-w-100px">Qty</th>
              <th class="min-w-150px">Total</th>
              <th class="min-w-70px text-end">Remove</th>
            </tr>
          </thead>
          <!--end::Table head-->

          <!--begin::Table body-->
          <tbody class="text-gray-600">
            <tr class="odd">
              <td>Beginner Plan</td>
              <td>1</td>
              <td>149.99 / Month</td>
              <td class="text-end">
                <!--begin::Delete-->
                <a
                  href="#"
                  class="btn btn-icon btn-flex btn-active-light-primary w-30px h-30px me-3"
                  data-bs-toggle="tooltip"
                  title=""
                  data-kt-action="product_remove"
                  data-bs-original-title="Delete"
                >
                  <span class="svg-icon svg-icon-3">
                    <inline-svg
                      :src="
                        getAssetPath('/media/icons/duotune/general/gen027.svg')
                      "
                    />
                  </span>
                </a>
                <!--end::Delete-->
              </td>
            </tr>
            <tr class="even">
              <td>Pro Plan</td>
              <td>1</td>
              <td>499.99 / Month</td>
              <td class="text-end">
                <!--begin::Delete-->
                <a
                  href="#"
                  class="btn btn-icon btn-flex btn-active-light-primary w-30px h-30px me-3"
                  data-bs-toggle="tooltip"
                  title=""
                  data-kt-action="product_remove"
                  data-bs-original-title="Delete"
                >
                  <span class="svg-icon svg-icon-3">
                    <inline-svg
                      :src="
                        getAssetPath('/media/icons/duotune/general/gen027.svg')
                      "
                    />
                  </span>
                </a>
                <!--end::Delete-->
              </td>
            </tr>
            <tr class="odd">
              <td>Team Plan</td>
              <td>1</td>
              <td>999.99 / Month</td>
              <td class="text-end">
                <!--begin::Delete-->
                <a
                  href="#"
                  class="btn btn-icon btn-flex btn-active-light-primary w-30px h-30px me-3"
                  data-bs-toggle="tooltip"
                  title=""
                  data-kt-action="product_remove"
                  data-bs-original-title="Delete"
                >
                  <span class="svg-icon svg-icon-3">
                    <inline-svg
                      :src="
                        getAssetPath('/media/icons/duotune/general/gen027.svg')
                      "
                    />
                  </span>
                </a>
                <!--end::Delete-->
              </td>
            </tr>
          </tbody>
          <!--end::Table body-->
        </table>
        <!--end::Table-->
      </div>
      <!--end::Table wrapper-->
    </div>
    <!--end::Card body-->
  </div>
  <!--end::Pricing-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-products",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
