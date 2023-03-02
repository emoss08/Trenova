import React, { useMemo } from "react";
import { useSystemHealthStore } from "../stores/SystemHealthStore";
import SystemHealth from "./SystemHealth";
import SystemHealthLoader from "./SystemHealthLoader";

interface CacheBackend {
  name: string;
  status: string;
}

function SystemHealthPage() {
  const { serviceData } = useSystemHealthStore();

  const memoizedData = useMemo(() => serviceData, [serviceData]);

  return (
    <div>
      {Object.entries(memoizedData || {}).map(([key, value], index) => (
        <div key={index}>
          <h2>{key}</h2>
          <ul>
            {key === "cache_backend"
              ? value?.map((service: CacheBackend, index: number) => (
                  <li key={index}>
                    <SystemHealth
                      service={service.name}
                      status={service.status}
                    />
                  </li>
                ))
              : value?.status && (
                  <li>
                    <SystemHealth service={key} status={value.status} />
                  </li>
                )}
          </ul>
        </div>
      ))}
    </div>
  );
}

function SystemHealthPageWrapper() {
  const { loading, fetchData } = useSystemHealthStore();

  React.useEffect(() => {
    fetchData();

    const interval = setInterval(() => {
      fetchData();
    }, 60000);

    return () => clearInterval(interval);
  }, [fetchData]);

  return loading ? <SystemHealthLoader /> : <SystemHealthPage />;
}

export { SystemHealthPageWrapper };
