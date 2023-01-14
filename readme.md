<h3 align="center">Monta Suite</h3>

  <p align="center">
    Suite of transportation & logistics application. Built to make your business better!
    <br />
    <a href="#"><strong>Explore the docs »</strong></a>

## Introduction

---
Monta provides a fully capable backend software that supports modern transportation and logistics business.
It is built on top of the Django framework and is designed to be easily extendable.

## Prerequisites

- Python 3.10+ or latest
- PostgreSQL 12+ (no plans to support other databases)
- Redis 6.0+
- Kafka 3.3.1+
- Zookeeper 3.7+

## What is Monta?

---
Monta is a Asset Transportation Management System primarily for Over-the-road trucking companies.

## What makes Monta Different?

---
Monta leverages Machine Learning to make predictions and leverages the forecasted data to make decisions in
the application without needing your input. Additionally, Monta has over 20 scheduled tasks that run
in the background of the application to assist your operation, these tasks are designed to reduce repetitiveness and 
to reduce manual entry. Finally, we know time is money and adding head count to manage a system is a real
world problem, our goal is to give you a system that manages your operation for you without needing to hire additional
help to manage your system!

### Order Prediction Application Example

---
The most common usage for Machine Learning is simply assigning the order to proper worker. Sounds easy enough right?
There are quite a few dependencies that need to be accounted for that our Machine Learning models are trained on.
First, does the driver have enough time to get the order to its destination based on the pickup and delivery date?
Second, does this driver have a pattern of missing deliveries or pickups? Finally, does this driver meet the KPI set
by you or your operations' leadership(On-Time Performance, Miles Per Gallon, Weekly Miles, Etc...). We take all of that
in to account to ensure your customers are receiving the best service possible. Once the decision is made, it takes care
of the rest! The order will be assigned to the driver automatically and if a telematics, email,
or text message is desired it can even automatically send the information over to the driver!

Then there is pre-assignments....
