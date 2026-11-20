package oracle

import (
	"fmt"
	"maps"
	"slices"
)

func (c *guideChecker) narrow(at string, use *entry, base *segmentDef) {
	use.def = base
	if len(use.Elements) == 0 {
		return
	}
	def := *base
	def.Elements = slices.Clone(base.Elements)
	def.byPos = maps.Clone(base.byPos)
	seen := make(map[int]bool, len(use.Elements))
	for i, o := range use.Elements {
		oAt := fmt.Sprintf("%s.elements[%d]", at, i)
		if o == nil {
			c.fail(oAt, "is empty")
			continue
		}
		el := base.byPos[o.Pos]
		if el == nil {
			c.fail(oAt+".pos", "%s lists no element %02d to override", base.id, o.Pos)
			continue
		}
		if seen[o.Pos] {
			c.fail(oAt+".pos", "position %d is overridden twice", o.Pos)
			continue
		}
		seen[o.Pos] = true
		narrowed := c.narrowElement(oAt, el, o)
		def.byPos[o.Pos] = narrowed
		for k := range def.Elements {
			if def.Elements[k] == el {
				def.Elements[k] = narrowed
			}
		}
	}
	use.def = &def
}

func (c *guideChecker) narrowElement(at string, base *elementDef, o *elementOverride) *elementDef {
	el := *base
	c.narrowUse(at, base, &el, o)
	if base.byPos != nil {
		if o.Min != nil || o.Max != nil || len(o.Codes) > 0 || o.Fixed != nil {
			c.fail(at, "%s is a composite: narrow its components in the segment", base.ref)
		}
		return &el
	}
	c.narrowLength(at, base, &el, o)
	c.narrowCodes(at, base, &el, o)
	return &el
}

func (c *guideChecker) narrowUse(at string, base, el *elementDef, o *elementOverride) {
	if o.Usage != "" {
		switch {
		case !validUsage(o.Usage):
			c.fail(at+".usage", "%q is not one of M, O, X, N", o.Usage)
		case base.Usage == "M" && o.Usage != "M":
			c.fail(at+".usage", "%s is mandatory in its segment; a use cannot relax it to %s",
				base.ref, o.Usage)
		default:
			el.Usage = o.Usage
		}
	}
	if o.Repeat != nil {
		if *o.Repeat < 1 || *o.Repeat > base.repeatLimit() {
			c.fail(at+".repeat", "%d is not between 1 and the segment's %d",
				*o.Repeat, base.repeatLimit())
		} else {
			el.Repeat = *o.Repeat
		}
	}
}

func (c *guideChecker) narrowCodes(at string, base, el *elementDef, o *elementOverride) {
	if len(o.Codes) > 0 {
		for i, code := range o.Codes {
			if base.codeSet != nil && !base.codeSet[string(code)] {
				c.fail(fmt.Sprintf("%s.codes[%d]", at, i),
					"%q is not one of %s's codes", code, base.ref)
			}
		}
		el.Codes = o.Codes
	}
	if o.Fixed != nil {
		if base.Fixed != nil && *o.Fixed != *base.Fixed {
			c.fail(at+".fixed", "%s is fixed to %q in its segment", base.ref, *base.Fixed)
		}
		fixed := *o.Fixed
		el.Fixed = &fixed
	}
	c.checkCodes(at, el)
}

func (c *guideChecker) narrowLength(at string, base, el *elementDef, o *elementOverride) {
	if o.Min != nil {
		if *o.Min < base.Min || *o.Min > base.Max {
			c.fail(at+".min", "%d is outside %s's %d..%d", *o.Min, base.ref, base.Min, base.Max)
		} else {
			el.Min = *o.Min
		}
	}
	if o.Max != nil {
		if *o.Max < base.Min || *o.Max > base.Max {
			c.fail(at+".max", "%d is outside %s's %d..%d", *o.Max, base.ref, base.Min, base.Max)
		} else {
			el.Max = *o.Max
		}
	}
	if el.Min > el.Max {
		c.fail(at+".min", "min %d is greater than max %d", el.Min, el.Max)
	}
	if el.dataType.name == "DT" && el.Max != 6 && el.Max != 8 {
		c.fail(at+".max", "a date is 6 or 8 long, not %d", el.Max)
	}
}
