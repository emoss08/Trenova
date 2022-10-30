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

- Python 3.11+
    - No lower version supported.
- Django 4.1.2 (Recommended)
    - Does support Django 4.0.0+
- PostgreSQL 12+ (no plans to support other databases)
- Redis 6.0+

### Built with

[Django](https://www.djangoproject.com/start/overview/) - Django is a high-level Python web framework that encourages
rapid development and clean, pragmatic design.

### Style Guides

We use the follow style guides to ensure consistency across the codebase.

- [Django Coding Style](https://docs.djangoproject.com/en/4.0/internals/contributing/writing-code/coding-style/) -
  Primary style guide for the backend.
- [Google Python Style Guide](https://google.github.io/styleguide/pyguide.html)  - Primary style guide for the
  docstrings.
- [Hacksoft Django Style Guide](https://github.com/HackSoftware/Django-Styleguide) - Secondary style guide for the
  backend.

Note - We only use certain parts of each style guide. Of course, we try our best to follow the style guides as much as
possible.
However, we do not follow the style guides to the letter.

### Formatting

- [Black](https://black.readthedocs.io/en/stable/) - Black is the uncompromising Python code formatter.
- [isort](https://pycqa.github.io/isort/) - A Python utility / library to sort imports alphabetically, and automatically
  separated into sections and by type.

### Linting

- [Mypy](http://mypy-lang.org/) - Optional static typing for Python.
- [flake8](https://flake8.pycqa.org/en/latest/) - The tool for style guide enforcement.
- [pylint](https://www.pylint.org/) - A Python code static checker.

### Testing

- [pytest](https://docs.pytest.org/en/stable/) - The pytest framework makes it easy to write small tests, yet scales to
  support complex functional testing for applications and libraries.
- [pytest-django](https://pytest-django.readthedocs.io/en/latest/) - Pytest plugin for Django projects.
- [pytest-cov](https://pytest-cov.readthedocs.io/en/latest/) - Pytest plugin for measuring coverage.
