<template>
  <div
    class="modal fade show"
    id="kt_modal_add_event"
    aria-modal="true"
    role="dialog"
    ref="newTargetModalRef"
  >
    <div class="modal-dialog modal-dialog-centered mw-650px">
      <div class="modal-content">
        <el-form
          class="form fv-plugins-bootstrap5 fv-plugins-framework"
          id="kt_modal_add_event_form"
          @submit.prevent="submit()"
          :model="targetData"
          :rules="rules"
          ref="formRef"
        >
          <div class="modal-header">
            <h2 class="fw-bold">Add a New Event</h2>
            <div
              class="btn btn-icon btn-sm btn-active-icon-primary"
              id="kt_modal_add_event_close"
              data-bs-dismiss="modal"
            >
              <span class="svg-icon svg-icon-1">
                <inline-svg
                  :src="getAssetPath('/media/icons/duotune/arrows/arr061.svg')"
                />
              </span>
            </div>
          </div>
          <!--end::Modal header-->
          <!--begin::Modal body-->
          <div class="modal-body py-10 px-lg-17">
            <!--begin::Input group-->
            <div class="fv-row mb-9 fv-plugins-icon-container">
              <!--begin::Label-->
              <label class="fs-6 fw-semobold required mb-2">Event Name</label>
              <!--end::Label-->
              <!--begin::Input-->
              <el-form-item prop="eventName">
                <el-input
                  v-model="targetData.eventName"
                  type="text"
                  name="eventName"
                />
              </el-form-item>
              <!--end::Input-->
              <div class="fv-plugins-message-container invalid-feedback"></div>
            </div>
            <!--end::Input group-->
            <!--begin::Input group-->
            <div class="fv-row mb-9">
              <!--begin::Label-->
              <label class="fs-6 fw-semobold mb-2">Event Description</label>
              <!--end::Label-->
              <!--begin::Input-->
              <el-input
                v-model="targetData.eventDescription"
                type="text"
                placeholder=""
                name="eventDescription"
              />
              <!--end::Input-->
            </div>
            <!--end::Input group-->
            <!--begin::Input group-->
            <div class="fv-row mb-9">
              <!--begin::Label-->
              <label class="fs-6 fw-semobold mb-2">Event Location</label>
              <!--end::Label-->
              <!--begin::Input-->
              <el-input
                v-model="targetData.eventLocation"
                type="text"
                placeholder=""
                name="eventLocation"
              />
              <!--end::Input-->
            </div>
            <!--end::Input group-->
            <!--begin::Input group-->
            <div class="fv-row mb-9">
              <!--begin::Checkbox-->
              <label class="form-check form-check-custom form-check-solid">
                <el-checkbox v-model="targetData.allDay" type="checkbox" />
                <span class="form-check-label fw-semobold">All Day</span>
              </label>
              <!--end::Checkbox-->
            </div>
            <!--end::Input group-->
            <!--begin::Input group-->
            <div class="row row-cols-lg-2 g-10">
              <div class="col">
                <div
                  class="fv-row mb-9 fv-plugins-icon-container fv-plugins-bootstrap5-row-valid"
                >
                  <!--begin::Label-->
                  <label class="fs-6 fw-semobold mb-2 required"
                    >Event Start Date</label
                  >
                  <!--end::Label-->
                  <!--begin::Input-->
                  <el-date-picker
                    v-model="targetData.eventStartDate"
                    type="date"
                    :teleported="false"
                    name="eventStartDate"
                  />
                  <!--end::Input-->
                  <div
                    class="fv-plugins-message-container invalid-feedback"
                  ></div>
                </div>
              </div>
            </div>
            <!--end::Input group-->
            <!--begin::Input group-->
            <div class="row row-cols-lg-2 g-10">
              <div class="col">
                <div
                  class="fv-row mb-9 fv-plugins-icon-container fv-plugins-bootstrap5-row-valid"
                >
                  <!--begin::Label-->
                  <label class="fs-6 fw-semobold mb-2 required"
                    >Event End Date</label
                  >
                  <!--end::Label-->
                  <!--begin::Input-->
                  <el-date-picker
                    v-model="targetData.eventEndDate"
                    type="date"
                    :teleported="false"
                    name="eventName"
                  />
                  <!--end::Input-->
                  <div
                    class="fv-plugins-message-container invalid-feedback"
                  ></div>
                </div>
              </div>
            </div>
            <!--end::Input group-->
          </div>
          <!--end::Modal body-->
          <!--begin::Modal footer-->
          <div class="modal-footer flex-center">
            <!--begin::Button-->
            <button
              data-bs-dismiss="modal"
              type="reset"
              id="kt_modal_add_event_cancel"
              class="btn btn-light me-3"
            >
              Cancel
            </button>
            <!--end::Button-->
            <!--begin::Button-->
            <button
              :data-kt-indicator="loading ? 'on' : null"
              class="btn btn-lg btn-primary"
              type="submit"
            >
              <span v-if="!loading" class="indicator-label">
                Submit
                <span class="svg-icon svg-icon-3 ms-2 me-0">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/arrows/arr064.svg')
                    "
                  />
                </span>
              </span>
              <span v-if="loading" class="indicator-progress">
                Please wait...
                <span
                  class="spinner-border spinner-border-sm align-middle ms-2"
                ></span>
              </span>
            </button>
            <!--end::Button-->
          </div>
          <!--end::Modal footer-->
          <div></div>
        </el-form>
        <!--end::Form-->
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref } from "vue";
import { hideModal } from "@/core/helpers/dom";
import Swal from "sweetalert2";

interface NewAddressData {
  eventName: string;
  eventDescription: string;
  eventLocation: string;
  allDay: boolean;
  eventStartDate: string;
  eventEndDate: string;
}

export default defineComponent({
  name: "new-event-modal",
  components: {},
  setup() {
    const formRef = ref<null | HTMLFormElement>(null);
    const newTargetModalRef = ref<null | HTMLElement>(null);
    const loading = ref<boolean>(false);

    const targetData = ref<NewAddressData>({
      eventName: "",
      eventDescription: "",
      eventLocation: "",
      allDay: true,
      eventStartDate: "",
      eventEndDate: "",
    });

    const rules = ref({
      eventName: [
        {
          required: true,
          message: "Please input event name",
          trigger: "blur",
        },
      ],
    });

    const submit = () => {
      if (!formRef.value) {
        return;
      }

      formRef.value.validate((valid: boolean) => {
        if (valid) {
          loading.value = true;

          setTimeout(() => {
            loading.value = false;

            Swal.fire({
              text: "Form has been successfully submitted!",
              icon: "success",
              buttonsStyling: false,
              confirmButtonText: "Ok, got it!",
              heightAuto: false,
              customClass: {
                confirmButton: "btn btn-primary",
              },
            }).then(() => {
              hideModal(newTargetModalRef.value);
            });
          }, 2000);
        } else {
          Swal.fire({
            text: "Sorry, looks like there are some errors detected, please try again.",
            icon: "error",
            buttonsStyling: false,
            confirmButtonText: "Ok, got it!",
            heightAuto: false,
            customClass: {
              confirmButton: "btn btn-primary",
            },
          });
          return false;
        }
      });
    };

    return {
      formRef,
      newTargetModalRef,
      loading,
      targetData,
      rules,
      submit,
      getAssetPath,
    };
  },
});
</script>

<style lang="scss">
.el-select {
  width: 100%;
}

.el-date-editor.el-input,
.el-date-editor.el-input__inner {
  width: 100%;
}
</style>
