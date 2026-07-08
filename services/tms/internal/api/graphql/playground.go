package graphql

const playgroundHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="robots" content="noindex,nofollow" />
    <title>Trenova GraphQL</title>
    <link
      rel="stylesheet"
      href="https://cdn.jsdelivr.net/npm/graphiql@4.1.2/graphiql.min.css"
      integrity="sha256-MEh+B2NdMSpj9kexQNN3QKc8UzMrCXW/Sx/phcpuyIU="
      crossorigin="anonymous"
    />
    <style>
      html,
      body,
      #graphiql {
        height: 100%;
        margin: 0;
      }
    </style>
  </head>
  <body>
    <div id="graphiql"></div>
    <script
      src="https://cdn.jsdelivr.net/npm/react@18.2.0/umd/react.production.min.js"
      integrity="sha256-S0lp+k7zWUMk2ixteM6HZvu8L9Eh//OVrt+ZfbCpmgY="
      crossorigin="anonymous"
    ></script>
    <script
      src="https://cdn.jsdelivr.net/npm/react-dom@18.2.0/umd/react-dom.production.min.js"
      integrity="sha256-IXWO0ITNDjfnNXIu5POVfqlgYoop36bDzhodR6LW5Pc="
      crossorigin="anonymous"
    ></script>
    <script
      src="https://cdn.jsdelivr.net/npm/graphiql@4.1.2/graphiql.min.js"
      integrity="sha256-hnImuor1znlJkD/FOTL3jayfS/xsyNoP04abi8bFJWs="
      crossorigin="anonymous"
    ></script>
    <script>
      const graphqlEndpoint = "/graphql";
      const csrfEndpoint = "/api/v1/auth/csrf";

      let csrfToken = null;
      let csrfHeaderName = "X-CSRF-Token";
      let csrfTokenRequest = null;

      function graphQLError(message, extensions) {
        return { errors: [{ message, extensions: extensions || {} }] };
      }

      async function fetchCsrfToken() {
        if (csrfToken) {
          return csrfToken;
        }

        csrfTokenRequest =
          csrfTokenRequest ||
          fetch(csrfEndpoint, {
            credentials: "include",
            headers: { Accept: "application/json" },
          })
            .then(async (response) => {
              if (response.status === 401) {
                throw new Error("Authentication required. Log in to Trenova, then reload /graphql.");
              }
              if (!response.ok) {
                throw new Error("Unable to bootstrap CSRF token for GraphQL.");
              }

              const data = await response.json();
              if (typeof data.headerName === "string" && data.headerName.trim()) {
                csrfHeaderName = data.headerName;
              }
              if (typeof data.csrfToken !== "string" || !data.csrfToken.trim()) {
                throw new Error("CSRF bootstrap response did not include a token.");
              }

              csrfToken = data.csrfToken;
              return csrfToken;
            })
            .finally(() => {
              csrfTokenRequest = null;
            });

        return csrfTokenRequest;
      }

      async function graphqlHeaders() {
        const headers = {
          Accept: "application/json",
          "Content-Type": "application/json",
        };
        headers[csrfHeaderName] = await fetchCsrfToken();
        return headers;
      }

      async function parseGraphQLResponse(response) {
        const text = await response.text();
        const payload = text ? JSON.parse(text) : null;
        if (response.ok) {
          return payload;
        }

        const message =
          (payload && (payload.detail || payload.title || payload.message)) || "GraphQL request failed.";
        return graphQLError(message, {
          status: response.status,
          type: payload && payload.type,
          traceId: payload && payload.traceId,
        });
      }

      async function sendGraphQLRequest(params, signal) {
        const headers = await graphqlHeaders();
        return fetch(graphqlEndpoint, {
          method: "POST",
          credentials: "include",
          headers,
          body: JSON.stringify(params),
          signal,
        });
      }

      async function graphQLFetcher(params, options) {
        try {
          let response = await sendGraphQLRequest(params, options && options.signal);
          if (response.status === 403) {
            csrfToken = null;
            response = await sendGraphQLRequest(params, options && options.signal);
          }
          return parseGraphQLResponse(response);
        } catch (error) {
          return graphQLError(error instanceof Error ? error.message : "GraphQL request failed.");
        }
      }

      ReactDOM.render(
        React.createElement(GraphiQL, {
          fetcher: graphQLFetcher,
          isHeadersEditorEnabled: false,
          shouldPersistHeaders: false,
          defaultEditorToolsVisibility: true,
        }),
        document.getElementById("graphiql")
      );
    </script>
  </body>
</html>
`
