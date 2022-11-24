<h3 align="center">Monta Exclusive Utils</h3>

## Table of Contents

- [GenericAdmin](#GenericAdmin)
- [GenericStackedInline](#GenericStackedInline)
- [GenericTabularInline](#GenericTabularInline)


## Django Admin

### GenericAdmin <a name="GenericAdmin"></a>

* Reference utils/admin.py for overriding specific methods
* Do not override the `get_autocomplete_fields` method! Change variable `autocomplete` to `False`
#### Example Usage
```python
@admin.register(movement.Movement)
class MovementAdmin(GenericAdmin[movement.Movement]):
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
class OrderComment(GenericStackedInline[order.OrderComment, order.Order]):
  """
  Order comment inline
  """

  model: type[order.OrderComment] = order.OrderComment
```

### GenericTabularInline <a name="GenericTabularInline"></a>
* Reference utils/admin.py for overriding specific methods
#### Example Usage
```python
class OrderDocumentationInline(GenericTabularInline[order.OrderDocumentation, order.Order]):
  """
  Order documentation inline
  """

  model: type[order.OrderDocumentation] = order.OrderDocumentation

```