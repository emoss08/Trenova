r = trenova.gather(*[trenova.customer.get.defer(id=str(i)) for i in range(5)], return_exceptions=True)
print(len(r))
