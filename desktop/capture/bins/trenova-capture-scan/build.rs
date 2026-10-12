//! Links the Visual C++ runtime into the executable, so it runs on a Windows
//! without the Visual C++ Redistributable. The Universal CRT stays dynamic:
//! it is part of Windows 10 and 11. Does nothing on other targets.

fn main() {
    println!("cargo:rerun-if-changed=build.rs");
    static_vcruntime::metabuild();
}
