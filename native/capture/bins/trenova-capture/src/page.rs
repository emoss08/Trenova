//! The window's page: one self-contained HTML document, its stylesheet,
//! script and logo inlined, so nothing is ever loaded from anywhere else.

use base64::Engine as _;
use base64::engine::general_purpose::STANDARD;

use crate::icon::{LOGO, best_for};

const TEMPLATE: &str = include_str!("../ui/page.html");
const STYLE: &str = include_str!("../ui/style.css");
const SCRIPT: &str = include_str!("../ui/app.js");

/// The Trenova mark as a data URI, from the largest PNG in the icon file.
fn logo() -> String {
    best_for(LOGO, 256)
        .filter(|image| image.bytes.starts_with(b"\x89PNG"))
        .map(|image| format!("data:image/png;base64,{}", STANDARD.encode(image.bytes)))
        .unwrap_or_default()
}

/// The page, ready to load.
pub fn html() -> String {
    TEMPLATE
        .replacen("/*STYLE*/", STYLE, 1)
        .replacen("/*SCRIPT*/", SCRIPT, 1)
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
        let addresses_removed = ADDRESSES
            .iter()
            .fold(page.clone(), |page, address| page.replace(address, ""));
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

    /// The only addresses in the page: the ones the server field suggests,
    /// and the SVG namespace the icons are created in, which is a name, not
    /// something fetched.
    const ADDRESSES: [&str; 3] = [
        "\"https://cloud.trenova.app\"",
        "\"http://localhost:5173\"",
        "\"http://www.w3.org/2000/svg\"",
    ];

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
