mod consumer;
mod producer;

fn main() {
    producer::produce();
    consumer::consume();
}
