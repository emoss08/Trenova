//! The window's page: one self-contained HTML document, its stylesheet,
//! script and logo inlined, so nothing is ever loaded from anywhere else.

use base64::Engine as _;
use base64::engine::general_purpose::STANDARD;
use capture_client::Server;

use crate::icon::{LOGO, best_for};

const TEMPLATE: &str = include_str!("../ui/page.html");
const STYLE: &str = include_str!("../ui/style.css");
const SCRIPT: &str = include_str!("../ui/app.js");

/// Where the script expects the suggested server address, inside a string.
const SUGGESTED_SERVER_SLOT: &str = "/*SUGGESTED_SERVER*/";

/// The server address a build suggests in the server field. Public builds
/// set none, so a person always enters the address of their own Trenova; a
/// distributor that ships for one Trenova sets
/// `TRENOVA_CAPTURE_SUGGESTED_SERVER` when building.
const BUILD_SUGGESTED_SERVER: Option<&str> = option_env!("TRENOVA_CAPTURE_SUGGESTED_SERVER");

/// The Trenova mark as a data URI, from the largest PNG in the icon file.
fn logo() -> String {
    best_for(LOGO, 256)
        .filter(|image| image.bytes.starts_with(b"\x89PNG"))
        .map(|image| format!("data:image/png;base64,{}", STANDARD.encode(image.bytes)))
        .unwrap_or_default()
}

/// The suggested address as it may appear inside the script: one the agent
/// would accept, made only of characters that cannot end the string or the
/// script it sits in. Anything else suggests nothing.
fn suggested_server(raw: Option<&str>) -> String {
    raw.and_then(|raw| Server::parse(raw).ok())
        .map(|server| server.as_str().to_owned())
        .filter(|address| {
            address
                .chars()
                .all(|c| c.is_ascii_alphanumeric() || ":/.-_~%[]@".contains(c))
        })
        .unwrap_or_default()
}

/// The page, ready to load.
pub fn html() -> String {
    let script = SCRIPT.replacen(
        SUGGESTED_SERVER_SLOT,
        &suggested_server(BUILD_SUGGESTED_SERVER),
        1,
    );
    TEMPLATE
        .replacen("/*STYLE*/", STYLE, 1)
        .replacen("/*SCRIPT*/", &script, 1)
        .replacen("/*LOGO*/", &logo(), 1)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_page_is_self_contained_and_locked_down() {
        let page = html();
        assert!(!page.contains("/*STYLE*/"));
        assert!(!page.contains("/*SCRIPT*/"));
        assert!(!page.contains("/*LOGO*/"));
        assert!(!page.contains(SUGGESTED_SERVER_SLOT));
        assert!(page.contains(
            "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:"
        ));
        assert!(page.contains(r#"src="data:image/png;base64,"#));
        assert_eq!(
            page.matches("</script>").count(),
            1,
            "the script cannot end early"
        );
        assert_eq!(
            page.matches("</style>").count(),
            1,
            "the stylesheet cannot end early"
        );
        let suggested = format!("\"{}\"", suggested_server(BUILD_SUGGESTED_SERVER));
        let addresses_removed = ADDRESSES
            .iter()
            .fold(page.replacen(&suggested, "\"\"", 1), |page, address| {
                page.replace(address, "")
            });
        for outside in ["http://", "https://", "src=\"//", "@import", "url("] {
            assert!(
                !addresses_removed.contains(outside),
                "the page reaches outside itself: {outside}"
            );
        }
        for address in ADDRESSES {
            assert_eq!(page.matches(address).count(), 1, "{address} appears once");
        }
    }

    /// The only addresses in the page besides the one a build may suggest:
    /// the server field's example, the development server it offers, and the
    /// SVG namespace the icons are created in, which is a name, not something
    /// fetched.
    const ADDRESSES: [&str; 3] = [
        "\"https://trenova.example.com\"",
        "\"http://localhost:5173\"",
        "\"http://www.w3.org/2000/svg\"",
    ];

    #[test]
    fn a_public_build_suggests_no_server() {
        assert_eq!(suggested_server(None), "");
        assert_eq!(suggested_server(Some("")), "");
    }

    #[test]
    fn a_suggested_server_is_one_the_agent_accepts() {
        assert_eq!(
            suggested_server(Some("tms.acme-freight.com")),
            "https://tms.acme-freight.com"
        );
        assert_eq!(
            suggested_server(Some("http://localhost:5173/")),
            "http://localhost:5173"
        );
        assert_eq!(suggested_server(Some("http://tms.acme-freight.com")), "");
        assert_eq!(suggested_server(Some("ftp://tms.acme-freight.com")), "");
    }

    #[test]
    fn a_suggested_server_cannot_end_the_script() {
        for raw in [
            "https://tms.acme.com/\"};alert(1);//",
            "https://tms.acme.com/</script><script>alert(1)</script>",
            "https://tms.acme.com/\\\"",
        ] {
            let address = suggested_server(Some(raw));
            assert!(
                !address.contains(['"', '<', '>', '\\', '\'']),
                "{raw} became {address}"
            );
        }
    }

    #[test]
    fn the_script_draws_text_never_markup() {
        for markup in [
            "innerHTML",
            "outerHTML",
            "insertAdjacentHTML",
            "document.write",
            "eval(",
        ] {
            assert!(!SCRIPT.contains(markup), "the script uses {markup}");
        }
    }
}
