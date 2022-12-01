<template>
  <div
    class="card card-flush pt-3 mb-5 mb-lg-10"
    data-kt-subscriptions-form="pricing"
  >
    <div class="card-header">
      <div class="card-title">
        <h2 class="fw-bold">Payment Method</h2>
      </div>

      <div class="card-toolbar">
        <a
          href="#"
          class="btn btn-light-primary"
          data-bs-toggle="modal"
          data-bs-target="#kt_modal_new_card"
          >New Method</a
        >
      </div>
    </div>

    <div class="card-body pt-0">
      <div id="kt_create_new_payment_method">
        <div class="py-1">
          <div class="py-3 d-flex flex-stack flex-wrap">
            <div
              class="d-flex align-items-center collapsible toggle collapsed"
              data-bs-toggle="collapse"
              data-bs-target="#kt_create_new_payment_method_1"
              aria-expanded="false"
            >
              <div
                class="btn btn-sm btn-icon btn-active-color-primary ms-n3 me-2"
              >
                <span class="svg-icon toggle-on svg-icon-primary svg-icon-2">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/general/gen036.svg')
                    "
                  />
                </span>
                <span class="svg-icon toggle-off svg-icon-2">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/general/gen035.svg')
                    "
                  />
                </span>
              </div>

              <img
                :src="getAssetPath('/media/svg/card-logos/mastercard.svg')"
                class="w-40px me-3"
                alt=""
              />

              <div class="me-3">
                <div class="d-flex align-items-center fw-bold">
                  Mastercard
                  <div class="badge badge-light-primary ms-5">Primary</div>
                </div>
                <div class="text-muted">Expires Dec 2024</div>
              </div>
            </div>

            <div class="d-flex my-3 ms-9">
              <label class="form-check form-check-custom form-check-solid me-5">
                <input
                  class="form-check-input"
                  type="radio"
                  name="payment_method"
                  checked
                />
              </label>
            </div>
          </div>

          <div
            id="kt_create_new_payment_method_1"
            class="fs-6 ps-10 collapse"
            style=""
          >
            <div class="d-flex flex-wrap py-5">
              <div class="flex-equal me-5">
                <table class="table table-flush fw-semobold gy-1">
                  <tbody>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Name</td>
                      <td class="text-gray-800">Emma Smith</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Number</td>
                      <td class="text-gray-800">**** 6129</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Expires</td>
                      <td class="text-gray-800">12/2024</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Type</td>
                      <td class="text-gray-800">Mastercard credit card</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Issuer</td>
                      <td class="text-gray-800">VICBANK</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">ID</td>
                      <td class="text-gray-800">id_4325df90sdf8</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <div class="flex-equal">
                <table class="table table-flush fw-semobold gy-1">
                  <tbody>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">
                        Billing address
                      </td>
                      <td class="text-gray-800">AU</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Phone</td>
                      <td class="text-gray-800">No phone provided</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Email</td>
                      <td class="text-gray-800">
                        <a href="#" class="text-gray-900 text-hover-primary"
                          >e.smith@kpmg.com.au</a
                        >
                      </td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Origin</td>
                      <td class="text-gray-800">
                        Australia
                        <div class="symbol symbol-20px symbol-circle ms-2">
                          <img
                            :src="getAssetPath('/media/flags/australia.svg')"
                          />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">CVC check</td>
                      <td class="text-gray-800">
                        Passed
                        <span class="svg-icon svg-icon-2 svg-icon-success">
                          <inline-svg
                            :src="
                              getAssetPath(
                                '/media/icons/duotune/general/gen043.svg'
                              )
                            "
                          />
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <div class="separator separator-dashed"></div>

        <div class="py-1">
          <div class="py-3 d-flex flex-stack flex-wrap">
            <div
              class="d-flex align-items-center collapsible toggle collapsed"
              data-bs-toggle="collapse"
              data-bs-target="#kt_create_new_payment_method_2"
            >
              <div
                class="btn btn-sm btn-icon btn-active-color-primary ms-n3 me-2"
              >
                <span class="svg-icon toggle-on svg-icon-primary svg-icon-2">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/general/gen036.svg')
                    "
                  />
                </span>

                <span class="svg-icon toggle-off svg-icon-2">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/general/gen035.svg')
                    "
                  />
                </span>
              </div>

              <img
                :src="getAssetPath('/media/svg/card-logos/visa.svg')"
                class="w-40px me-3"
                alt=""
              />

              <div class="me-3">
                <div class="d-flex align-items-center fw-bold">Visa</div>
                <div class="text-muted">Expires Feb 2022</div>
              </div>
            </div>

            <div class="d-flex my-3 ms-9">
              <label class="form-check form-check-custom form-check-solid me-5">
                <input
                  class="form-check-input"
                  type="radio"
                  name="payment_method"
                />
              </label>
            </div>
          </div>

          <div id="kt_create_new_payment_method_2" class="collapse fs-6 ps-10">
            <div class="d-flex flex-wrap py-5">
              <div class="flex-equal me-5">
                <table class="table table-flush fw-semobold gy-1">
                  <tbody>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Name</td>
                      <td class="text-gray-800">Melody Macy</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Number</td>
                      <td class="text-gray-800">**** 1141</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Expires</td>
                      <td class="text-gray-800">02/2022</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Type</td>
                      <td class="text-gray-800">Visa credit card</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Issuer</td>
                      <td class="text-gray-800">ENBANK</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">ID</td>
                      <td class="text-gray-800">id_w2r84jdy723</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <div class="flex-equal">
                <table class="table table-flush fw-semobold gy-1">
                  <tbody>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">
                        Billing address
                      </td>
                      <td class="text-gray-800">UK</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Phone</td>
                      <td class="text-gray-800">No phone provided</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Email</td>
                      <td class="text-gray-800">
                        <a href="#" class="text-gray-900 text-hover-primary"
                          >melody@altbox.com</a
                        >
                      </td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Origin</td>
                      <td class="text-gray-800">
                        United Kingdom
                        <div class="symbol symbol-20px symbol-circle ms-2">
                          <img
                            :src="
                              getAssetPath('/media/flags/united-kingdom.svg')
                            "
                          />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">CVC check</td>
                      <td class="text-gray-800">
                        Passed

                        <span class="svg-icon svg-icon-2 svg-icon-success">
                          <inline-svg
                            :src="
                              getAssetPath(
                                '/media/icons/duotune/arrows/arr012.svg'
                              )
                            "
                          />
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <div class="separator separator-dashed"></div>

        <div class="py-1">
          <div class="py-3 d-flex flex-stack flex-wrap">
            <div
              class="d-flex align-items-center collapsible toggle collapsed"
              data-bs-toggle="collapse"
              data-bs-target="#kt_create_new_payment_method_3"
            >
              <div
                class="btn btn-sm btn-icon btn-active-color-primary ms-n3 me-2"
              >
                <span class="svg-icon toggle-on svg-icon-primary svg-icon-2">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/general/gen036.svg')
                    "
                  />
                </span>

                <span class="svg-icon toggle-off svg-icon-2">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/general/gen035.svg')
                    "
                  />
                </span>
              </div>

              <img
                :src="
                  getAssetPath('/media/svg/card-logos/american-express.svg')
                "
                class="w-40px me-3"
                alt=""
              />

              <div class="me-3">
                <div class="d-flex align-items-center fw-bold">
                  American Express
                  <div class="badge badge-light-danger ms-5">Expired</div>
                </div>
                <div class="text-muted">Expires Aug 2021</div>
              </div>
            </div>

            <div class="d-flex my-3 ms-9">
              <label class="form-check form-check-custom form-check-solid me-5">
                <input
                  class="form-check-input"
                  type="radio"
                  name="payment_method"
                />
              </label>
            </div>
          </div>

          <div id="kt_create_new_payment_method_3" class="collapse fs-6 ps-10">
            <div class="d-flex flex-wrap py-5">
              <div class="flex-equal me-5">
                <table class="table table-flush fw-semobold gy-1">
                  <tbody>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Name</td>
                      <td class="text-gray-800">Max Smith</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Number</td>
                      <td class="text-gray-800">**** 4625</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Expires</td>
                      <td class="text-gray-800">08/2021</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Type</td>
                      <td class="text-gray-800">
                        American express credit card
                      </td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Issuer</td>
                      <td class="text-gray-800">USABANK</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">ID</td>
                      <td class="text-gray-800">id_89457jcje63</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <div class="flex-equal">
                <table class="table table-flush fw-semobold gy-1">
                  <tbody>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">
                        Billing address
                      </td>
                      <td class="text-gray-800">US</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Phone</td>
                      <td class="text-gray-800">No phone provided</td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Email</td>
                      <td class="text-gray-800">
                        <a href="#" class="text-gray-900 text-hover-primary"
                          >max@kt.com</a
                        >
                      </td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">Origin</td>
                      <td class="text-gray-800">
                        United States of America
                        <div class="symbol symbol-20px symbol-circle ms-2">
                          <img
                            :src="
                              getAssetPath('/media/flags/united-states.svg')
                            "
                          />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="text-muted min-w-125px w-125px">CVC check</td>
                      <td class="text-gray-800">
                        Failed

                        <span class="svg-icon svg-icon-2 svg-icon-danger">
                          <inline-svg
                            :src="
                              getAssetPath(
                                '/media/icons/duotune/arrows/arr061.svg'
                              )
                            "
                          />
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-payment-method",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
