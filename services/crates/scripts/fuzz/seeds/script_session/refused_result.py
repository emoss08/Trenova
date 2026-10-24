try:
    trenova.shipment.list()
except trenova.Refused as e:
    print(e)
trenova.result({"a": [1, 2]})
