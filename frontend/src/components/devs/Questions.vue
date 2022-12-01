<template>
  <!--begin::Questions-->
  <div class="mb-10">
    <template v-for="(question, i) in questions" :key="i">
      <!--begin::Question-->
      <div class="mb-0">
        <!--begin::Head-->
        <div class="d-flex align-items-center mb-4">
          <!--begin::Title-->
          <router-link
            to="/apps/devs/question"
            class="fs-2 fw-bold text-gray-900 text-hover-primary me-1"
          >
            {{ question.title }}
          </router-link>
          <!--end::Title-->

          <!--begin::Icons-->
          <div class="d-flex align-items-center">
            <template v-for="(icon, i) in question.icons" :key="i">
              <span class="ms-1" data-bs-toggle="tooltip" :title="icon.tooltip">
                <span :class="`svg-icon svg-icon-1 ${icon.class}`">
                  <inline-svg :src="icon.path" />
                </span>
              </span>
            </template>
          </div>
          <!--end::Icons-->
        </div>
        <!--end::Head-->

        <!--begin::Summary-->
        <div class="fs-base fw-normal text-gray-700 mb-4">
          {{ question.summary }}
        </div>
        <!--end::Summary-->

        <!--begin::Foot-->
        <div class="d-flex flex-stack flex-wrap">
          <!--begin::Author-->
          <div class="d-flex align-items-center py-1">
            <!--begin::Symbol-->
            <div class="symbol symbol-35px me-2">
              <img v-if="question.avatar" :src="question.avatar" alt="user" />
              <div
                v-else
                class="symbol-label bg-light-success fs-3 fw-semobold text-success text-uppercase"
              >
                {{ question.author[0] }}
              </div>
            </div>
            <!--end::Symbol-->

            <!--begin::Name-->
            <div
              class="d-flex flex-column align-items-start justify-content-center"
            >
              <span class="text-gray-900 fs-7 fw-semobold lh-1 mb-2">{{
                question.author
              }}</span>
              <span class="text-muted fs-8 fw-semobold lh-1">{{
                question.date
              }}</span>
            </div>
            <!--end::Name-->
          </div>
          <!--end::Author-->

          <!--begin::Info-->
          <div class="d-flex align-items-center py-1">
            <!--begin::Answers-->
            <a
              to="/apps/devs/question"
              class="btn btn-sm btn-outline btn-outline-dashed btn-outline-default px-4 me-2"
            >
              {{ question.answers }}
              Answers
            </a>
            <!--end::Answers-->

            <!--begin::Tags-->
            <template v-for="(tag, i) in question.tags" :key="i">
              <a href="#" class="btn btn-sm btn-light px-4 me-2">
                {{ tag }}
              </a>
            </template>
            <!--end::Tags-->

            <!--begin::Upvote-->
            <a
              href="#"
              class="btn btn-sm btn-flex btn-light"
              :class="`${question.upvotes ? 'btn-icons' : 'px-3'}`"
              data-bs-toggle="tooltip"
              title="Upvote this question"
              data-bs-dismiss="click"
            >
              {{ question.upvotes }}
              <span
                class="svg-icon svg-icon-7"
                :class="`${question.upvotes ? '' : 'ms-2 me-0'}`"
              >
                <inline-svg
                  :src="getAssetPath('/media/icons/duotune/arrows/arr062.svg')"
                />
              </span>
            </a>
            <!--end::Upvote-->
          </div>
          <!--end::Info-->
        </div>
        <!--end::Foot-->
      </div>
      <!--end::Question-->

      <!--begin::Separator-->
      <div class="separator separator-dashed border-gray-300 my-8"></div>
      <!--end::Separator-->
    </template>

    <div class="d-flex flex-center mb-0">
      <a
        href="#"
        class="btn btn-icon btn-light btn-active-light-primary h-30px w-30px fw-semobold fs-6 mx-2"
        >1</a
      >
      <a
        href="#"
        class="btn btn-icon btn-light btn-active-light-primary h-30px w-30px fw-semobold fs-6 mx-2 active"
        >2</a
      >
      <a
        href="#"
        class="btn btn-icon btn-light btn-active-light-primary h-30px w-30px fw-semobold fs-6 mx-2"
        >3</a
      >
      <a
        href="#"
        class="btn btn-icon btn-light btn-active-light-primary h-30px w-30px fw-semobold fs-6 mx-2"
        >4</a
      >
      <a
        href="#"
        class="btn btn-icon btn-light btn-active-light-primary h-30px w-30px fw-semobold fs-6 mx-2"
        >5</a
      >
      <span class="text-muted fw-semobold fs-6 mx-2">..</span>
      <a
        href="#"
        class="btn btn-icon btn-light btn-active-light-primary h-30px w-30px fw-semobold fs-6 mx-2"
        >19</a
      >
    </div>
  </div>
  <!--end::Questions-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref } from "vue";

