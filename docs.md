# Monta Documentation

## Django Admin Commands

### Setup Celery Beat

```bash
py manage.py setupcelerybeat
```

#### Description

`setupcelerybeat` command creates the necessary interval schedules for period tasks.

### Install Plugin

```bash
py manage.py install_plugin
```

#### Description

`install_plugin` command installs a plugin from a zip file.

### PSQL Listener

```bash
py manage.py psql_listener
```

#### Description

`psql_listener` command listens to PostgreSQL notifications and executes the corresponding tasks.

### Wait For DB

```bash
py manage.py wait_for_db
```

#### Description

`wait_for_db` command waits for the database to be available.

### Create Test Users

```bash
py manage.py createtestusers
```

#### Description

`createtestusers` command creates test users.

### Create System User

```bash
py manage.py createsystemuser
```

#### Description

`createsystemuser` command creates a system user.
