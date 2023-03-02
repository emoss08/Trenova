import React, { useState, useEffect, useMemo } from "react";
import SystemHealthLoader from "../../frontend_app/src/components/SystemHealthLoader";
import SystemHealth from "../../frontend_app/src/components/SystemHealth";

import "./App.css";

interface SystemHealthData {
  [key: string]: any;

  // Add other properties as needed
}

interface CacheBackend {
  name: string;
  status: string;
}

interface AppProps {
  data: SystemHealthData;
}

const App = React.memo((props: AppProps) => {
  const { data } = props;

  const memoizedData = useMemo(() => data, [data]);

  return (
    <div>
      {Object.entries(memoizedData).map(([key, value], index) => (
        <div key={index}>
          <h2>{key}</h2>
          <ul>
            {key === "cache_backend"
              ? value.map((service: CacheBackend, index: number) => (
                <li key={index}>
                  <SystemHealth
                    service={service.name}
                    status={service.status}
                  />
                </li>
              ))
              : value.status && (
              <li>
                <SystemHealth service={key} status={value.status} />
              </li>
            )}
          </ul>
        </div>
      ))}
    </div>
  );
});

function AppWrapper() {
  const [loading, setLoading] = useState(true);
  const [serviceData, setServiceData] = useState<SystemHealthData | null>(null);

  function fetchData() {
    fetch("http://127.0.0.1:8000/api/system_health/")
      .then((response) => response.json())
      .then((data) => {
        setServiceData(data);
        setLoading(false);
      })
      .catch((error) => {
        // Handle error here
        console.error("Error:", error);
      });
  }

  useEffect(() => {
    fetchData();

    const interval = setInterval(() => {
      fetchData();
    }, 60000);

    return () => clearInterval(interval);
  }, []);

  return loading ? (
    <SystemHealthLoader />
  ) : serviceData ? (
    <App data={serviceData} />
  ) : null;
}

export default AppWrapper;
