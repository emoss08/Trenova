use std::time::Duration;
use kafka::producer::{Producer, Record, RequiredAcks};
use serde::{Serialize};
use serde_json;

#[derive(Serialize)]
struct BillingMessage {
    id: u32,
    amount: f64,
}

pub(crate) fn produce() {
    let mut producer: Producer = Producer::from_hosts(vec!["localhost:9092".to_owned()])
        .with_ack_timeout(Duration::from_secs(1))
        .with_required_acks(RequiredAcks::One)
        .create()
        .unwrap();

    for i in 0..100 {
        let message = BillingMessage { id: i, amount: 10.0 * (i as f64) };
        let message_json = serde_json::to_string(&message).unwrap();
        let record: Record<(), String> = Record::from_value("billing", message_json);
        producer.send(&record).unwrap();
    }
}
