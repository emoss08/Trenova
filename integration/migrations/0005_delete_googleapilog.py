# Generated by Django 4.2 on 2023-04-20 04:00

from django.db import migrations


class Migration(migrations.Migration):

    dependencies = [
        ('integration', '0004_alter_googleapilog_request_and_more'),
    ]

    operations = [
        migrations.DeleteModel(
            name='GoogleAPILog',
        ),
    ]