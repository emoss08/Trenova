<template>
  <!--begin::Card-->
  <div class="card">
    <!--begin::Card header-->
    <div class="card-header border-0 pt-6">
      <!--begin::Card title-->
      <div class="card-title">
        <!--begin::Search-->
        <div class="d-flex align-items-center position-relative my-1">
          <span class="svg-icon svg-icon-1 position-absolute ms-6">
            <inline-svg
              :src="getAssetPath('/media/icons/duotune/general/gen021.svg')"
            />
          </span>
          <input
            type="text"
            data-kt-subscription-table-filter="search"
            class="form-control form-control-solid w-250px ps-14"
            placeholder="Search Subscriptions"
          />
        </div>
        <!--end::Search-->
      </div>
      <!--begin::Card title-->

      <!--begin::Card toolbar-->
      <div class="card-toolbar">
        <!--begin::Toolbar-->
        <div
          v-if="selectedIds.length === 0"
          class="d-flex justify-content-end"
          data-kt-subscription-table-toolbar="base"
        >
          <!--begin::Export-->
          <button
            type="button"
            class="btn btn-light-primary me-3"
            data-bs-toggle="modal"
            data-bs-target="#kt_subscriptions_export_modal"
          >
            <span class="svg-icon svg-icon-2">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr078.svg')"
              />
            </span>
            Export
          </button>
          <!--end::Export-->

          <!--begin::Add subscription-->
          <router-link
            to="/apps/subscriptions/add-subscription"
            class="btn btn-primary"
          >
            <span class="svg-icon svg-icon-2">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr075.svg')"
              />
            </span>
            Add Subscription
          </router-link>
          <!--end::Add subscription-->
        </div>
        <!--end::Toolbar-->

        <!--begin::Group actions-->
        <div v-else class="d-flex justify-content-end align-items-center">
          <div class="fw-bold me-5">
            <span class="me-2">{{ selectedIds.length }}</span
            >Selected
          </div>
          <button
            type="button"
            class="btn btn-danger"
            @click="deleteFewSubscriptions()"
          >
            Delete Selected
          </button>
        </div>
        <!--end::Group actions-->
      </div>
      <!--end::Card toolbar-->
    </div>
    <!--end::Card header-->

    <!--begin::Card body-->
    <div class="card-body pt-0">
      <KTDatatable
        @on-sort="sort"
        @on-items-select="onItemSelect"
        :data="data"
        :header="headerConfig"
        :checkbox-enabled="true"
      >
        <template v-slot:customer="{ row: customer }">
          <router-link
            to="/apps/subscriptions/view-subscription"
            href=""
            class="text-gray-800 text-hover-primary mb-1"
          >
            {{ customer.customer }}
          </router-link>
        </template>
        <template v-slot:status="{ row: customer }">
          <a href="#" class="text-gray-600 text-hover-primary mb-1">
            <div :class="`badge badge-light-${customer.color}`">
              {{ customer.status }}
            </div>
          </a>
        </template>
        <template v-slot:billing="{ row: customer }">
          <div class="badge badge-light">{{ customer.billing }}</div>
        </template>
        <template v-slot:product="{ row: customer }">
          {{ customer.product }}
        </template>
        <template v-slot:createdDate="{ row: customer }">
          {{ customer.createdDate }}
        </template>
        <template v-slot:actions="{ row: customer }">
          <a
            href="#"
            class="btn btn-sm btn-light btn-active-light-primary"
            data-kt-menu-trigger="click"
            data-kt-menu-placement="bottom-end"
            data-kt-menu-flip="top-end"
            >Actions
            <span class="svg-icon svg-icon-5 m-0">
              <inline-svg
                :src="getAssetPath('/media/icons/duotune/arrows/arr072.svg')"
              />
            </span>
          </a>
          <!--begin::Menu-->
          <div
            class="menu menu-sub menu-sub-dropdown menu-column menu-rounded menu-gray-600 menu-state-bg-light-primary fw-semobold fs-7 w-125px py-4"
            data-kt-menu="true"
          >
            <!--begin::Menu item-->
            <div class="menu-item px-3">
              <router-link
                to="/apps/customers/customer-details"
                class="menu-link px-3"
                >View</router-link
              >
            </div>
            <!--end::Menu item-->
            <!--begin::Menu item-->
            <div class="menu-item px-3">
              <a @click="deleteSubscription(customer.id)" class="menu-link px-3"
                >Delete</a
              >
            </div>
            <!--end::Menu item-->
          </div>
          <!--end::Menu-->
        </template>
      </KTDatatable>
    </div>
    <!--end::Card body-->
  </div>
  <!--end::Card-->
</template>

<script lang="ts">
import { getAssetPath } from "@/core/helpers/assets";
import { defineComponent, ref } from "vue";
import KTDatatable from "@/components/kt-datatable/KTDataTable.vue";
import type { Sort } from "@/components/kt-datatable/table-partials/models";
import arraySort from "array-sort";

