package emaildesign

const (
	ink        = "#121210"
	headingInk = "#141413"
	muted      = "#55544e"
	hairline   = "#dddbd3"
	paper      = "#f7f6f2"
	white      = "#ffffff"
	amber      = "#ffa31a"
	bandInk    = "#1a1203"
	noticeFill = "#fff6e5"

	sans = "Geist, -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif"
	mono = "'Geist Mono', ui-monospace, 'SF Mono', Menlo, Consolas, monospace"

	cardPadding = 40
	bandPadding = 32
	flowGap     = 16

	markHeight   = 30
	markMaxWidth = 180
	markMaxBytes = 2 << 20
)

const (
	paragraphStyle = "margin:0 0 16px;padding:0;font-size:16px;line-height:1.6;color:" + ink
	linkStyle      = "color:" + ink + ";text-decoration:underline;text-underline-offset:3px"
	kickerFont     = "padding:0;font-family:" + mono +
		";font-size:12px;line-height:1.4;letter-spacing:1px;text-transform:uppercase;color:"
	tableAttrs  = `width="100%" border="0" cellpadding="0" cellspacing="0" role="presentation"`
	detailStyle = "font-size:14px;line-height:22px;font-weight:400;color:" + muted
	labelFont   = "font-family:" + mono +
		";font-size:12px;line-height:26px;letter-spacing:0.4px;text-transform:uppercase;color:" + muted
)
