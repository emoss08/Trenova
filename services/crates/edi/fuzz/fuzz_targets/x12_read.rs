//! Reads arbitrary text as a whole file and as a fragment, against the base guides, and
//! acknowledges every group read: none of it may panic, and every finding must point inside
//! the text.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_edi::x12::read::{ReadRequest, read};

fuzz_target!(|data: &[u8]| {
    let Ok(text) = std::str::from_utf8(data) else {
        return;
    };
    for fragment in [false, true] {
        let request = ReadRequest {
            text: text.to_owned(),
            separators: None,
            guides: Vec::new(),
            include_segments: true,
            fragment,
        };
        let Ok(reading) = read(&request) else {
            return;
        };
        for segment in &reading.segments {
            assert!(segment.start <= segment.end && segment.end <= text.len());
        }
        for interchange in &reading.interchanges {
            for group in &interchange.groups {
                for transaction in &group.transactions {
                    for finding in &transaction.findings {
                        if let Some((start, end)) = finding.span {
                            assert!(start <= end && end <= text.len());
                        }
                    }
                }
                let json = serde_json::json!({
                    "op": "x12.acknowledge",
                    "input": {
                        "kind": "999",
                        "acknowledgments": [{
                            "group": group,
                            "envelope": {
                                "senderQualifier": "ZZ", "senderId": "TRENOVA",
                                "receiverQualifier": "ZZ", "receiverId": "PARTNER",
                                "applicationSender": "TRENOVA", "applicationReceiver": "PARTNER",
                                "date": "20260107", "time": "1200",
                                "interchangeControlNumber": "1", "groupControlNumber": "1",
                                "transactionControlNumber": "0001", "usage": "T"
                            }
                        }]
                    }
                });
                let response = trenova_edi::ops::dispatch(json.to_string().as_bytes());
                assert!(
                    response.starts_with(b"{\"ok\":"),
                    "{}",
                    String::from_utf8_lossy(&response)
                );
            }
        }
    }
});
