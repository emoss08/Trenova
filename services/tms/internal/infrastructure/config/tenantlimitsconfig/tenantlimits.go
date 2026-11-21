package tenantlimitsconfig

import (
	"errors"
	"fmt"
	"time"
)

type Switch string

const (
	SwitchAuto = Switch("auto")
	SwitchOn   = Switch("on")
	SwitchOff  = Switch("off")
)

var ErrInvalidSwitch = errors.New("a tenant limit switch is auto, on or off")

func (s Switch) Validate() error {
	switch s {
	case "", SwitchAuto, SwitchOn, SwitchOff:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidSwitch, s)
	}
}

func (s Switch) On(shared bool) bool {
	switch s {
	case SwitchOn:
		return true
	case SwitchOff:
		return false
	case SwitchAuto:
		return shared
	default:
		return shared
	}
}

type Slot struct {
	PerOrganization int           `mapstructure:"perOrganization" validate:"omitempty,min=1,max=1000"`
	Total           int           `mapstructure:"total"           validate:"omitempty,min=1,max=10000"`
	Hold            time.Duration `mapstructure:"hold"`
	Wait            time.Duration `mapstructure:"wait"`
}

type HeavyJobs struct {
	Organizations Switch `mapstructure:"organizations"`
	PDF           Slot   `mapstructure:"pdf"`
	Extraction    Slot   `mapstructure:"extraction"`
	Export        Slot   `mapstructure:"export"`
	Import        Slot   `mapstructure:"import"`
	Preview       Slot   `mapstructure:"preview"`
	Removal       Slot   `mapstructure:"removal"`
	Archive       Slot   `mapstructure:"archive"`
}

type AISlots struct {
	Organizations   Switch `mapstructure:"organizations"`
	PerOrganization int    `mapstructure:"perOrganization" validate:"omitempty,min=1,max=1000"`
}

type Requests struct {
	Organizations     Switch `mapstructure:"organizations"`
	InstancePerMinute int    `mapstructure:"instancePerMinute" validate:"omitempty,min=1"`
	InstanceBurst     int    `mapstructure:"instanceBurst"     validate:"omitempty,min=1"`
}

type Email struct {
	Organizations          Switch `mapstructure:"organizations"`
	PerOrganizationPerHour int    `mapstructure:"perOrganizationPerHour" validate:"omitempty,min=1,max=1000000"`
}

type Config struct {
	Requests  Requests  `mapstructure:"requests"`
	HeavyJobs HeavyJobs `mapstructure:"heavyJobs"`
	AISlots   AISlots   `mapstructure:"aiSlots"`
	Email     Email     `mapstructure:"email"`
}

type Kind string

const (
	KindPDF        = Kind("pdf")
	KindExtraction = Kind("extraction")
	KindExport     = Kind("export")
	KindImport     = Kind("import")
	KindPreview    = Kind("preview")
	KindRemoval    = Kind("removal")
	KindArchive    = Kind("archive")
)

var defaultSlots = map[Kind]Slot{
	KindPDF:        {PerOrganization: 2, Total: 4, Hold: 2 * time.Minute, Wait: 15 * time.Second},
	KindExtraction: {PerOrganization: 1, Total: 1, Hold: 10 * time.Minute, Wait: 15 * time.Second},
	KindExport:     {PerOrganization: 1, Total: 4, Hold: 30 * time.Minute, Wait: 15 * time.Second},
	KindImport:     {PerOrganization: 1, Total: 4, Hold: 5 * time.Minute, Wait: 15 * time.Second},
	KindPreview:    {PerOrganization: 1, Total: 4, Hold: 2 * time.Minute, Wait: 15 * time.Second},
	KindRemoval:    {PerOrganization: 1, Total: 1, Hold: 30 * time.Minute, Wait: 15 * time.Second},
	KindArchive:    {PerOrganization: 1, Total: 2, Hold: 30 * time.Minute, Wait: 15 * time.Second},
}

const (
	defaultInstanceRequestsPerMinute = 30000
	defaultInstanceRequestBurst      = 3000
	defaultAISlotsPerOrganization    = 2
	defaultEmailPerOrganization      = 300
)

func (c *Config) Validate() error {
	for _, s := range []Switch{c.Requests.Organizations, c.HeavyJobs.Organizations, c.AISlots.Organizations, c.Email.Organizations} {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) SlotFor(kind Kind) Slot {
	configured := c.slot(kind)
	base := defaultSlots[kind]
	if configured.PerOrganization > 0 {
		base.PerOrganization = configured.PerOrganization
	}
	if configured.Total > 0 {
		base.Total = configured.Total
	}
	if configured.Hold > 0 {
		base.Hold = configured.Hold
	}
	if configured.Wait > 0 {
		base.Wait = configured.Wait
	}
	return base
}

func (c *Config) slot(kind Kind) Slot {
	switch kind {
	case KindPDF:
		return c.HeavyJobs.PDF
	case KindExtraction:
		return c.HeavyJobs.Extraction
	case KindExport:
		return c.HeavyJobs.Export
	case KindImport:
		return c.HeavyJobs.Import
	case KindPreview:
		return c.HeavyJobs.Preview
	case KindRemoval:
		return c.HeavyJobs.Removal
	case KindArchive:
		return c.HeavyJobs.Archive
	default:
		return Slot{}
	}
}

func (c *Config) InstanceRequests() (rate, burst int) {
	rate, burst = defaultInstanceRequestsPerMinute, defaultInstanceRequestBurst
	if c.Requests.InstancePerMinute > 0 {
		rate = c.Requests.InstancePerMinute
	}
	if c.Requests.InstanceBurst > 0 {
		burst = c.Requests.InstanceBurst
	}
	return rate, burst
}

func (c *Config) AISlotsPerOrganization() int {
	if c.AISlots.PerOrganization > 0 {
		return c.AISlots.PerOrganization
	}
	return defaultAISlotsPerOrganization
}

func (c *Config) OrganizationEmailPerHour(shared bool) int {
	if c == nil || !c.Email.Organizations.On(shared) {
		return 0
	}
	if c.Email.PerOrganizationPerHour > 0 {
		return c.Email.PerOrganizationPerHour
	}
	return defaultEmailPerOrganization
}
