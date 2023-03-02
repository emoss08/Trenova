function createRequestFactory(
  url: string,
  setFunc: (data: any) => void,
  setLoading: (loading: boolean) => void
) {
  return () => {
    setLoading(true);
    fetch(url)
      .then((response) => response.json())
      .then((data) => {
        setFunc(data);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Error:", error);
        setLoading(false);
      });
  };
}

export default createRequestFactory;
