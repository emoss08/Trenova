# Generated by Django 4.2 on 2023-04-20 04:18

from django.db import migrations, models
import django.db.models.deletion


class Migration(migrations.Migration):

    dependencies = [
        ('location', '0002_alter_location_code_alter_locationcategory_name_and_more'),
        ('route', '0002_routecontrol_distance_method'),
    ]

    operations = [
        migrations.AlterModelOptions(
            name='route',
            options={'ordering': ('origin_location', 'destination_location'), 'verbose_name': 'Route', 'verbose_name_plural': 'Routes'},
        ),
        migrations.RemoveField(
            model_name='route',
            name='destination',
        ),
        migrations.RemoveField(
            model_name='route',
            name='origin',
        ),
        migrations.AddField(
            model_name='route',
            name='destination_location',
            field=models.ForeignKey(default=1, help_text='Destination of the route', on_delete=django.db.models.deletion.CASCADE, related_name='destination_location', to='location.location', verbose_name='Destination Location'),
            preserve_default=False,
        ),
        migrations.AddField(
            model_name='route',
            name='origin_location',
            field=models.ForeignKey(default=1, help_text='Origin of the route', on_delete=django.db.models.deletion.CASCADE, related_name='origin_location', to='location.location', verbose_name='Origin Location'),
            preserve_default=False,
        ),
    ]