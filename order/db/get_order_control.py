from order.models import OrderControl


def get_order_control(organization):
    o_control: OrderControl = OrderControl.objects.get(organization=organization)
    return o_control
