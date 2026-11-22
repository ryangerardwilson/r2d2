package web

import "strings"

// row is one editable landing field.
type row struct {
	key   string
	label string
	value string
	long  bool
}

func landingRows(landing Landing) []row {
	rows := []row{
		plain("brand_full", "Brand", landing.BrandFull, false),
		plain("brand_short", "Short brand", landing.BrandShort, false),
		plain("meta_title", "Meta title", landing.MetaTitle, false),
		plain("meta_description", "Meta description", landing.MetaDescription, true),
		plain("hero", "Hero", landing.Hero, true),
		plain("lede", "Lede", landing.Lede, true),
		plain("praise", "Praise", landing.Praise, true),
		plain("about_heading", "About heading", landing.AboutHeading, false),
		plain("about_text", "About", landing.AboutText, true),
		plain("cta_title", "CTA title", landing.CTATitle, false),
		plain("cta_lede", "CTA lede", landing.CTALede, true),
		plain("cta_fine", "CTA fine", landing.CTAFine, false),
		plain("footer_name", "Footer", landing.FooterName, false),
		plain("resource.heading", "Resource heading", landing.Resource.Heading, false),
		plain("resource.title", "Resource title", landing.Resource.Title, false),
		plain("resource.sub", "Resource sub", landing.Resource.Sub, false),
		plain("resource.cta", "Resource button", landing.Resource.CTA, false),
		plain("resource.href", "Resource link", landing.Resource.Href, false),
	}
	for i := 0; i < 6; i++ {
		quote := Quote{}
		if i < len(landing.Quotes) {
			quote = landing.Quotes[i]
		}
		n := string(rune('1' + i))
		rows = append(rows,
			plain("quote."+itoa(i)+".quote", "Quote "+n, quote.Quote, true),
			plain("quote."+itoa(i)+".name", "Quote "+n+" name", quote.Name, false),
			plain("quote."+itoa(i)+".work", "Quote "+n+" work", quote.Work, false),
		)
	}
	for i := 0; i < 4; i++ {
		social := Social{}
		if i < len(landing.Socials) {
			social = landing.Socials[i]
		}
		n := string(rune('1' + i))
		rows = append(rows,
			plain("social."+itoa(i)+".label", "Social "+n, social.Label, false),
			plain("social."+itoa(i)+".url", "Social "+n+" url", social.URL, false),
		)
	}
	rows = append(rows,
		plain("portrait", "Portrait", landing.Portrait, false),
		plain("cover", "Cover", landing.Cover, false),
	)
	return rows
}

func plain(key, label, value string, long bool) row {
	return row{key: key, label: label, value: value, long: long}
}

func itoa(n int) string {
	return string(rune('0' + n))
}

// withField is landing with one field changed. The quotes and socials are copied,
// so the loaded page stays what the site returned until the save answers.
func withField(landing Landing, key, value string) Landing {
	landing.Quotes = append([]Quote(nil), landing.Quotes...)
	landing.Socials = append([]Social(nil), landing.Socials...)
	setField(&landing, key, value)
	return landing
}

func setField(landing *Landing, key, value string) {
	if setPlain(landing, key, value) || setResource(landing, key, value) {
		return
	}
	setIndexed(landing, key, value)
}

func setPlain(landing *Landing, key, value string) bool {
	switch key {
	case "brand_full":
		landing.BrandFull = value
	case "brand_short":
		landing.BrandShort = value
	case "meta_title":
		landing.MetaTitle = value
	case "meta_description":
		landing.MetaDescription = value
	case "hero":
		landing.Hero = value
	case "lede":
		landing.Lede = value
	case "praise":
		landing.Praise = value
	case "about_heading":
		landing.AboutHeading = value
	case "about_text":
		landing.AboutText = value
	case "cta_title":
		landing.CTATitle = value
	case "cta_lede":
		landing.CTALede = value
	case "cta_fine":
		landing.CTAFine = value
	case "footer_name":
		landing.FooterName = value
	case "portrait":
		landing.Portrait = value
	case "cover":
		landing.Cover = value
	default:
		return false
	}
	return true
}

func setResource(landing *Landing, key, value string) bool {
	switch key {
	case "resource.heading":
		landing.Resource.Heading = value
	case "resource.title":
		landing.Resource.Title = value
	case "resource.sub":
		landing.Resource.Sub = value
	case "resource.cta":
		landing.Resource.CTA = value
	case "resource.href":
		landing.Resource.Href = value
	default:
		return false
	}
	return true
}

func setIndexed(landing *Landing, key, value string) {
	parts := strings.Split(key, ".")
	if len(parts) != 3 {
		return
	}
	n := int(parts[1][0] - '0')
	if parts[0] == "quote" {
		for len(landing.Quotes) <= n {
			landing.Quotes = append(landing.Quotes, Quote{})
		}
		switch parts[2] {
		case "quote":
			landing.Quotes[n].Quote = value
		case "name":
			landing.Quotes[n].Name = value
		case "work":
			landing.Quotes[n].Work = value
		}
	}
	if parts[0] == "social" {
		for len(landing.Socials) <= n {
			landing.Socials = append(landing.Socials, Social{})
		}
		switch parts[2] {
		case "label":
			landing.Socials[n].Label = value
		case "url":
			landing.Socials[n].URL = value
		}
	}
}
