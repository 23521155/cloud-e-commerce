// Package email writes the emails the user service sends, as plain text and
// HTML. The sendEmail Azure Function only delivers them.
package email

import (
	"bytes"
	"html/template"
	"strings"
	textTemplate "text/template"

	"github.com/fizzisme/user-service/internal/outbox"
)

// shopName signs every email.
const shopName = "Marginalleya"

// link is the data of every template: who it is for and the one link to
// follow.
type link struct {
	Name string
	URL  string
	Shop string
}

var verifyText = textTemplate.Must(textTemplate.New("verify").Parse(`Hello {{.Name}},

Please confirm that this is your email address for your {{.Shop}} account:

{{.URL}}

The link works for 24 hours. If you did not create an account, ignore this email.

{{.Shop}}
`))

var verifyHTML = template.Must(template.New("verify").Parse(`<p>Hello {{.Name}},</p>
<p>Please confirm that this is your email address for your {{.Shop}} account:</p>
<p><a href="{{.URL}}">Confirm my email address</a></p>
<p>The link works for 24 hours. If you did not create an account, ignore this email.</p>
<p>{{.Shop}}</p>
`))

var resetText = textTemplate.Must(textTemplate.New("reset").Parse(`Hello {{.Name}},

Someone asked to reset the password of your {{.Shop}} account. To choose a new one, open:

{{.URL}}

The link works for 30 minutes and only once. Choosing a new password signs you out everywhere.
If you did not ask, ignore this email: your password stays as it is.

{{.Shop}}
`))

var resetHTML = template.Must(template.New("reset").Parse(`<p>Hello {{.Name}},</p>
<p>Someone asked to reset the password of your {{.Shop}} account.</p>
<p><a href="{{.URL}}">Choose a new password</a></p>
<p>The link works for 30 minutes and only once. Choosing a new password signs you out everywhere.
If you did not ask, ignore this email: your password stays as it is.</p>
<p>{{.Shop}}</p>
`))

// ResetPassword sends the link that lets the owner of an account choose a
// new password.
func ResetPassword(to, name, url string) (outbox.Email, error) {
	return render(to, "Reset your password", resetText, resetHTML, link{Name: name, URL: url, Shop: shopName})
}

// VerifyEmail asks the owner of an address to confirm it by following url.
func VerifyEmail(to, name, url string) (outbox.Email, error) {
	return render(to, "Confirm your email address", verifyText, verifyHTML, link{Name: name, URL: url, Shop: shopName})
}

func render(to, subject string, text *textTemplate.Template, html *template.Template, data link) (outbox.Email, error) {

	var t, h bytes.Buffer
	if err := text.Execute(&t, data); err != nil {
		return outbox.Email{}, err
	}
	if err := html.Execute(&h, data); err != nil {
		return outbox.Email{}, err
	}

	return outbox.Email{
		To:      to,
		Subject: subject + " — " + shopName,
		Text:    t.String(),
		HTML:    strings.TrimSpace(h.String()),
	}, nil
}
