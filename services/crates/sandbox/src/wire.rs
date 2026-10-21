//! The bodies of the sandbox's own frames (`Init`, `Ready`). Job bodies are
//! the service's.
//!
//! `Init` is a length-prefixed JSON policy followed by the service's
//! parameters as raw bytes: `u32 LE policy length ‖ policy JSON ‖ service`.

use crate::harden::HardeningPolicy;

/// What the supervisor tells a child at start.
#[derive(Debug)]
pub struct Init {
    pub policy: HardeningPolicy,
    pub service: Vec<u8>,
}

pub fn encode_init(policy: &HardeningPolicy, service: &[u8]) -> Vec<u8> {
    let json = serde_json::to_vec(policy).unwrap_or_default();
    let length = u32::try_from(json.len()).unwrap_or(u32::MAX);
    let mut out = Vec::with_capacity(4 + json.len() + service.len());
    out.extend_from_slice(&length.to_le_bytes());
    out.extend_from_slice(&json);
    out.extend_from_slice(service);
    out
}

pub fn decode_init(body: &[u8]) -> Result<Init, String> {
    let (length, rest) = body
        .split_first_chunk::<4>()
        .ok_or_else(|| "the body is shorter than its length prefix".to_owned())?;
    let length = u32::from_le_bytes(*length) as usize;
    if length > rest.len() {
        return Err("the policy runs past the body".into());
    }
    let (json, service) = rest.split_at(length);
    let policy = serde_json::from_slice(json).map_err(|err| err.to_string())?;
    Ok(Init {
        policy,
        service: service.to_vec(),
    })
}

#[cfg(test)]
mod tests {
    use proptest::prelude::*;

    use super::*;

    #[test]
    fn init_round_trips() {
        let policy = HardeningPolicy {
            read_only: vec!["/usr/lib".into()],
            read_write: Vec::new(),
            landlock: true,
            seccomp: true,
        };
        let decoded = decode_init(&encode_init(&policy, b"service")).expect("decodes");
        assert_eq!(decoded.policy, policy);
        assert_eq!(decoded.service, b"service");
    }

    proptest! {
        #[test]
        fn any_bytes_decode_or_are_refused_without_panicking(body in proptest::collection::vec(any::<u8>(), 0..256)) {
            let _ = decode_init(&body);
        }
    }
}
