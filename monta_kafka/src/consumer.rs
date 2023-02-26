use kafka::consumer::{Consumer, FetchOffset, GroupOffsetStorage};
use serde::{Deserialize};

#[derive(Debug, Deserialize)]
struct BillingMessage {
    id: u32,
    amount: f64,
}

pub(crate) fn consume() {
    let mut consumer: Consumer = Consumer::from_hosts(vec!["localhost:9092".to_owned()])
        .with_topic("billing".parse().unwrap())
        .with_fallback_offset(FetchOffset::Earliest)
        .with_group("billing_group".parse().unwrap())
        .with_offset_storage(GroupOffsetStorage::Kafka)
        .create()
        .unwrap();

    for ms in consumer.poll().unwrap().iter() {
        for m in ms.messages() {
            let message_str = std::str::from_utf8(m.value).unwrap();
            let message: BillingMessage = serde_json::from_str(message_str).unwrap();
            println!("{:?}", message);
        }
        consumer.consume_messageset(ms).unwrap();
    }
    consumer.commit_consumed().unwrap();
}