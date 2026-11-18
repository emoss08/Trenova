This file is a copy of everything your Trenova instance held: its database and
its documents. It was made from the instance's backups, so your instance was
not slowed down or changed while it was made.

What is in this file
--------------------

README.txt
    This explanation.

manifest.json
    A short summary for software: which databases and how many documents the
    export holds, the PostgreSQL version they came from, the extensions each
    database uses, and the last moment the database holds (dataAsOf).

database/<name>.dump
    One file for each database, made with pg_dump in its "custom" format. It
    holds every table, every row, and everything the database needs to be
    rebuilt. "trenova" holds your business's data; "temporal" and
    "temporal_visibility" hold the state of Trenova's background work.
    manifest.json lists each.

database/roles.sql
    The database's users and their permissions, without their passwords.

documents/
    Every document your instance stored, such as uploaded files, scanned
    paperwork and photos, in the folders Trenova kept them in.

Checking the file
-----------------

The console shows a SHA-256 checksum for this export. To check the file
arrived whole, run "sha256sum" on Linux, "shasum -a 256" on macOS or
"certutil -hashfile <file> SHA256" on Windows, and compare the result with the
checksum in the console.

Opening the file
----------------

It is a tar archive. It unpacks into the folder you open it in, so make an
empty folder for it first. On Linux and macOS, "tar -xf <file>" in that
folder unpacks it. On Windows, use "tar -xf <file>" in a terminal, or an
archive tool such as 7-Zip.

Restoring the database
----------------------

You need PostgreSQL of the same major version as the one named in
manifest.json, or newer, with the extensions manifest.json lists for each
database installed. Most come with PostgreSQL; PostGIS and pgvector are
installed separately.

Run these as a PostgreSQL superuser, from the folder you unpacked the file
into. First create the database's users: its permissions and row-level
security rules name them, so the databases cannot be restored without them.

    psql --file=database/roles.sql

For a user your server already has, such as "postgres", psql reports
'role "postgres" already exists'; that is expected. The users have no
passwords until you set them. Then, for each database file:

    createdb <name>
    pg_restore --dbname=<name> database/<name>.dump

pg_restore should finish without an error.

To see what a dump holds without restoring it:

    pg_restore --list database/<name>.dump

Keeping it safe
---------------

This file holds all of your business's data. Store it somewhere only the
people who need it can reach, and delete copies you no longer need.
