//! Where the Trenova server is.

use url::Url;

use capture_protocol::api::API_PREFIX;

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum ServerUrlError {
    #[error("enter the address of your Trenova server")]
    Empty,
    #[error("that is not a web address")]
    Invalid,
    #[error("the address must start with https://")]
    NotHttps,
    #[error("the address must not contain a user name or password")]
    Credentials,
}

/// The server the agent talks to: an origin, plus the path prefix when the
/// API is served below one. Only HTTPS is accepted, except on the machine
/// itself, where a developer runs the server over plain HTTP.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Server {
    base: Url,
}

impl Server {
    /// Parses what a person or an installer typed. A bare host name is taken
    /// as HTTPS.
    pub fn parse(raw: &str) -> Result<Self, ServerUrlError> {
        let raw = raw.trim();
        if raw.is_empty() {
            return Err(ServerUrlError::Empty);
        }
        let with_scheme = if raw.contains("://") {
            raw.to_owned()
        } else {
            format!("https://{raw}")
        };
        let mut url = Url::parse(&with_scheme).map_err(|_| ServerUrlError::Invalid)?;
        if url.host_str().is_none_or(str::is_empty) {
            return Err(ServerUrlError::Invalid);
        }
        match url.scheme() {
            "https" => {}
            "http" if is_loopback(&url) => {}
            "http" => return Err(ServerUrlError::NotHttps),
            _ => return Err(ServerUrlError::Invalid),
        }
        if !url.username().is_empty() || url.password().is_some() {
            return Err(ServerUrlError::Credentials);
        }
        url.set_query(None);
        url.set_fragment(None);
        let path = url.path().trim_end_matches('/').to_owned();
        url.set_path(&format!("{path}/"));
        Ok(Self { base: url })
    }

    /// A capture API route, `path` relative to `/api/v1/capture/`.
    pub fn api(&self, path: &str) -> Url {
        let relative = format!(
            "{}{}",
            API_PREFIX.trim_start_matches('/'),
            path.trim_start_matches('/')
        );
        self.base
            .join(&relative)
            .unwrap_or_else(|_| self.base.clone())
    }

    /// The address as stored and shown.
    pub fn as_str(&self) -> &str {
        self.base.as_str().trim_end_matches('/')
    }

    /// The host, which names the stored credential.
    pub fn host(&self) -> &str {
        self.base.host_str().unwrap_or_default()
    }
}

fn is_loopback(url: &Url) -> bool {
    match url.host() {
        Some(url::Host::Domain(domain)) => domain.eq_ignore_ascii_case("localhost"),
        Some(url::Host::Ipv4(ip)) => ip.is_loopback(),
        Some(url::Host::Ipv6(ip)) => ip.is_loopback(),
        None => false,
    }
}

/// The web app's address, taken from where the server told the person to
/// approve the pairing: its origin and any prefix before `/capture/pair`.
pub fn web_base_from_verification(uri: &str) -> Option<String> {
    let url = Url::parse(uri).ok()?;
    if !matches!(url.scheme(), "https" | "http") {
        return None;
    }
    let path = url.path();
    let prefix = path
        .strip_suffix("/capture/pair")
        .or_else(|| path.strip_suffix("/capture/pair/"))?;
    let mut base = url.clone();
    base.set_path(prefix);
    base.set_query(None);
    base.set_fragment(None);
    Some(base.as_str().trim_end_matches('/').to_owned())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_bare_host_is_https_and_routes_join_below_the_api_prefix() {
        let server = Server::parse("tms.acme-freight.com").expect("server");
        assert_eq!(server.as_str(), "https://tms.acme-freight.com");
        assert_eq!(
            server.api("device/batches/").as_str(),
            "https://tms.acme-freight.com/api/v1/capture/device/batches/"
        );
    }

    #[test]
    fn a_path_prefix_is_kept() {
        let server = Server::parse("https://acme.example/tms/?x=1#y").expect("server");
        assert_eq!(
            server.api("pair/").as_str(),
            "https://acme.example/tms/api/v1/capture/pair/"
        );
    }

    #[test]
    fn plain_http_is_only_for_this_machine() {
        assert_eq!(
            Server::parse("http://tms.acme.com"),
            Err(ServerUrlError::NotHttps)
        );
        assert!(Server::parse("http://localhost:8080").is_ok());
        assert!(Server::parse("http://127.0.0.1:8080").is_ok());
        assert!(Server::parse("http://[::1]:8080").is_ok());
        assert_eq!(
            Server::parse("ftp://tms.acme.com"),
            Err(ServerUrlError::Invalid)
        );
        assert_eq!(
            Server::parse("https://u:p@tms.acme.com"),
            Err(ServerUrlError::Credentials)
        );
        assert_eq!(Server::parse("   "), Err(ServerUrlError::Empty));
    }

    #[test]
    fn the_web_base_comes_from_the_approval_address() {
        assert_eq!(
            web_base_from_verification("https://app.acme.com/capture/pair").as_deref(),
            Some("https://app.acme.com")
        );
        assert_eq!(
            web_base_from_verification("https://acme.com/tms/capture/pair?code=BCDF-GHJK")
                .as_deref(),
            Some("https://acme.com/tms")
        );
        assert_eq!(
            web_base_from_verification("https://acme.com/elsewhere"),
            None
        );
        assert_eq!(web_base_from_verification("javascript:alert(1)"), None);
    }
}
