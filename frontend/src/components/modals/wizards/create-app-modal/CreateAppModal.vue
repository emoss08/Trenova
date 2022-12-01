<template>
  <div
    class="modal fade"
    id="kt_modal_create_app"
    ref="createAppModalRef"
    tabindex="-1"
    aria-hidden="true"
  >
    <!--begin::Modal dialog-->
    <div class="modal-dialog modal-dialog-centered mw-900px">
      <!--begin::Modal content-->
      <div class="modal-content">
        <!--begin::Modal header-->
        <div class="modal-header">
          <!--begin::Modal title-->
          <h2>Create App</h2>
          <!--end::Modal title-->

          <!--begin::Close-->
          <div
            class="btn btn-sm btn-icon btn-active-color-primary"
            data-bs-dismiss="modal"
          >
            <span class="svg-icon svg-icon-1">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr061.svg')"
              />
            </span>
          </div>
          <!--end::Close-->
        </div>
        <!--end::Modal header-->

        <!--begin::Modal body-->
        <div class="modal-body py-lg-10 px-lg-10">
          <!--begin::Stepper-->
          <div
            class="stepper stepper-pills stepper-column d-flex flex-column flex-xl-row flex-row-fluid"
            id="kt_modal_create_app_stepper"
            ref="createAppRef"
          >
            <!--begin::Aside-->
            <div
              class="d-flex justify-content-center justify-content-xl-start flex-row-auto w-100 w-xl-300px"
            >
              <!--begin::Nav-->
              <div class="stepper-nav ps-lg-10">
                <!--begin::Step 1-->
                <div class="stepper-item current" data-kt-stepper-element="nav">
                  <div class="stepper-wrapper">
                    <!--begin::Icon-->
                    <div class="stepper-icon w-40px h-40px">
                      <i class="stepper-check fas fa-check"></i>
                      <span class="stepper-number">1</span>
                    </div>
                    <!--end::Icon-->

                    <!--begin::Label-->
                    <div class="stepper-label">
                      <h3 class="stepper-title">Details</h3>

                      <div class="stepper-desc">Name your App</div>
                    </div>
                    <!--end::Label-->
                  </div>

                  <!--begin::Line-->
                  <div class="stepper-line h-40px"></div>
                  <!--end::Line-->
                </div>
                <!--end::Step 1-->

                <!--begin::Step 2-->
                <div class="stepper-item" data-kt-stepper-element="nav">
                  <div class="stepper-wrapper">
                    <!--begin::Icon-->
                    <div class="stepper-icon w-40px h-40px">
                      <i class="stepper-check fas fa-check"></i>
                      <span class="stepper-number">2</span>
                    </div>
                    <!--begin::Icon-->

                    <!--begin::Label-->
                    <div class="stepper-label">
                      <h3 class="stepper-title">Frameworks</h3>

                      <div class="stepper-desc">Define your app framework</div>
                    </div>
                    <!--begin::Label-->
                  </div>

                  <!--begin::Line-->
                  <div class="stepper-line h-40px"></div>
                  <!--end::Line-->
                </div>
                <!--end::Step 2-->

                <!--begin::Step 3-->
                <div class="stepper-item" data-kt-stepper-element="nav">
                  <div class="stepper-wrapper">
                    <!--begin::Icon-->
                    <div class="stepper-icon w-40px h-40px">
                      <i class="stepper-check fas fa-check"></i>
                      <span class="stepper-number">3</span>
                    </div>
                    <!--end::Icon-->

                    <!--begin::Label-->
                    <div class="stepper-label">
                      <h3 class="stepper-title">Database</h3>

                      <div class="stepper-desc">
                        Select the app database type
                      </div>
                    </div>
                    <!--end::Label-->
                  </div>
                  <!--begin::Line-->
                  <div class="stepper-line h-40px"></div>
                  <!--end::Line-->
                </div>
                <!--end::Step 3-->

                <!--begin::Step 4-->
                <div class="stepper-item" data-kt-stepper-element="nav">
                  <div class="stepper-wrapper">
                    <!--begin::Icon-->
                    <div class="stepper-icon w-40px h-40px">
                      <i class="stepper-check fas fa-check"></i>
                      <span class="stepper-number">4</span>
                    </div>
                    <!--end::Icon-->

                    <!--begin::Label-->
                    <div class="stepper-label">
                      <h3 class="stepper-title">Billing</h3>

                      <div class="stepper-desc">Provide payment details</div>
                    </div>
                    <!--end::Label-->
                  </div>

                  <!--begin::Line-->
                  <div class="stepper-line h-40px"></div>
                  <!--end::Line-->
                </div>
                <!--end::Step 4-->

                <!--begin::Step 5-->
                <div class="stepper-item" data-kt-stepper-element="nav">
                  <div class="stepper-wrapper">
                    <!--begin::Icon-->
                    <div class="stepper-icon w-40px h-40px">
                      <i class="stepper-check fas fa-check"></i>
                      <span class="stepper-number">5</span>
                    </div>
                    <!--end::Icon-->

                    <!--begin::Label-->
                    <div class="stepper-label">
                      <h3 class="stepper-title">Release</h3>

                      <div class="stepper-desc">Review and Submit</div>
                    </div>
                    <!--end::Label-->
                  </div>
                </div>
                <!--end::Step 5-->
              </div>
              <!--end::Nav-->
            </div>
            <!--begin::Aside-->

            <!--begin::Content-->
            <div class="flex-row-fluid py-lg-5 px-lg-15">
              <!--begin::Form-->
              <CreateAppModalForm />
              <!--end::Form-->
            </div>
            <!--end::Content-->
          </div>
          <!--end::Stepper-->
        </div>
        <!--end::Modal body-->
      </div>
      <!--end::Modal content-->
    </div>
    <!--end::Modal dialog-->
  </div>
  <!--end::Modal - Create App-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import CreateAppModalForm from "@/components/modals/wizards/create-app-modal/CreateAppModalForm.vue";

export default defineComponent({
  name: "create-app-modal",
  components: {
    CreateAppModalForm,
  },
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
