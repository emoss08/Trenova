package carrier

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (c *Carrier) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	c.Code = format.Code(c.Code)
	c.Name = format.Name(c.Name)
	c.DBAName = format.Name(c.DBAName)
	c.RemitToName = format.Name(c.RemitToName)
	c.City = format.City(c.City)
	c.RemitCity = format.City(c.RemitCity)
	c.Email = format.Email(c.Email)

	for _, contact := range c.Contacts {
		if contact == nil {
			continue
		}
		contact.Name = format.Name(contact.Name)
		contact.Email = format.Email(contact.Email)
	}
}
