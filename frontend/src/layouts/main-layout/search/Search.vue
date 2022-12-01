<template>
  <MenuComponent menu-selector="#kt-search-menu">
    <template v-slot:toggle>
      <!--begin::Search-->
      <div
        id="kt_header_search"
        class="d-flex align-items-stretch"
        data-kt-menu-target="#kt-search-menu"
        data-kt-menu-trigger="click"
        data-kt-menu-attach="parent"
        data-kt-menu-placement="bottom-end"
        data-kt-menu-flip="bottom"
      >
        <!--begin::Search toggle-->
        <div class="d-flex align-items-center" id="kt_header_search_toggle">
          <div class="btn btn-icon btn-active-light-primary">
            <span class="svg-icon svg-icon-1">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/general/gen021.svg')"
              />
            </span>
          </div>
        </div>
        <!--end::Search toggle-->
      </div>
      <!--end::Search-->
    </template>
    <template v-slot:content>
      <!--begin::Menu-->
      <div
        class="menu menu-sub menu-sub-dropdown menu-column p-7 w-325px w-md-375px"
        data-kt-menu="true"
        id="kt-search-menu"
      >
        <!--begin::Wrapper-->
        <div>
          <!--begin::Form-->
          <form class="w-100 position-relative mb-3" autocomplete="off">
            <!--begin::Icon-->
            <span
              class="svg-icon svg-icon-2 svg-icon-lg-1 svg-icon-gray-500 position-absolute top-50 translate-middle-y ms-0"
            >
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/general/gen021.svg')"
              />
            </span>
            <!--end::Icon-->

            <!--begin::Input-->
            <input
              ref="inputRef"
              v-model="search"
              @input="searching"
              type="text"
              class="form-control form-control-flush ps-10"
              name="search"
              placeholder="Search..."
            />
            <!--end::Input-->

            <!--begin::Spinner-->
            <span
              v-if="loading"
              class="position-absolute top-50 end-0 translate-middle-y lh-0 me-1"
            >
              <span
                class="spinner-border h-15px w-15px align-middle text-gray-400"
              ></span>
            </span>
            <!--end::Spinner-->

            <!--begin::Reset-->
            <span
              v-show="search.length && !loading"
              @click="reset()"
              class="btn btn-flush btn-active-color-primary position-absolute top-50 end-0 translate-middle-y lh-0"
            >
              <span class="svg-icon svg-icon-2 svg-icon-lg-1 me-0">
                <inline-svg
                  :src="getAssetPath('/media/icons/duotune/arrows/arr061.svg')"
                />
              </span>
            </span>
            <!--end::Reset-->

            <!--begin::Toolbar-->
            <div class="position-absolute top-50 end-0 translate-middle-y">
              <!--begin::Preferences toggle-->
              <div
                v-if="!search && !loading"
                @click="state = 'preferences'"
                class="btn btn-icon w-20px btn-sm btn-active-color-primary me-1"
                data-bs-toggle="tooltip"
                title="Show search preferences"
              >
                <span class="svg-icon svg-icon-1">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/coding/cod001.svg')
                    "
                  />
                </span>
              </div>
              <!--end::Preferences toggle-->

              <!--begin::Advanced search toggle-->
              <div
                v-if="!search && !loading"
                @click="state = 'advanced-options'"
                class="btn btn-icon w-20px btn-sm btn-active-color-primary"
                data-bs-toggle="tooltip"
                title="Show more search options"
              >
                <span class="svg-icon svg-icon-2">
                  <inline-svg
                    :src="
                      getAssetPath('/media/icons/duotune/arrows/arr072.svg')
                    "
                  />
                </span>
              </div>
              <!--end::Advanced search toggle-->
            </div>
            <!--end::Toolbar-->
          </form>
          <!--end::Form-->

          <!--begin::Separator-->
          <div class="separator border-gray-200 mb-6"></div>
          <!--end::Separator-->
          <Results v-if="state === 'results'"></Results>
          <PartialMain v-else-if="state === 'main'"></PartialMain>
          <Empty v-else-if="state === 'empty'"></Empty>
        </div>
        <!--end::Wrapper-->

        <form v-if="state === 'advanced-options'" class="pt-1">
          <!--begin::Heading-->
          <h3 class="fw-semobold text-dark mb-7">Advanced Search</h3>
          <!--end::Heading-->

          <!--begin::Input group-->
          <div class="mb-5">
            <input
              type="text"
              class="form-control form-control-sm form-control-solid"
              placeholder="Contains the word"
              name="query"
            />
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="mb-5">
            <!--begin::Radio group-->
            <div class="nav-group nav-group-fluid">
              <!--begin::Option-->
              <label>
                <input
                  type="radio"
                  class="btn-check"
                  name="type"
                  value="has"
                  checked
                />
                <span
                  class="btn btn-sm btn-color-muted btn-active btn-active-primary"
                >
                  All
                </span>
              </label>
              <!--end::Option-->

              <!--begin::Option-->
              <label>
                <input
                  type="radio"
                  class="btn-check"
                  name="type"
                  value="users"
                />
                <span
                  class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4"
                >
                  Users
                </span>
              </label>
              <!--end::Option-->

              <!--begin::Option-->
              <label>
                <input
                  type="radio"
                  class="btn-check"
                  name="type"
                  value="orders"
                />
                <span
                  class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4"
                >
                  Orders
                </span>
              </label>
              <!--end::Option-->

              <!--begin::Option-->
              <label>
                <input
                  type="radio"
                  class="btn-check"
                  name="type"
                  value="projects"
                />
                <span
                  class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4"
                >
                  Projects
                </span>
              </label>
              <!--end::Option-->
            </div>
            <!--end::Radio group-->
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="mb-5">
            <input
              type="text"
              name="assignedto"
              class="form-control form-control-sm form-control-solid"
              placeholder="Assigned to"
              value=""
            />
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="mb-5">
            <input
              type="text"
              name="collaborators"
              class="form-control form-control-sm form-control-solid"
              placeholder="Collaborators"
              value=""
            />
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="mb-5">
            <!--begin::Radio group-->
            <div class="nav-group nav-group-fluid">
              <!--begin::Option-->
              <label>
                <input
                  type="radio"
                  class="btn-check"
                  name="attachment"
                  value="has"
                  checked
                />
                <span
                  class="btn btn-sm btn-color-muted btn-active btn-active-primary"
                >
                  Has attachment
                </span>
              </label>
              <!--end::Option-->

              <!--begin::Option-->
              <label>
                <input
                  type="radio"
                  class="btn-check"
                  name="attachment"
                  value="any"
                />
                <span
                  class="btn btn-sm btn-color-muted btn-active btn-active-primary px-4"
                >
                  Any
                </span>
              </label>
              <!--end::Option-->
            </div>
            <!--end::Radio group-->
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="mb-5">
            <select
              name="timezone"
              aria-label="Select a Timezone"
              data-control="select2"
              data-placeholder="date_period"
              class="form-select form-select-sm form-select-solid"
            >
              <option value="next">Within the next</option>
              <option value="last">Within the last</option>
              <option value="between">Between</option>
              <option value="on">On</option>
            </select>
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="row mb-8">
            <!--begin::Col-->
            <div class="col-6">
              <input
                type="number"
                name="date_number"
                class="form-control form-control-sm form-control-solid"
                placeholder="Lenght"
                value=""
              />
            </div>
            <!--end::Col-->

            <!--begin::Col-->
            <div class="col-6">
              <select
                name="date_typer"
                aria-label="Select a Timezone"
                data-control="select2"
                data-placeholder="Period"
                class="form-select form-select-sm form-select-solid"
              >
                <option value="days">Days</option>
                <option value="weeks">Weeks</option>
                <option value="months">Months</option>
                <option value="years">Years</option>
              </select>
            </div>
            <!--end::Col-->
          </div>
          <!--end::Input group-->

          <!--begin::Actions-->
          <div class="d-flex justify-content-end">
            <button
              @click="state = 'main'"
              class="btn btn-sm btn-light fw-bold btn-active-light-primary me-2"
            >
              Cancel
            </button>

            <a href="#" class="btn btn-sm fw-bold btn-primary">Search</a>
          </div>
          <!--end::Actions-->
        </form>

        <form v-if="state === 'preferences'" class="pt-1">
          <!--begin::Heading-->
          <h3 class="fw-semobold text-dark mb-7">Search Preferences</h3>
          <!--end::Heading-->

          <!--begin::Input group-->
          <div class="pb-4 border-bottom">
            <label
              class="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack"
            >
              <span
                class="form-check-label text-gray-700 fs-6 fw-semobold ms-0 me-2"
              >
                Projects
              </span>

              <input
                class="form-check-input"
                type="checkbox"
                value="1"
                checked
              />
            </label>
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="py-4 border-bottom">
            <label
              class="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack"
            >
              <span
                class="form-check-label text-gray-700 fs-6 fw-semobold ms-0 me-2"
              >
                Targets
              </span>
              <input
                class="form-check-input"
                type="checkbox"
                value="1"
                checked
              />
            </label>
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="py-4 border-bottom">
            <label
              class="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack"
            >
              <span
                class="form-check-label text-gray-700 fs-6 fw-semobold ms-0 me-2"
              >
                Affiliate Programs
              </span>
              <input class="form-check-input" type="checkbox" value="1" />
            </label>
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="py-4 border-bottom">
            <label
              class="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack"
            >
              <span
                class="form-check-label text-gray-700 fs-6 fw-semobold ms-0 me-2"
              >
                Referrals
              </span>
              <input
                class="form-check-input"
                type="checkbox"
                value="1"
                checked
              />
            </label>
          </div>
          <!--end::Input group-->

          <!--begin::Input group-->
          <div class="py-4 border-bottom">
            <label
              class="form-check form-switch form-switch-sm form-check-custom form-check-solid flex-stack"
            >
              <span
                class="form-check-label text-gray-700 fs-6 fw-semobold ms-0 me-2"
              >
                Users
              </span>
              <input class="form-check-input" type="checkbox" value="1" />
            </label>
          </div>
          <!--end::Input group-->

          <!--begin::Actions-->
          <div class="d-flex justify-content-end pt-7">
            <div
              @click="state = 'main'"
              class="btn btn-sm btn-light fw-bold btn-active-light-primary me-2"
            >
              Cancel
            </div>
            <button class="btn btn-sm fw-bold btn-primary">Save Changes</button>
          </div>
          <!--end::Actions-->
        </form>
      </div>
      <!--end::Menu-->
    </template>
  </MenuComponent>
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref } from "vue";
import Results from "@/layouts/main-layout/search/partials/Results.vue";
import PartialMain from "@/layouts/main-layout/search/partials/Main.vue";
import Empty from "@/layouts/main-layout/search/partials/Empty.vue";
import MenuComponent from "@/components/menu/MenuComponent.vue";

export default defineComponent({
  name: "kt-search",
  components: {
    Results,
    PartialMain,
    Empty,
    MenuComponent,
  },
  setup() {
    const search = ref<string>("");
    const state = ref<
      "main" | "empty" | "advanced-options" | "preferences" | "results"
    >("main");
    const loading = ref<boolean>(false);
    const inputRef = ref<HTMLInputElement | null>(null);

    const searching = (e: Event) => {
      const target = e.target as HTMLInputElement;
      if (target.value.length <= 1) {
        load("main");
      } else {
        if (target.value.length > 5) {
          load("empty");
          return;
        }
        load("results");
      }
    };

    const load = (
      current: "main" | "empty" | "advanced-options" | "preferences" | "results"
    ) => {
      loading.value = true;
      setTimeout(() => {
        state.value = current;
        loading.value = false;
      }, 1000);
    };

    const reset = () => {
      search.value = "";
      state.value = "main";
    };

    const setState = (
      curr: "main" | "empty" | "advanced-options" | "preferences" | "results"
    ) => {
      state.value = curr;
    };

    return {
      search,
      state,
      loading,
      searching,
      reset,
      inputRef,
      setState,
      getAssetPath,
    };
  },
});
</script>
