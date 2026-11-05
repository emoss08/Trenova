//! Who is on the other end of a loopback connection.

use std::io;
use std::net::{IpAddr, SocketAddr};

use capture_ipp::ClientAccess;
use trenova_capture_svc::listener::Peers;
use windows::Win32::Foundation::{ERROR_INSUFFICIENT_BUFFER, NO_ERROR};
use windows::Win32::NetworkManagement::IpHelper::{
    GetExtendedTcpTable, MIB_TCPROW_OWNER_PID, MIB_TCPTABLE_OWNER_PID,
    TCP_TABLE_OWNER_PID_CONNECTIONS,
};
use windows::Win32::Networking::WinSock::AF_INET;
use windows_service::service::ServiceAccess;
use windows_service::service_manager::{ServiceManager, ServiceManagerAccess};

/// The process holding the client end of a connection to `server_port`.
pub fn owning_process(client: SocketAddr, server_port: u16) -> io::Result<Option<u32>> {
    let IpAddr::V4(client_ip) = client.ip() else {
        return Ok(None);
    };
    let local_addr = u32::from_ne_bytes(client_ip.octets());
    let rows = connections()?;
    Ok(rows
        .iter()
        .find(|row| {
            row.dwLocalAddr == local_addr
                && port_of(row.dwLocalPort) == client.port()
                && port_of(row.dwRemotePort) == server_port
        })
        .map(|row| row.dwOwningPid))
}

fn port_of(raw: u32) -> u16 {
    u16::from_be(u16::try_from(raw & 0xffff).unwrap_or(0))
}

fn connections() -> io::Result<Vec<MIB_TCPROW_OWNER_PID>> {
    let mut size = 0u32;
    let mut buffer: Vec<u64> = Vec::new();
    loop {
        let table = if buffer.is_empty() {
            None
        } else {
            Some(buffer.as_mut_ptr().cast())
        };
        // SAFETY: `table` is null or an 8-aligned buffer of at least `size` bytes.
        let status = unsafe {
            GetExtendedTcpTable(
                table,
                &raw mut size,
                false,
                u32::from(AF_INET.0),
                TCP_TABLE_OWNER_PID_CONNECTIONS,
                0,
            )
        };
        if status == ERROR_INSUFFICIENT_BUFFER.0 {
            buffer.resize((size as usize).div_ceil(8) + 1, 0);
            continue;
        }
        if status != NO_ERROR.0 {
            return Err(io::Error::from_raw_os_error(
                i32::try_from(status).unwrap_or(i32::MAX),
            ));
        }
        break;
    }
    let header = buffer.as_ptr().cast::<MIB_TCPTABLE_OWNER_PID>();
    // SAFETY: the call above filled the buffer with a table of `dwNumEntries`
    // rows laid out from `table`.
    let rows = unsafe {
        std::slice::from_raw_parts(
            (&raw const (*header).table).cast::<MIB_TCPROW_OWNER_PID>(),
            (*header).dwNumEntries as usize,
        )
    };
    Ok(rows.to_vec())
}

/// The process the Windows print spooler runs in.
pub fn spooler_process() -> io::Result<u32> {
    let manager = ServiceManager::local_computer(None::<&str>, ServiceManagerAccess::CONNECT)
        .map_err(io::Error::other)?;
    let service = manager
        .open_service("Spooler", ServiceAccess::QUERY_STATUS)
        .map_err(io::Error::other)?;
    let status = service.query_status().map_err(io::Error::other)?;
    status
        .process_id
        .ok_or_else(|| io::Error::other("the print spooler is not running"))
}

/// Only the print spooler may send a job: it is the one client whose jobs the
/// queue the service attributes them from describes. Anything else on this
/// computer may still read what the printer is, which Windows' own device
/// discovery does when the printer is added.
#[derive(Clone, Copy, Debug)]
pub struct SpoolerPeers;

impl Peers for SpoolerPeers {
    fn access(&self, peer: SocketAddr, port: u16) -> ClientAccess {
        let owner = owning_process(peer, port);
        match (&owner, spooler_process()) {
            (Ok(Some(owner)), Ok(spooler)) if *owner == spooler => ClientAccess::Print,
            (_, spooler) => {
                tracing::info!(%peer, ?owner, ?spooler, "a client that is not the spooler");
                ClientAccess::Discovery
            }
        }
    }
}
