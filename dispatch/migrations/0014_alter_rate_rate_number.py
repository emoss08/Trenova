# Generated by Django 4.2 on 2023-04-08 01:28

from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('dispatch', '0013_rate_rate_amount_currency_and_more'),
    ]

    operations = [
        migrations.AlterField(
            model_name='rate',
            name='rate_number',
            field=models.CharField(editable=False, help_text='Rate Number for Rate', max_length=6, verbose_name='Rate Number'),
        ),
    ]