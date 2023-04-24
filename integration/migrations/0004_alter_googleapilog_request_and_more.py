# Generated by Django 4.2 on 2023-04-20 03:56

from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('integration', '0003_googleapilog'),
    ]

    operations = [
        migrations.AlterField(
            model_name='googleapilog',
            name='request',
            field=models.TextField(blank=True, help_text='Request made to the Google API', verbose_name='Request'),
        ),
        migrations.AlterField(
            model_name='googleapilog',
            name='response',
            field=models.TextField(blank=True, help_text='Response received from the Google API', verbose_name='Response'),
        ),
        migrations.AlterField(
            model_name='googleapilog',
            name='status',
            field=models.CharField(blank=True, help_text='Status of the request', max_length=255, verbose_name='Status'),
        ),
    ]