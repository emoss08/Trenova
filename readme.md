<h3 align="center">Monta Suite</h3>

  <p align="center">
    Suite of transportation & logistics application. Built to make your business better!
    <br />
    <a href="#"><strong>Explore the docs »</strong></a>

## Introduction

Monta offers a comprehensive backend software that caters to the needs of contemporary transportation and logistics
enterprises. Our software is built on top of the Django framework, making it highly flexible and customizable to suit
your unique requirements. Its extendable design ensures that it can adapt to the evolving needs of your business,
enabling you to stay ahead of the competition.

## Prerequisites

- Python 3.10+ or latest
- PostgreSQL 12+ (no plans to support other databases)
- Redis 6.0+
- Kafka 3.3.1+
- Zookeeper 3.7+

## What makes Monta Different?

Monta is a cutting-edge system that utilizes the power of Machine Learning
to predict and analyze data, enabling it to make informed decisions within
the application, without requiring any input from you. With over 20 pre-scheduled tasks
that run seamlessly in the background, Monta streamlines your operation by eliminating repetitive and time-consuming
manual entries. We understand that time is a valuable asset and that hiring additional staff to manage a system can be a
real-world challenge. Our objective is to provide you with a system that effortlessly manages your operation, without
the need for extra help.

### Order Prediction Application Example

Although assigning orders to the right worker may seem like a simple task, Machine Learning is crucial in achieving this
efficiently. Our Machine Learning models take into account a range of dependencies that are critical to the success of
your operations.

First, we assess whether the driver has sufficient time to deliver the order based on the pickup and delivery dates.
Next, we consider the driver's track record in terms of missed deliveries or pickups. Finally, we evaluate whether the
driver meets your specified KPIs, such as On-Time Performance, Miles Per Gallon, and Weekly Miles.

By accounting for all of these factors, we ensure that your customers receive top-quality service. Once the decision is
made, our system takes care of the rest. The order is automatically assigned to the most suitable driver, and if
necessary, our system can even send the relevant information to the driver via telematics, email, or text message.

Then there is pre-assignments....

**Disclaimer:**

It is essential to note that the utilization of Metronics from <b>Keenthemes</b> on Monta's frontend is not permitted without a valid license. This template is proprietary and requires proper authorization to be utilized. Any unauthorized usage may lead to legal consequences, and it is the sole responsibility of the end-user to obtain the necessary license before utilizing the Metronics template. <b>Monta LLC.</b> will not be held accountable for any violations or illegal usage of the Metronics template without proper authorization. To purchase a license for <b>Metronic</b> by <b>Keenthemes</b>, please visit <https://1.envato.market/EA4JP>. For further information on Keenthemes, please visit <https://keenthemes.com/>.

### Developer Features

Monta incorporates a number of developer-friendly features that make it easy to customize and extend the system to suit your
unique requirements. These include:

<details>
<summary>Traceback</summary>

Monta incorporates an abstract representation of [Rich Tracebacks](https://rich.readthedocs.io/en/latest/traceback.html), which enhances the
debugging experience of your code. This feature is enabled by default, but you have the option to disable it by setting the MONTA_TRACEBACK
environment variable to 0.

Check out the example below to see how it works:
![Example](https://github.com/Monta-Application/Monta/blob/main/imgs/traceback.png?raw=true)
</details>
