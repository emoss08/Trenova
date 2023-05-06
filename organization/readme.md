# Cache Manager

## Introduction

Cache Manager is an essential tool for managing cached data in Monta. It integrates seamlessly with django-cacheops, a
popular Django library that automatically caches database queries. With Cache Manager, you can easily retrieve cached
data for faster processing and delete cached data to free up memory space.

In addition to its basic functionalities, Cache Manager also utilizes Django signals to automatically delete cached data
when a model is updated or created. This ensures that the data you retrieve is always up-to-date and reflects the latest
changes in your database. Whether you are handling large datasets or complex operations, Cache Manager simplifies the
management of cached data and helps to optimize the performance of your Monta application.

### Cached Data

- Control Files
    - Order Control
    - Invoice Control
    - Billing Control
    - Route Control
    - Dispatch Control
