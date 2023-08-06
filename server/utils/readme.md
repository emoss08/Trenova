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
        "order",
        "equipment",
        "primary_worker",
    )
    search_fields = ("ref_num",)
```

### GenericStackedInline <a name="GenericStackedInline"></a>

* Reference utils/admin.py for overriding specific methods

#### Example Usage

```python
class OrderComment(GenericStackedInline[models.OrderComment, models.Order]):
    """
    Order comment inline
    """

    model: type[models.OrderComment] = models.OrderComment
```

### GenericTabularInline <a name="GenericTabularInline"></a>

* Reference utils/admin.py for overriding specific methods

#### Example Usage

```python
class OrderDocumentationInline(GenericTabularInline[models.OrderDocumentation, models.Order]):
    """
    Order documentation inline
    """

    model: type[models.OrderDocumentation] = models.OrderDocumentation

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

## Utils

### FormatDocstring <a name="FormatDocstring"></a>

* This is a utility that will format the docstrings in a file to be in the correct format,
It is not perfect, but will wrap lines longer than 100 characters

```bash
python format_docstrings.py <path_to_directory> <file_name>
```
