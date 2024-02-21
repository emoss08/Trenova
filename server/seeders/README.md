# Managing Django Data: Load and Dump Instructions

## Loading Data into Django

To import data into your Django application, execute the following command:

```bash
python manage.py loaddata ./seeders/feature_flags.json
```
**Important:** Substitute `feature_flags.json` with the specific file you intend to load.

## Exporting Data from Django

For exporting data from your Django application, use the command below:

```bash
python manage.py dumpdata organization.featureflag --indent 2 > feature_flags.json
```
**Guidance:** Ensure to replace `organization.featureflag` with the appropriate application and model names from which you wish to export data.

### Advisory:

It is crucial to avoid committing dump files to the version control repository. Seeder files are committed for the purpose of database seeding, whereas dump files, if committed, should solely be used for database loading operations.