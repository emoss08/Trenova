//! `trenova-imaging certs`: the private CA, server certificate and client
//! certificate a self-hosted install's TMS and imaging worker authenticate
//! each other with.
//!
//! It writes two directories, so each half can be mounted only where it is
//! used: the server's (`ca.pem`, `server.pem`, `server-key.pem`) into the
//! worker, the client's (`ca.pem`, `client.pem`, `client-key.pem`) into the
//! TMS processes. The CA's private key is written to neither: it is kept
//! only so the set can be reissued, in a third directory that nothing
//! mounts. Existing files are left alone unless `--force` is given, so the
//! command can run on every start.

use std::fs;
use std::io::Write;
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::process::ExitCode;
use std::time::{SystemTime, UNIX_EPOCH};

use rcgen::{
    BasicConstraints, CertificateParams, DistinguishedName, DnType, ExtendedKeyUsagePurpose, IsCa,
    Issuer, KeyPair, KeyUsagePurpose, date_time_ymd,
};

/// What `certs` writes.
#[derive(Debug)]
pub struct CertsRequest {
    pub server_dir: PathBuf,
    pub client_dir: PathBuf,
    pub ca_dir: PathBuf,
    /// DNS names the worker is reached by.
    pub server_names: Vec<String>,
    pub years: i32,
    pub force: bool,
}

pub fn run(request: &CertsRequest) -> ExitCode {
    match issue(request) {
        Ok(true) => {
            let _ = writeln!(
                std::io::stdout(),
                "issued the imaging CA, server and client certificates"
            );
            ExitCode::SUCCESS
        }
        Ok(false) => {
            let _ = writeln!(
                std::io::stdout(),
                "the imaging certificates already exist; nothing to do"
            );
            ExitCode::SUCCESS
        }
        Err(err) => {
            let _ = writeln!(std::io::stderr(), "certs: {err}");
            ExitCode::FAILURE
        }
    }
}

/// Issues the set. Returns whether anything was written.
pub fn issue(request: &CertsRequest) -> Result<bool, Box<dyn std::error::Error>> {
    let complete = [
        request.server_dir.join("server.pem"),
        request.server_dir.join("server-key.pem"),
        request.client_dir.join("client.pem"),
        request.client_dir.join("client-key.pem"),
    ]
    .iter()
    .all(|path| path.is_file());
    if complete && !request.force {
        return Ok(false);
    }
    if request.server_names.is_empty() {
        return Err("at least one --server-name is needed".into());
    }
    let years = request.years.clamp(1, 10);
    let this_year =
        1970 + i32::try_from(SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs() / 31_556_952)?;

    let validity = |params: &mut CertificateParams| {
        params.not_before = date_time_ymd(this_year - 1, 1, 1);
        params.not_after = date_time_ymd(this_year + years, 12, 31);
    };

    let ca_key = KeyPair::generate()?;
    let mut ca_params = CertificateParams::new(Vec::<String>::new())?;
    ca_params.distinguished_name = name("Trenova imaging CA");
    ca_params.is_ca = IsCa::Ca(BasicConstraints::Constrained(0));
    ca_params.key_usages = vec![KeyUsagePurpose::KeyCertSign, KeyUsagePurpose::CrlSign];
    validity(&mut ca_params);
    let ca_cert = ca_params.self_signed(&ca_key)?;
    let ca = Issuer::new(ca_params, &ca_key);

    let server_key = KeyPair::generate()?;
    let mut server_params = CertificateParams::new(request.server_names.clone())?;
    server_params.distinguished_name = name(&request.server_names[0]);
    server_params.key_usages = vec![KeyUsagePurpose::DigitalSignature];
    server_params.extended_key_usages = vec![ExtendedKeyUsagePurpose::ServerAuth];
    validity(&mut server_params);
    let server_cert = server_params.signed_by(&server_key, &ca)?;

    let client_key = KeyPair::generate()?;
    let mut client_params = CertificateParams::new(Vec::<String>::new())?;
    client_params.distinguished_name = name("trenova-tms");
    client_params.key_usages = vec![KeyUsagePurpose::DigitalSignature];
    client_params.extended_key_usages = vec![ExtendedKeyUsagePurpose::ClientAuth];
    validity(&mut client_params);
    let client_cert = client_params.signed_by(&client_key, &ca)?;

    for dir in [&request.server_dir, &request.client_dir, &request.ca_dir] {
        fs::create_dir_all(dir)?;
    }
    let ca_pem = ca_cert.pem();
    write(&request.ca_dir.join("ca.pem"), &ca_pem, 0o644)?;
    write(
        &request.ca_dir.join("ca-key.pem"),
        &ca_key.serialize_pem(),
        0o600,
    )?;
    write(&request.server_dir.join("ca.pem"), &ca_pem, 0o644)?;
    write(
        &request.server_dir.join("server.pem"),
        &server_cert.pem(),
        0o644,
    )?;
    write(
        &request.server_dir.join("server-key.pem"),
        &server_key.serialize_pem(),
        0o644,
    )?;
    write(&request.client_dir.join("ca.pem"), &ca_pem, 0o644)?;
    write(
        &request.client_dir.join("client.pem"),
        &client_cert.pem(),
        0o644,
    )?;
    write(
        &request.client_dir.join("client-key.pem"),
        &client_key.serialize_pem(),
        0o644,
    )?;
    Ok(true)
}

fn name(common_name: &str) -> DistinguishedName {
    let mut name = DistinguishedName::new();
    name.push(DnType::OrganizationName, "Trenova");
    name.push(DnType::CommonName, common_name);
    name
}

/// Writes a file whole, replacing any old one. Keys are readable by every
/// user inside the one container their directory is mounted into, because
/// the TMS and worker images run as different users; the directories, not
/// the modes, are what keep each key to its service.
fn write(path: &Path, contents: &str, mode: u32) -> std::io::Result<()> {
    let temp = path.with_extension("tmp");
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .mode(mode)
        .open(&temp)?;
    file.write_all(contents.as_bytes())?;
    file.sync_all()?;
    fs::set_permissions(&temp, fs::Permissions::from_mode(mode))?;
    fs::rename(&temp, path)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_set_is_issued_once_and_kept() {
        let dir = tempfile::tempdir().expect("tempdir");
        let request = CertsRequest {
            server_dir: dir.path().join("server"),
            client_dir: dir.path().join("client"),
            ca_dir: dir.path().join("ca"),
            server_names: vec!["imaging".into()],
            years: 2,
            force: false,
        };
        assert!(issue(&request).expect("issues"));
        let server = fs::read_to_string(dir.path().join("server/server.pem")).expect("server cert");
        assert!(server.starts_with("-----BEGIN CERTIFICATE-----"));
        assert!(
            !dir.path().join("server/ca-key.pem").exists(),
            "the CA key stays out of the mounted directories"
        );
        assert!(
            !issue(&request).expect("checks"),
            "a complete set is left alone"
        );
    }
}
