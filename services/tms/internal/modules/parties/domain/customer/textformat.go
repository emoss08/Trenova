package customer

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (c *Customer) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	c.Code = format.Code(c.Code)
	c.Name = format.Name(c.Name)
	c.City = format.City(c.City)
	c.StatusUpdateRecipients = format.Email(c.StatusUpdateRecipients)

	if profile := c.EmailProfile; profile != nil {
		profile.ToRecipients = format.Email(profile.ToRecipients)
		profile.CCRecipients = format.Email(profile.CCRecipients)
		profile.BCCRecipients = format.Email(profile.BCCRecipients)
	}
}
