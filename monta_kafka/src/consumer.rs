use kafka::consumer::{Consumer, FetchOffset, GroupOffsetStorage};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct BillingMessage {
    pub id: String,
    pub order_type: String,
    pub revenue_code: String,
    pub customer: String,
    pub invoice_number: String,
    pub pieces: u32,
    pub weight: u32,
    pub bill_type: String,
    pub ready_to_bill: bool,
    pub bill_date: String,
    pub mileage: f32,
    pub worker: String,
    pub commodity: String,
    pub commodity_descr: String,
    pub consignee_ref_number: String,
    pub other_charge_amount: String,
    pub freight_charge_amount: String,
    pub total_amount: String,
    pub is_summary: bool,
    pub is_cancelled: bool,
    pub bol_number: String,
    pub user: String,
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
            println!("Billing Message received {message:#?}");
        }
        consumer.consume_messageset(ms).unwrap();
    }
    consumer.commit_consumed().unwrap();
}