export default defineComponent({
  name: "kt-subscription-list",
  components: {
    KTDatatable,
  },
  setup() {
    const data = ref([
      {
        id: 1,
        customer: "Emma Smith",
        status: "Active",
        color: "success",
        billing: "Auto-debit",
        product: "Basic",
        createdDate: "Oct 25, 2021",
      },
      {
        id: 2,
        customer: "Melody Macy",
        status: "Active",
        color: "success",
        billing: "Manual - Credit Card",
        product: "Basic",
        createdDate: "Mar 10, 2021",
      },
      {
        id: 3,
        customer: "Max Smith",
        status: "Active",
        color: "primary",
        billing: "Manual - Cash",
        product: "Teams Bundle",
        createdDate: "Jul 25, 2021",
      },
      {
        id: 4,
        customer: "Sean Bean",
        status: "Expiring",
        color: "warning",
        billing: "Manual - Paypal",
        product: "Enterprise",
        createdDate: "Aug 19, 2021",
      },
      {
        id: 5,
        customer: "Brian Cox",
        status: "Expiring",
        color: "warning",
        billing: "Auto-debit",
        product: "Basic",
        createdDate: "May 05, 2021",
      },
      {
        id: 6,
        customer: "Mikaela Collins",
        status: "Active",
        color: "success",
        billing: "Auto-debit",
        product: "Enterprise Bundle",
        createdDate: "Aug 19, 2021",
      },
      {
        id: 7,
        customer: "Francis Mitcham",
        status: "Active",
        color: "success",
        billing: "Auto-debit",
        product: "Teams",
        createdDate: "Jun 20, 2021",
      },
      {
        id: 8,
        customer: "Olivia Wild",
        status: "Suspended",
        color: "danger",
        billing: "--",
        product: "Enterprise",
        createdDate: "Jun 24, 2021",
      },
      {
        id: 9,
        customer: "Neil Owen",
        status: "Expiring",
        color: "warning",
        billing: "Auto-debit",
        product: "Basic",
        createdDate: "Aug 19, 2021",
      },
      {
        id: 10,
        customer: "Dan Wilson",
        status: "Active",
        color: "success",
        billing: "Auto-debit",
        product: "Enterprise Bundle",
        createdDate: "Feb 21, 2021",
      },
      {
        id: 11,
        customer: "Emma Bold",
        status: "Active",
        color: "success",
        billing: "Manual - Credit Card",
        product: "Enterprise",
        createdDate: "May 05, 2021",
      },
      {
        id: 12,
        customer: "Ana Crown",
        status: "Active",
        color: "success",
        billing: "Manual - Credit Card",
        product: "Basic",
        createdDate: "Jun 24, 2021",
      },
      {
        id: 13,
        customer: "Robert Doe",
        status: "Suspended",
        color: "danger",
        billing: "--",
        product: "Teams Bundle",
        createdDate: "Jul 25, 2021",
      },
      {
        id: 14,
        customer: "John Miller",
        status: "Active",
        color: "success",
        billing: "Manual - Paypal",
        product: "Enterprise",
        createdDate: "Sep 22, 2021",
      },
      {
        id: 15,
        customer: "Lucy Kunic",
        status: "Active",
        color: "success",
        billing: "Manual - Credit Card",
        product: "Basic",
        createdDate: "Nov 10, 2021",
      },
      {
        id: 16,
        customer: "Neil Owen",
        status: "Suspended",
        color: "danger",
        billing: "--",
        product: "Basic Bundle",
        createdDate: "Jun 20, 2021",
      },
      {
        id: 17,
        customer: "Dan Wilson",
        status: "Expiring",
        color: "warning",
        billing: "Manual - Paypal",
        product: "Enterprise",
        createdDate: "May 05, 2021",
      },
      {
        id: 18,
        customer: "Emma Smith",
        status: "Active",
        color: "success",
        billing: "Auto-debit",
        product: "Teams",
        createdDate: "Apr 15, 2021",
      },
      {
        id: 19,
        customer: "Melody Macy",
        status: "Active",
        color: "success",
        billing: "Manual - Credit Card",
        product: "Basic",
        createdDate: "Oct 25, 2021",
      },
      {
        id: 20,
        customer: "Max Smith",
        status: "Suspended",
        color: "danger",
        billing: "--",
        product: "Basic Bundle",
        createdDate: "Feb 21, 2021",
      },
    ]);
    const headerConfig = ref([
      {
        columnName: "Customer",
        columnLabel: "customer",
        sortEnabled: true,
      },
      {
        columnName: "Status",
        columnLabel: "status",
        sortEnabled: true,
      },
      {
        columnName: "Billing",
        columnLabel: "billing",
        sortEnabled: true,
      },
      {
        columnName: "Product",
        columnLabel: "product",
        sortEnabled: true,
      },
      {
        columnName: "Created Date",
        columnLabel: "createdDate",
        sortEnabled: true,
      },
      {
        columnName: "Actions",
        columnLabel: "actions",
      },
    ]);

    const selectedIds = ref<Array<number>>([]);
    const deleteFewSubscriptions = () => {
      selectedIds.value.forEach((item) => {
        deleteSubscription(item);
      });
      selectedIds.value.length = 0;
    };
    const deleteSubscription = (id: number) => {
      for (let i = 0; i < data.value.length; i++) {
        if (data.value[i].id === id) {
          data.value.splice(i, 1);
        }
      }
    };
    const sort = (sort: Sort) => {
      const reverse: boolean = sort.order === "asc";
      if (sort.label) {
        arraySort(data.value, sort.label, { reverse });
      }
    };
    const onItemSelect = (selectedItems: Array<number>) => {
      selectedIds.value = selectedItems;
    };

    return {
      data,
      headerConfig,
      sort,
      onItemSelect,
      selectedIds,
      deleteFewSubscriptions,
      deleteSubscription,
      getAssetPath,
    };
  },
});
</script>
