FROM python:3.11-slim-buster

LABEL maintainer="montadev@dev.io"

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV DJANGO_SETTINGS_MODULE=backend.settings
ENV SECRET_KEY='69tgugtg%^fgJO&*&'
ENV DB_NAME=postgres
ENV DB_USER=postgres
ENV DB_HOST=localhost
ENV DB_PASSWORD=postgres
ENV FIELD_ENCRYPTION_KEY='cxvoIIUnDvcCE9IkjaS_l3pvUUjngSK0eRubxEBwkRs='

RUN apt-get update \
    && apt-get -y install libpq-dev gcc \
    && apt-get -y install git

RUN git clone https://github.com/Monta-Application/Monta.git

WORKDIR /Monta

RUN pip install --no-cache-dir -r requirements.txt

EXPOSE 8080

CMD ["python", "manage.py", "migrate"]
