<h3 align="center">Monta Exclusive Utils</h3>

## Table of Contents

- Admin
    - [GenericAdmin](#GenericAdmin)
    - [GenericStackedInline](#GenericStackedInline)
    - [GenericTabularInline](#GenericTabularInline)
- Model
    - [GenericModel](#GenericModel)
    - [ChoiceField](#ChoiceField)
- Utils
  - [FormatDocstring](#FormatDocstring)

## Django Admin

### GenericAdmin <a name="GenericAdmin"></a>

* Reference utils/admin.py for overriding specific methods
* Do not override the `get_autocomplete_fields` method! Change variable `autocomplete` to `False`

#### Example Usage

```python
@admin.register(models.Movement)
class MovementAdmin(GenericAdmin[models.Movement]):
    """
    Movement Admin
    """

    list_display = (
        "status",
        "ref_num",
        "shipment",
        "equipment",
        "primary_worker",
    )
    search_fields = ("ref_num",)
```

### GenericStackedInline <a name="GenericStackedInline"></a>

* Reference utils/admin.py for overriding specific methods

#### Example Usage

```python
class ShipmentComment(GenericStackedInline[models.ShipmentComment, models.Shipment]):
    """
    Order comment inline
    """

    model: type[models.ShipmentComment] = models.ShipmentComment
```

### GenericTabularInline <a name="GenericTabularInline"></a>

* Reference utils/admin.py for overriding specific methods

#### Example Usage

```python
class ShipmentDocumentationInline(GenericTabularInline[models.ShipmentDocumentation, models.Shipment]):
    """
    Order documentation inline
    """

    model: type[models.ShipmentDocumentation] = models.ShipmentDocumentation

```

### Generic Model <a name="GenericModel"></a>

* Reference utils/models.py for overriding specific fields

#### Example Usage

```python
class RandomModel(GenericModel):
    """
    Random model
    """

    name = models.CharField(max_length=255)
```

### ChoiceField <a name="ChoiceField"></a>

* Reference utils/models.py for overriding specific methods
    * This is a field that can be used to create a choice field, it will automatically set the max_length
      to the length of the longest choice. If a choice is updated it will automatically set a new max_length.

#### Example Usage

```python
class RandomModel(GenericModel):
    """
    Random model
    """

    name = models.CharField(max_length=255)
    status = ChoiceField(choices=Status.choices, default=Status.ACTIVE)
```