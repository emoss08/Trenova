<template>
  <div ref="resultsElementRef" data-kt-search-element="results" class="d-none">
    <!--begin::Items-->
    <div class="scroll-y mh-200px mh-lg-325px">
      <!--begin::Category title-->
      <h3 class="fs-5 text-muted m-0 pb-5">Users</h3>
      <!--end::Category title-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <img :src="getAssetPath('/media/avatars/300-6.jpg')" alt="" />
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Karina Clark</span>
          <span class="fs-7 fw-semobold text-muted">Marketing Manager</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <img :src="getAssetPath('/media/avatars/300-2.jpg')" alt="" />
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Olivia Bold</span>
          <span class="fs-7 fw-semobold text-muted">Software Engineer</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <img :src="getAssetPath('/media/avatars/300-9.jpg')" alt="" />
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Ana Clark</span>
          <span class="fs-7 fw-semobold text-muted">UI/UX Designer</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <img :src="getAssetPath('/media/avatars/300-14.jpg')" alt="" />
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Nick Pitola</span>
          <span class="fs-7 fw-semobold text-muted">Art Director</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <img :src="getAssetPath('/media/avatars/300-11.jpg')" alt="" />
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Edward Kulnic</span>
          <span class="fs-7 fw-semobold text-muted">System Administrator</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Category title-->
      <h3 class="fs-5 text-muted m-0 pt-5 pb-5">Customers</h3>
      <!--end::Category title-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <img
              class="w-20px h-20px"
              :src="getAssetPath('/media/svg/brand-logos/volicity-9.svg')"
              alt=""
            />
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Company Rbranding</span>
          <span class="fs-7 fw-semobold text-muted">UI Design</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <img
              class="w-20px h-20px"
              :src="getAssetPath('/media/svg/brand-logos/tvit.svg')"
              alt=""
            />
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Company Re-branding</span>
          <span class="fs-7 fw-semobold text-muted">Web Development</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <img
              class="w-20px h-20px"
              :src="getAssetPath('/media/svg/misc/infography.svg')"
              alt=""
            />
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Business Analytics App</span>
          <span class="fs-7 fw-semobold text-muted">Administration</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <img
              class="w-20px h-20px"
              :src="getAssetPath('/media/svg/brand-logos/leaf.svg')"
              alt=""
            />
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">EcoLeaf App Launch</span>
          <span class="fs-7 fw-semobold text-muted">Marketing</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <img
              class="w-20px h-20px"
              :src="getAssetPath('/media/svg/brand-logos/tower.svg')"
              alt=""
            />
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column justify-content-start fw-semobold">
          <span class="fs-6 fw-semobold">Tower Group Website</span>
          <span class="fs-7 fw-semobold text-muted">Google Adwords</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Category title-->
      <h3 class="fs-5 text-muted m-0 pt-5 pb-5">Projects</h3>
      <!--end::Category title-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/general/gen005.svg')"
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <span class="fs-6 fw-semobold">Si-Fi Project by AU Themes</span>
          <span class="fs-7 fw-semobold text-muted">#45670</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/general/gen032.svg')"
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <span class="fs-6 fw-semobold">Shopix Mobile App Planning</span>
          <span class="fs-7 fw-semobold text-muted">#45690</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="
                  getAssetPath('/media/icons/duotune/communication/com012.svg')
                "
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <span class="fs-6 fw-semobold"
            >Finance Monitoring SAAS Discussion</span
          >
          <span class="fs-7 fw-semobold text-muted">#21090</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
      <!--begin::Item-->
      <a
        href="#"
        class="d-flex text-dark text-hover-primary align-items-center mb-5"
      >
        <!--begin::Symbol-->
        <div class="symbol symbol-40px symbol-circle me-4">
          <span class="symbol-label bg-light">
            <span class="svg-icon svg-icon-2 svg-icon-primary">
              <inline-svg
                :src="
                  getAssetPath('/media/icons/duotune/communication/com006.svg')
                "
              />
            </span>
          </span>
        </div>
        <!--end::Symbol-->
        <!--begin::Title-->
        <div class="d-flex flex-column">
          <span class="fs-6 fw-semobold">Dashboard Analitics Launch</span>
          <span class="fs-7 fw-semobold text-muted">#34560</span>
        </div>
        <!--end::Title-->
      </a>
      <!--end::Item-->
    </div>
    <!--end::Items-->
  </div>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";

export default defineComponent({
  name: "kt-results",
  components: {},
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
