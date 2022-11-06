# Monta Worker Application

----

## Table of Contents

- [Introduction](#introduction)
- [Files](#files)
- [Models](#models)
    - [Worker](#WorkerModel)
        - [Worker Type Enum](#WorkerType)

### Introduction <a name="introduction"></a>

- Monta Worker is an application that is used for storing Worker(e.q. Driver) information.

### Files <a name="files"></a>

- `admin.py` - Admin configuration for the application.
- `apps.py` - Application configuration.
- `models.py` - Database models for the application.
- `serializers.py` - Serializers for the application.
- `tests.py` - Tests for the application.
- `urls.py` - URL configuration for the application.
- `views.py` - Views for the application.

### Models.py <a name="models"></a>

- `Worker` - Worker model. <a name="WorkerModel"></a>
    - `WorkerType` - Enum of Worker Types. <a name="WorkerType"></a>
        - `EMPLOYEE` - Employee.
        - `CONTRACTOR` - Driver.
    - Fields of `Worker` model.
        - code - Primary key.
        - is_active - Boolean field for active status.
        - worker_type - Enum field for Worker type.
        - first_name - First name of the Worker.
        - last_name - Last name of the Worker.
        - address_line_1 - Address line 1 of the Worker.
        - address_line_2 - Address line 2 of the Worker.
        - city - City of the Worker.
        - state - State of the Worker.
        - zip_code - Zip code of the Worker.
        - depot - Foreign key to `Depot` model.
        - manager - Foreign key to `User` model for Manager.
    - Methods of `Worker` model.
        - `__str__` - Returns string representation of the Worker.
        - `get_full_name` - Returns full name of the Worker.
        - `get_full_address` - Returns full address of the Worker.

----

- `WorkerProfile` - Worker profile model.
    - `WorkerSexChoices` - Enum of Gender/Sex Choices
        - MALE
        - FEMALE
        - NON_BINARY
        - OTHER
    - `EndorsementChoices` - Enum of Endorsements
        - NONE
        - HAZMAT
        - TANKER
        - X
    - Fields of `WorkerProfile` model.
        - worker - Foreign key to `Worker` model.
        - race - Race/Ethnicity of the worker.
        - sex = Sex/Gender of the worker.
        - date_of_birth - Date of birth of the worker.
        - license_number - License number of the worker.
        - license_state - License state of the worker.
        - license_expiration_date - License expiration date of the worker.
        - endorsements - Endorsements of the worker.
        - hazmat_expiration_date - Hazmat expiration date of the worker.
        - hm_126_expiration_date - HM 126 expiration date of the worker.
        - hire_date - Hire date of the worker.
        - termination_date - Termination date of the worker.
        - review_date - Review date of the worker.
        - physical_due_date - Physical due date of the worker.
        - mvr_due_date - MVR due date of the worker.
        - medical_cert_date - Medical certification date of the worker.
    - Methods of `WorkerProfile` model.
        - `__str__` - Returns string representation of the WorkerProfile.
        - `clean` - Clean method for the WorkerProfile model.
