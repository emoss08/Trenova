//! Embeds the Trenova mark as the executable's icon, from the one copy the
//! web app ships (`client/apps/web/public/logo.ico`), so the two never drift.
//! The tray reads the same file at compile time (`src/tray/icon.rs`).

const LOGO: &str = "../../../../client/apps/web/public/logo.ico";

fn main() {
    println!("cargo:rerun-if-changed={LOGO}");
    println!("cargo:rerun-if-changed=build.rs");
    // The Visual C++ runtime goes into the executable, so it runs on a
    // Windows without the Visual C++ Redistributable; the Universal CRT is
    // part of Windows. Nothing happens on other targets.
    static_vcruntime::metabuild();
    #[cfg(windows)]
    embed();
}

/// A resource compiler is only available when building on Windows, which is
/// where releases are built; a check from another host skips the resource
/// and says so.
#[cfg(windows)]
fn embed() {
    let mut resource = winresource::WindowsResource::new();
    resource
        .set_icon(LOGO)
        .set("ProductName", "Trenova Capture")
        .set("FileDescription", "Trenova Capture")
        .set("CompanyName", "Trenova")
        .set("LegalCopyright", "Trenova");
    if let Err(err) = resource.compile() {
        println!("cargo:warning=the icon resource was not embedded: {err}");
    }
}
