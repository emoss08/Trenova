<template>
  <!--begin::Timeline-->
  <div class="card">
    <!--begin::Card head-->
    <div class="card-header card-header-stretch">
      <!--begin::Title-->
      <div class="card-title d-flex align-items-center">
        <span class="svg-icon svg-icon-1 svg-icon-primary me-3 lh-0">
          <inline-svg
            :src="getAssetPath('/media/icons/duotune/general/gen014.svg')"
          />
        </span>

        <h3 class="fw-bold m-0 text-gray-800">Jan 23, 2021</h3>
      </div>
      <!--end::Title-->

      <!--begin::Toolbar-->
      <div class="card-toolbar m-0">
        <!--begin::Tab nav-->
        <ul
          class="nav nav-tabs nav-line-tabs nav-stretch fs-6 border-0 fw-bold"
          role="tablist"
        >
          <li class="nav-item" role="presentation">
            <a
              id="kt_activity_today_tab"
              class="nav-link justify-content-center text-active-gray-800 active"
              data-bs-toggle="tab"
              role="tab"
              href="#kt_activity_today"
            >
              Today
            </a>
          </li>
          <li class="nav-item" role="presentation">
            <a
              id="kt_activity_week_tab"
              class="nav-link justify-content-center text-active-gray-800"
              data-bs-toggle="tab"
              role="tab"
              href="#kt_activity_week"
            >
              Week
            </a>
          </li>
          <li class="nav-item" role="presentation">
            <a
              id="kt_activity_month_tab"
              class="nav-link justify-content-center text-active-gray-800"
              data-bs-toggle="tab"
              role="tab"
              href="#kt_activity_month"
            >
              Month
            </a>
          </li>
          <li class="nav-item" role="presentation">
            <a
              id="kt_activity_year_tab"
              class="nav-link justify-content-center text-active-gray-800 text-hover-gray-800"
              data-bs-toggle="tab"
              role="tab"
              href="#kt_activity_year"
            >
              2021
            </a>
          </li>
        </ul>
        <!--end::Tab nav-->
      </div>
      <!--end::Toolbar-->
    </div>
    <!--end::Card head-->

    <!--begin::Card body-->
    <div class="card-body">
      <!--begin::Tab Content-->
      <div class="tab-content">
        <!--begin::Tab panel-->
        <div
          id="kt_activity_today"
          class="card-body p-0 tab-pane fade show active"
          role="tabpanel"
          aria-labelledby="kt_activity_today_tab"
        >
          <!--begin::Timeline-->
          <div class="timeline">
            <KTActivityItem1></KTActivityItem1>
            <KTActivityItem2></KTActivityItem2>
            <KTActivityItem3></KTActivityItem3>
            <KTActivityItem4></KTActivityItem4>
            <KTActivityItem5></KTActivityItem5>
            <KTActivityItem6></KTActivityItem6>
            <KTActivityItem7></KTActivityItem7>
            <KTActivityItem8></KTActivityItem8>
          </div>
          <!--end::Timeline-->
        </div>
        <!--end::Tab panel-->

        <!--begin::Tab panel-->
        <div
          id="kt_activity_week"
          class="card-body p-0 tab-pane fade show"
          role="tabpanel"
          aria-labelledby="kt_activity_week_tab"
        >
          <!--begin::Timeline-->
          <div class="timeline">
            <KTActivityItem2></KTActivityItem2>
            <KTActivityItem3></KTActivityItem3>
            <KTActivityItem4></KTActivityItem4>
            <KTActivityItem5></KTActivityItem5>
            <KTActivityItem6></KTActivityItem6>
          </div>
          <!--end::Timeline-->
        </div>
        <!--end::Tab panel-->

        <!--begin::Tab panel-->
        <div
          id="kt_activity_month"
          class="card-body p-0 tab-pane fade show"
          role="tabpanel"
          aria-labelledby="kt_activity_month_tab"
        >
          <!--begin::Timeline-->
          <div class="timeline">
            <KTActivityItem5></KTActivityItem5>
            <KTActivityItem6></KTActivityItem6>
            <KTActivityItem8></KTActivityItem8>
            <KTActivityItem2></KTActivityItem2>
            <KTActivityItem3></KTActivityItem3>
            <KTActivityItem4></KTActivityItem4>
          </div>
          <!--end::Timeline-->
        </div>
        <!--end::Tab panel-->

        <!--begin::Tab panel-->
        <div
          id="kt_activity_year"
          class="card-body p-0 tab-pane fade show"
          role="tabpanel"
          aria-labelledby="kt_activity_year_tab"
        >
          <!--begin::Timeline-->
          <div class="timeline">
            <KTActivityItem3></KTActivityItem3>
            <KTActivityItem4></KTActivityItem4>
            <KTActivityItem5></KTActivityItem5>
            <KTActivityItem6></KTActivityItem6>
          </div>
          <!--end::Timeline-->
        </div>
        <!--end::Tab panel-->
      </div>
      <!--end::Tab Content-->
    </div>
    <!--end::Card body-->
  </div>
  <!--end::Timeline-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent } from "vue";
import KTActivityItem1 from "@/components/activity-timeline-items/Item1.vue";
import KTActivityItem2 from "@/components/activity-timeline-items/Item2.vue";
import KTActivityItem3 from "@/components/activity-timeline-items/Item3.vue";
import KTActivityItem4 from "@/components/activity-timeline-items/Item4.vue";
import KTActivityItem5 from "@/components/activity-timeline-items/Item5.vue";
import KTActivityItem6 from "@/components/activity-timeline-items/Item6.vue";
import KTActivityItem7 from "@/components/activity-timeline-items/Item7.vue";
import KTActivityItem8 from "@/components/activity-timeline-items/Item8.vue";

export default defineComponent({
  name: "profile-activity",
  components: {
    KTActivityItem1,
    KTActivityItem2,
    KTActivityItem3,
    KTActivityItem4,
    KTActivityItem5,
    KTActivityItem6,
    KTActivityItem7,
    KTActivityItem8,
  },
  setup() {
    return {
      getAssetPath,
    };
  },
});
</script>
