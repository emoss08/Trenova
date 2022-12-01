<template>
  <!--begin::Heading-->
  <a id="answers" data-kt-scroll-offset="{default: 100, lg: 125}"></a>
  <h2 class="fw-bold text-gray-900 mb-10">Replies(14)</h2>
  <!--end::Heading-->

  <!--begin::Replies-->
  <div class="mb-10">
    <template v-for="(reply, i) in replies" :key="i">
      <!--begin::Reply-->
      <div :class="`border rounded p-2 p-lg-6 mb-10 ${reply.indent}`">
        <!--begin::Wrapper-->
        <div class="mb-0">
          <!--begin::Foot-->
          <div class="d-flex flex-stack flex-wrap mb-5">
            <!--begin::Author-->
            <div class="d-flex align-items-center py-1">
              <!--begin::Symbol-->
              <div class="symbol symbol-35px me-2">
                <img v-if="reply.avatar" :src="reply.avatar" alt="user" />
                <div
                  v-else
                  class="symbol-label bg-light-success fs-3 fw-semobold text-success text-uppercase"
                >
                  {{ reply.author[0] }}
                </div>
              </div>
              <!--end::Symbol-->

              <!--begin::Name-->
              <div
                class="d-flex flex-column align-items-start justify-content-center"
              >
                <span class="text-gray-800 fs-7 fw-semobold lh-1 mb-2">{{
                  reply.author
                }}</span>
                <span class="text-muted fs-8 fw-semobold lh-1">{{
                  reply.date
                }}</span>
              </div>
              <!--end::Name-->
            </div>
            <!--end::Author-->

            <!--begin::Info-->
            <div class="d-flex align-items-center py-1">
              <!--begin::Reply-->
              <a
                href="#"
                class="btn btn-sm btn-flex btn-color-gray-500 btn-active-light me-1"
              >
                Reply
              </a>
              <!--end::Reply-->

              <!--begin::Upvote-->
              <a
                href="#"
                :class="`btn btn-sm btn-flex btn-light ${
                  reply.upvotes ? 'btn-icon' : 'px-3'
                }`"
                data-bs-toggle="tooltip"
                title="Upvote"
                data-bs-dismiss="click"
              >
                {{ reply.upvotes }}
                <span
                  :class="`svg-icon svg-icon-7 ${
                    reply.upvotes ? '' : 'mx-2 me-0'
                  }`"
                >
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/arrows/arr062.svg')
                    "
                  />
                </span>
              </a>
              <!--end::Upvote-->
            </div>
            <!--end::Info-->
          </div>
          <!--end::Foot-->

          <!--begin::Message-->
          <div class="fs-5 fw-normal text-gray-800">
            {{ reply.message }}
          </div>
          <!--end::Message-->
        </div>
        <!--end::Wrapper-->

        <!--begin::Replies-->
        <div class="ps-10 mb-0"></div>
        <!--end::Replies-->
      </div>
      <!--end::Reply-->
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
  <!--end::Replies-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref } from "vue";

export default defineComponent({
  name: "dev-replies",
  components: {},
  setup() {
    const replies = ref([
      {
        message:
          "I’ve been doing some ajax request, to populate a inside drawer, the content of that drawer has a sub menu, that you are using in list and all card toolbar.",
        author: "Sandra Piquet",
        date: "24 minutes ago",
        avatar: getAssetPath("/media/avatars/300-2.jpg"),
        upvotes: "",
        indent: "",
      },
      {
        message:
          "I’ve been doing some ajax request, to populate a inside drawer, the content of that drawer has a sub menu, that you are using in list and all card toolbar.",
        author: "Niko Roseberg",
        date: "1 day ago",
        avatar: undefined,
        upvotes: "2",
        indent: "ms-5 ms-lg-10",
      },
      {
        message:
          "I’ve been doing some ajax request, to populate a inside drawer, the content of that drawer has a sub menu, that you are using in list and all card toolbar.",
        author: "Sandra Piquet",
        date: "24 minutes ago",
        avatar: getAssetPath("/media/avatars/300-12.jpg"),
        upvotes: "6",
        indent: "",
      },
      {
        message:
          "I’ve been doing some ajax request, to populate a inside drawer, the content of that drawer has a sub menu, that you are using in list and all card toolbar.",
        author: "Sandra Piquet",
        date: "24 minutes ago",
        avatar: undefined,
        upvotes: "4",
        indent: "",
      },
      {
        message:
          "I’ve been doing some ajax request, to populate a inside drawer, the content of that drawer has a sub menu, that you are using in list and all card toolbar.",
        author: "Niko Roseberg",
        date: "1 day ago",
        avatar: getAssetPath("/media/avatars/300-20.jpg"),
        upvotes: "",
        indent: "",
      },
    ]);

    return {
      replies,
      getAssetPath,
    };
  },
});
</script>
