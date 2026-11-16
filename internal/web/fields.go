package web

// row is one editable landing field.
type row struct {
	key   string
	label string
	value string
	long  bool
}

func landingRows(home Home) []row {
	rows := []row{
		plain("brand_full", "Brand", home.BrandFull, false),
		plain("brand_short", "Short brand", home.BrandShort, false),
		plain("meta_title", "Meta title", home.MetaTitle, false),
		plain("meta_description", "Meta description", home.MetaDescription, true),
		plain("hero", "Hero", home.Hero, true),
		plain("lede", "Lede", home.Lede, true),
		plain("praise", "Praise", home.Praise, true),
		plain("about_heading", "About heading", home.AboutHeading, false),
		plain("about_text", "About", home.AboutText, true),
		plain("cta_title", "CTA title", home.CTATitle, false),
		plain("cta_lede", "CTA lede", home.CTALede, true),
		plain("cta_fine", "CTA fine", home.CTAFine, false),
		plain("footer_name", "Footer", home.FooterName, false),
		plain("resource.heading", "Resource heading", home.Resource.Heading, false),
		plain("resource.title", "Resource title", home.Resource.Title, false),
		plain("resource.sub", "Resource sub", home.Resource.Sub, false),
		plain("resource.cta", "Resource button", home.Resource.CTA, false),
		plain("resource.href", "Resource link", home.Resource.Href, false),
	}
	for i := 0; i < 6; i++ {
		quote := Quote{}
		if i < len(home.Quotes) {
			quote = home.Quotes[i]
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
		if i < len(home.Socials) {
			social = home.Socials[i]
		}
		n := string(rune('1' + i))
		rows = append(rows,
			plain("social."+itoa(i)+".label", "Social "+n, social.Label, false),
			plain("social."+itoa(i)+".url", "Social "+n+" url", social.URL, false),
		)
	}
	rows = append(rows,
		plain("portrait", "Portrait", home.Portrait, false),
		plain("cover", "Cover", home.Cover, false),
	)
	return rows
}

func plain(key, label, value string, long bool) row {
	return row{key: key, label: label, value: value, long: long}
}

func itoa(n int) string {
	return string(rune('0' + n))
}

func setField(home *Home, key, value string) {
	if setPlain(home, key, value) || setResource(home, key, value) {
		return
	}
	setIndexed(home, key, value)
}

func setPlain(home *Home, key, value string) bool {
	switch key {
	case "brand_full":
		home.BrandFull = value
	case "brand_short":
		home.BrandShort = value
	case "meta_title":
		home.MetaTitle = value
	case "meta_description":
		home.MetaDescription = value
	case "hero":
		home.Hero = value
	case "lede":
		home.Lede = value
	case "praise":
		home.Praise = value
	case "about_heading":
		home.AboutHeading = value
	case "about_text":
		home.AboutText = value
	case "cta_title":
		home.CTATitle = value
	case "cta_lede":
		home.CTALede = value
	case "cta_fine":
		home.CTAFine = value
	case "footer_name":
		home.FooterName = value
	case "portrait":
		home.Portrait = value
	case "cover":
		home.Cover = value
	default:
		return false
	}
	return true
}

func setResource(home *Home, key, value string) bool {
	switch key {
	case "resource.heading":
		home.Resource.Heading = value
	case "resource.title":
		home.Resource.Title = value
	case "resource.sub":
		home.Resource.Sub = value
	case "resource.cta":
		home.Resource.CTA = value
	case "resource.href":
		home.Resource.Href = value
	default:
		return false
	}
	return true
}

func setIndexed(home *Home, key, value string) {
	parts := splitKey(key)
	if len(parts) != 3 {
		return
	}
	n := int(parts[1][0] - '0')
	if parts[0] == "quote" {
		for len(home.Quotes) <= n {
			home.Quotes = append(home.Quotes, Quote{})
		}
		switch parts[2] {
		case "quote":
			home.Quotes[n].Quote = value
		case "name":
			home.Quotes[n].Name = value
		case "work":
			home.Quotes[n].Work = value
		}
	}
	if parts[0] == "social" {
		for len(home.Socials) <= n {
			home.Socials = append(home.Socials, Social{})
		}
		switch parts[2] {
		case "label":
			home.Socials[n].Label = value
		case "url":
			home.Socials[n].URL = value
		}
	}
}

func splitKey(key string) []string {
	return stringsSplit(key)
}

func stringsSplit(key string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			out = append(out, key[start:i])
			start = i + 1
		}
	}
	return append(out, key[start:])
}