interface IIcon {
  path: string;
  class: string;
  tooltip: string;
}

interface IQuestion {
  title: string;
  summary: string;
  author: string;
  date: string;
  avatar: string | undefined;
  answers: string;
  upvotes: string;
  icons: Array<IIcon>;
  tags: Array<string>;
}

export default defineComponent({
  name: "dev-questions",
  components: {},
  setup() {
    const questions = ref<Array<IQuestion>>([
      {
        title: "How to use Metronic with Django Framework ?",
        summary:
          "I’ve been doing some ajax request, to populate a inside drawer, the content of that drawer has a sub menu, that you are using in list and all card toolbar.",
        author: "James Hunt",
        date: "24 minutes ago",
        avatar: undefined,
        answers: "16",
        upvotes: "23",
        icons: [
          {
            path: getAssetPath("/media/icons/duotune/general/gen045.svg"),
            class: "svg-icon-primary",
            tooltip: "New question",
          },
          {
            path: getAssetPath("/media/icons/duotune/communication/com010.svg"),
            class: "svg-icon-danger",
            tooltip: "User replied",
          },
        ],
        tags: ["Metronic"],
      },
      {
        title: "When to expect new version of Laravel ?",
        summary:
          "When approx. is the next update for the Laravel version planned? Waiting for the CRUD, 2nd factor etc. features before starting my project. Also can we expect the Laravel + Vue version in the next update ?",
        author: "Sandra Piquet",
        date: "1 day ago",
        avatar: getAssetPath("/media/avatars/300-2.jpg"),
        answers: "2",
        upvotes: "4",
        icons: [
          {
            path: getAssetPath("/media/icons/duotune/general/gen044.svg"),
            class: "svg-icon-warning",
            tooltip: "In-process",
          },
        ],
        tags: ["Pre-sale"],
      },
      {
        title: "Could not get Demo 7 working",
        summary:
          "could not get demo7 working from latest metronic version. Had a lot of issues installing, I had to downgrade my npm to 6.14.4 as someone else recommended here in the comments, this goot it to compile but when I ran it, the browser showed errors TypeErr..",
        author: "Niko Roseberg",
        date: "2 days ago",
        avatar: undefined,
        answers: "4",
        upvotes: "",
        icons: [
          {
            path: getAssetPath("/media/icons/duotune/general/gen044.svg"),
            class: "svg-icon-warning",
            tooltip: "In-process",
          },
        ],
        tags: ["Angular"],
      },
      {
        title: "I want to get refund",
        summary:
          "Your Metronic theme is so good but the reactjs version is typescript only. The description did not write any warn about it. Since I only know javascript, I can not do anything with your theme. I want to refund.",
        author: "Alex Bold",
        date: "1 day ago",
        avatar: getAssetPath("/media/avatars/300-23.jpg"),
        answers: "22",
        upvotes: "11",
        icons: [
          {
            path: getAssetPath("/media/icons/duotune/general/gen043.svg"),
            class: "svg-icon-success",
            tooltip: "Resolved",
          },
        ],
        tags: ["React", "Demo 1"],
      },
      {
        title: "How to integrate Metronic with Blazor Server Side ?",
        summary:
          "could not get demo7 working from latest metronic version. Had a lot of issues installing, I had to downgrade my npm to 6.14.4 as someone else recommended here in the comments, this goot it to compile but when I ran it, the browser showed errors TypeErr..",
        author: "Tim Nilson",
        date: "3 days ago",
        avatar: undefined,
        answers: "44",
        upvotes: "3",
        icons: [
          {
            path: getAssetPath("/media/icons/duotune/general/gen043.svg"),
            class: "svg-icon-success",
            tooltip: "In-process",
          },
        ],
        tags: ["Blazor"],
      },
      {
        title: "Using Metronic with .NET multi tenant application",
        summary:
          "When approx. is the next update for the Laravel version planned? Waiting for the CRUD, 2nd factor etc. features before starting my project. Also can we expect the Laravel + Vue version in the next update ?",
        author: "Ana Quil",
        date: "5 days ago",
        avatar: getAssetPath("/media/avatars/300-10.jpg"),
        answers: "2",
        upvotes: "4",
        icons: [
          {
            path: getAssetPath("/media/icons/duotune/general/gen043.svg"),
            class: "svg-icon-success",
            tooltip: "Resolved",
          },
        ],
        tags: ["Aspdotnet"],
      },
    ]);

    return {
      questions,
      getAssetPath,
    };
  },
});
</script>
